package build

// Local artifact packaging for `omnistrate-ctl build --dry-run`, per
// dry-run-implementation-specs/04-local-artifacts-and-ctl.md §"CTL execution
// algorithm" steps 5-7.
//
// Two properties drive every decision in this file:
//
//  1. The backend is NOT trusted to choose which local files are read. A
//     validation response arrives over the network and names filesystem paths.
//     Before anything is opened, each returned path is checked lexically
//     (absolute/escaping/NUL/backslash/`..`), reconciled against the local
//     source declarations in the user's own specification, and finally resolved
//     with symlink evaluation so a symlink cannot point outside the working
//     directory. A server that asks for `secrets/` or `../../.aws` gets an
//     error, not bytes.
//
//  2. Limits are enforced WHILE writing, never after allocating an unbounded
//     string. The tar stream is written through byte-counting writers that fail
//     the moment a budget is exceeded.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"gopkg.in/yaml.v3"
)

// Client-side defaults for the bounds in 04 §"Bounds". The backend owns the
// authoritative constants; when a response advertises `limits`, the smaller of
// the client and server value wins (see validationLimits.withServerLimits).
const (
	defaultMaxRequestBodyBytes              int64 = 24 << 20 // 24 MiB
	defaultMaxTotalCompressedArtifactBytes  int64 = 16 << 20 // 16 MiB
	defaultMaxTotalExtractedBytes           int64 = 64 << 20 // 64 MiB
	defaultMaxTotalSpecBytes                int64 = 1 << 20  // 1 MiB
	defaultMaxArtifacts                     int64 = 64
	defaultMaxArchiveEntries                int64 = 4096
	defaultMaxArchiveMemberPathBytes        int64 = 1024
	defaultMaxConcurrentContentValidations  int64 = 4
	defaultValidationRequestDeadlineSeconds int64 = 120
)

// validationLimits are the effective limits for one dry-run invocation.
type validationLimits struct {
	MaxRequestBodyBytes             int64
	MaxTotalCompressedArtifactBytes int64
	MaxTotalExtractedBytes          int64
	MaxTotalSpecBytes               int64
	MaxArtifacts                    int64
	MaxArchiveEntries               int64
	MaxArchiveMemberPathBytes       int64
	RequestDeadlineSeconds          int64
}

func defaultValidationLimits() validationLimits {
	return validationLimits{
		MaxRequestBodyBytes:             defaultMaxRequestBodyBytes,
		MaxTotalCompressedArtifactBytes: defaultMaxTotalCompressedArtifactBytes,
		MaxTotalExtractedBytes:          defaultMaxTotalExtractedBytes,
		MaxTotalSpecBytes:               defaultMaxTotalSpecBytes,
		MaxArtifacts:                    defaultMaxArtifacts,
		MaxArchiveEntries:               defaultMaxArchiveEntries,
		MaxArchiveMemberPathBytes:       defaultMaxArchiveMemberPathBytes,
		RequestDeadlineSeconds:          defaultValidationRequestDeadlineSeconds,
	}
}

// withServerLimits keeps the smaller of the client and server value for every
// advertised limit, as 04 §"Bounds" requires. A zero or negative advertised
// value is ignored rather than treated as "no content allowed".
func (l validationLimits) withServerLimits(server *openapiclient.ValidationLimits) validationLimits {
	if server == nil {
		return l
	}
	smaller := func(current, advertised int64) int64 {
		if advertised > 0 && advertised < current {
			return advertised
		}
		return current
	}
	l.MaxRequestBodyBytes = smaller(l.MaxRequestBodyBytes, server.MaxRequestBodyBytes)
	l.MaxTotalCompressedArtifactBytes = smaller(l.MaxTotalCompressedArtifactBytes, server.MaxTotalCompressedArtifactBytes)
	l.MaxTotalExtractedBytes = smaller(l.MaxTotalExtractedBytes, server.MaxTotalExtractedBytes)
	l.MaxTotalSpecBytes = smaller(l.MaxTotalSpecBytes, server.MaxTotalSpecBytes)
	l.MaxArtifacts = smaller(l.MaxArtifacts, server.MaxArtifacts)
	l.MaxArchiveEntries = smaller(l.MaxArchiveEntries, server.MaxArchiveEntries)
	l.MaxArchiveMemberPathBytes = smaller(l.MaxArchiveMemberPathBytes, server.MaxArchiveMemberPathBytes)
	l.RequestDeadlineSeconds = smaller(l.RequestDeadlineSeconds, server.RequestDeadlineSeconds)
	return l
}

// errArtifactLimitExceeded marks any failure caused by one of the transport
// bounds so the caller can report "this input is too large for read-only
// validation" instead of an opaque archive error.
var errArtifactLimitExceeded = errors.New("validation artifact limit exceeded")

// ---------------------------------------------------------------------------
// Canonical validation paths
// ---------------------------------------------------------------------------

// canonicalValidationPath applies the validation-specific checks that must run
// BEFORE the shared normalisation, then normalises.
//
// The backend's NormalizeWorkspaceArtifactPath trims whitespace, removes "./",
// returns "." for the workspace root and strips a leading "/". Stripping a
// leading "/" would silently turn an absolute path into a relative one, so this
// stricter client-side check rejects absolute and volume-qualified input
// outright, along with NUL bytes, backslashes and any ".." component. The
// canonical forms match the backend exactly: "./terraform/network" becomes
// "terraform/network" and "./" becomes ".".
func canonicalValidationPath(raw string, maxBytes int64) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("artifact path is empty")
	}
	if maxBytes > 0 && int64(len(trimmed)) > maxBytes {
		return "", fmt.Errorf("%w: artifact path is longer than %d bytes", errArtifactLimitExceeded, maxBytes)
	}
	if strings.ContainsRune(trimmed, 0) {
		return "", fmt.Errorf("artifact path %q contains a NUL byte", raw)
	}
	if strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("artifact path %q contains a backslash", raw)
	}
	if strings.HasPrefix(trimmed, "/") || filepath.IsAbs(trimmed) || filepath.VolumeName(trimmed) != "" {
		return "", fmt.Errorf("artifact path %q is absolute; only paths inside the working directory are allowed", raw)
	}
	// Component-aware ".." rejection, before any normalisation can collapse it.
	for _, component := range strings.Split(trimmed, "/") {
		if component == ".." {
			return "", fmt.Errorf("artifact path %q escapes the working directory", raw)
		}
	}

	canonical := path.Clean(trimmed)
	if canonical == "." || canonical == "./" {
		return ".", nil
	}
	if canonical == ".." || strings.HasPrefix(canonical, "../") || strings.HasPrefix(canonical, "/") {
		return "", fmt.Errorf("artifact path %q escapes the working directory", raw)
	}
	return canonical, nil
}

// ---------------------------------------------------------------------------
// Local source declaration collector — the allowlist
// ---------------------------------------------------------------------------

// declaredLocalArtifactPaths is a narrow reader over the typed YAML positions
// that can declare a local artifact source. It exists ONLY to build the
// allowlist that constrains which paths a validation response may ask for; the
// backend remains authoritative for semantic and provider discovery.
//
// The field names mirror the backend's own structs
// (servicebuild/common/module/{terraform,helm,kustomize,operator_crd}.go), so a
// path the real build would archive is a path this collector accepts.
type declaredLocalArtifactPaths struct {
	Services []struct {
		TerraformConfigurations *struct {
			ConfigurationPerCloudProvider  map[string]declaredTerraformConfiguration `yaml:"configurationPerCloudProvider"`
			ConfigurationPerOnPremPlatform map[string]declaredTerraformConfiguration `yaml:"configurationPerOnPremPlatform"`
		} `yaml:"terraformConfigurations"`
		HelmChartConfiguration *struct {
			ArtifactsLocalPath   string `yaml:"artifactsLocalPath"`
			ArtifactRelativePath string `yaml:"artifactRelativePath"`
		} `yaml:"helmChartConfiguration"`
		KustomizeConfiguration *struct {
			ArtifactsLocalPath string                   `yaml:"artifactsLocalPath"`
			GitConfiguration   declaredGitConfiguration `yaml:"gitConfiguration"`
		} `yaml:"kustomizeConfiguration"`
		OperatorCRDConfiguration *struct {
			ArtifactsLocalPath string `yaml:"artifactsLocalPath"`
		} `yaml:"operatorCRDConfiguration"`
	} `yaml:"services"`
}

type declaredTerraformConfiguration struct {
	ArtifactsLocalPath   string                   `yaml:"artifactsLocalPath"`
	ArtifactRelativePath string                   `yaml:"artifactRelativePath"`
	GitConfiguration     declaredGitConfiguration `yaml:"gitConfiguration"`
}

type declaredGitConfiguration struct {
	RepositoryURL string `yaml:"repositoryUrl"`
}

// collectDeclaredArtifactPaths returns the canonical local artifact paths the
// specification itself declares, plus the documented Terraform default root.
//
// Compose specifications declare no local artifact sources at all (there is no
// x-omnistrate extension for one), so the allowlist for a compose dry run is
// empty and any requirement a server returns for it is refused.
//
// This function never fails the dry run: an undecodable or self-contradictory
// declaration simply contributes nothing to the allowlist, and the backend
// reports the real problem as INVALID.
func collectDeclaredArtifactPaths(specType string, fileData []byte, maxPathBytes int64) map[string]struct{} {
	allowed := make(map[string]struct{})
	if specType != ServicePlanSpecType {
		return allowed
	}

	var spec declaredLocalArtifactPaths
	if err := yaml.Unmarshal(fileData, &spec); err != nil {
		return allowed
	}

	add := func(raw string) {
		canonical, err := canonicalValidationPath(raw, maxPathBytes)
		if err != nil {
			return
		}
		allowed[canonical] = struct{}{}
	}

	// resolveExclusive mirrors module.TerraformConfiguration.ResolveArtifactPath
	// and module.HelmChartConfiguration.ResolveArtifactPath: both fields map to
	// the same backend field, so setting both is an error and contributes
	// nothing.
	resolveExclusive := func(localPath, relativePath string) (string, bool) {
		if localPath != "" && relativePath != "" {
			return "", false
		}
		if relativePath != "" {
			return relativePath, true
		}
		return localPath, true
	}

	addTerraform := func(config declaredTerraformConfiguration) {
		if config.GitConfiguration.RepositoryURL != "" {
			// Git-backed Terraform has no local requirement.
			return
		}
		effective, ok := resolveExclusive(config.ArtifactsLocalPath, config.ArtifactRelativePath)
		if !ok {
			return
		}
		if effective == "" {
			// Documented default local root when neither Git nor a local path
			// is set (extractArtifactUploadTasks defaults to "./").
			effective = "./"
		}
		add(effective)
	}

	for _, service := range spec.Services {
		if service.TerraformConfigurations != nil {
			for _, config := range service.TerraformConfigurations.ConfigurationPerCloudProvider {
				addTerraform(config)
			}
			for _, config := range service.TerraformConfigurations.ConfigurationPerOnPremPlatform {
				addTerraform(config)
			}
		}
		if service.HelmChartConfiguration != nil {
			if effective, ok := resolveExclusive(
				service.HelmChartConfiguration.ArtifactsLocalPath,
				service.HelmChartConfiguration.ArtifactRelativePath,
			); ok && effective != "" {
				add(effective)
			}
		}
		if service.KustomizeConfiguration != nil &&
			service.KustomizeConfiguration.GitConfiguration.RepositoryURL == "" &&
			service.KustomizeConfiguration.ArtifactsLocalPath != "" {
			add(service.KustomizeConfiguration.ArtifactsLocalPath)
		}
		if service.OperatorCRDConfiguration != nil && service.OperatorCRDConfiguration.ArtifactsLocalPath != "" {
			add(service.OperatorCRDConfiguration.ArtifactsLocalPath)
		}
	}

	return allowed
}

func sortedPaths(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Packaging
// ---------------------------------------------------------------------------

// archiveBudget carries the per-request limits that span all artifacts.
type archiveBudget struct {
	remainingCompressed int64
	remainingExtracted  int64
	remainingEntries    int64
	maxMemberPathBytes  int64
}

func newArchiveBudget(limits validationLimits) *archiveBudget {
	return &archiveBudget{
		remainingCompressed: limits.MaxTotalCompressedArtifactBytes,
		remainingExtracted:  limits.MaxTotalExtractedBytes,
		remainingEntries:    limits.MaxArchiveEntries,
		maxMemberPathBytes:  limits.MaxArchiveMemberPathBytes,
	}
}

// boundedWriter fails as soon as the budget is exhausted, so an oversized
// artifact never materialises fully in memory.
type boundedWriter struct {
	dst       io.Writer
	remaining *int64
	what      string
	limit     int64
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > *w.remaining {
		return 0, fmt.Errorf("%w: %s exceeds %d bytes", errArtifactLimitExceeded, w.what, w.limit)
	}
	n, err := w.dst.Write(p)
	*w.remaining -= int64(n)
	return n, err
}

// packageValidationArtifacts turns the server's requirements into request-scoped
// archives. cwd is the archive base, matching ArchiveArtifactPaths(cwd, ...) in
// the normal build path.
func packageValidationArtifacts(
	cwd string,
	requirements []openapiclient.ArtifactRequirement,
	allowed map[string]struct{},
	limits validationLimits,
) ([]openapiclient.ValidationArtifactInput, error) {
	if len(requirements) == 0 {
		return nil, nil
	}

	resolvedBaseDir, err := resolveArchiveBaseDir(cwd)
	if err != nil {
		return nil, err
	}

	// Deduplicate by canonical path first: the same local bytes are never
	// packaged twice, however many uses reference them.
	canonicalPaths := make([]string, 0, len(requirements))
	seen := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		canonical, pathErr := canonicalValidationPath(requirement.LogicalPath, limits.MaxArchiveMemberPathBytes)
		if pathErr != nil {
			return nil, fmt.Errorf("the server requested local content for an unusable path: %w", pathErr)
		}
		if _, exists := allowed[canonical]; !exists {
			return nil, fmt.Errorf(
				"the server requested local content for %q, which this specification does not declare as a local artifact source (declared: %s); refusing to read it",
				requirement.LogicalPath, describeAllowedPaths(allowed))
		}
		if _, duplicate := seen[canonical]; duplicate {
			continue
		}
		seen[canonical] = struct{}{}
		canonicalPaths = append(canonicalPaths, canonical)
	}

	if limits.MaxArtifacts > 0 && int64(len(canonicalPaths)) > limits.MaxArtifacts {
		return nil, fmt.Errorf("%w: %d artifacts requested, the limit is %d",
			errArtifactLimitExceeded, len(canonicalPaths), limits.MaxArtifacts)
	}

	sort.Strings(canonicalPaths)
	budget := newArchiveBudget(limits)

	artifacts := make([]openapiclient.ValidationArtifactInput, 0, len(canonicalPaths))
	for _, canonical := range canonicalPaths {
		compressed, buildErr := buildValidationArchive(resolvedBaseDir, canonical, budget)
		if buildErr != nil {
			return nil, buildErr
		}

		digest := sha256.Sum256(compressed)
		artifacts = append(artifacts, openapiclient.ValidationArtifactInput{
			LogicalPath:         canonical,
			Encoding:            validationArtifactEncoding,
			ArchiveContent:      base64.StdEncoding.EncodeToString(compressed),
			Sha256:              hex.EncodeToString(digest[:]),
			CompressedSizeBytes: int64(len(compressed)),
		})
	}

	return artifacts, nil
}

func describeAllowedPaths(allowed map[string]struct{}) string {
	if len(allowed) == 0 {
		return "none"
	}
	return strings.Join(sortedPaths(allowed), ", ")
}

// buildValidationArchive resolves one canonical path under the archive base and
// returns the exact compressed bytes to send. resolveArtifactPath performs the
// symlink evaluation and containment check, so a symlinked artifact path that
// points outside the working directory is rejected here.
func buildValidationArchive(resolvedBaseDir, canonical string, budget *archiveBudget) ([]byte, error) {
	resolvedPath, info, err := resolveArtifactPath(resolvedBaseDir, canonical)
	if err != nil {
		return nil, fmt.Errorf("cannot read local artifact %q: %w", canonical, err)
	}

	if !info.IsDir() {
		// Existing .tgz/.tar.gz inputs are sent as-is, never rewrapped.
		if !isGzipTarFile(resolvedPath) {
			return nil, fmt.Errorf("artifact path %q is not a directory or a .tar.gz file", canonical)
		}
		return readExistingArchive(resolvedPath, canonical, budget)
	}

	return createBoundedTarGz(resolvedPath, canonical, budget)
}

// readExistingArchive reads a pre-existing archive within the compressed budget
// and verifies it really is a gzip stream, so a malformed file fails locally
// with a clear message rather than as an opaque server-side rejection.
func readExistingArchive(resolvedPath, canonical string, budget *archiveBudget) ([]byte, error) {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read local artifact %q: %w", canonical, err)
	}
	if info.Size() > budget.remainingCompressed {
		return nil, fmt.Errorf("%w: archive %q is %d bytes and does not fit in the remaining %d byte budget",
			errArtifactLimitExceeded, canonical, info.Size(), budget.remainingCompressed)
	}

	content, err := os.ReadFile(resolvedPath) //nolint:gosec // resolvedPath is constrained to the archive base by resolveArtifactPath
	if err != nil {
		return nil, fmt.Errorf("cannot read local artifact %q: %w", canonical, err)
	}
	if int64(len(content)) != info.Size() {
		return nil, fmt.Errorf("local artifact %q changed while it was being read; aborting rather than validating stale content", canonical)
	}
	budget.remainingCompressed -= int64(len(content))

	gzReader, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("local artifact %q is not a valid gzip archive: %w", canonical, err)
	}
	_ = gzReader.Close()

	return content, nil
}

// createBoundedTarGz reproduces the tar layout of createTarGzBase64 (entry names
// relative to the artifact directory's own root, no implicit exclusions) while
// enforcing the entry, member-path, extracted-byte and compressed-byte budgets
// as the stream is written.
//
// Unlike createTarGzBase64 it refuses symlinks and other non-regular members
// explicitly: the validation extractor rejects them, and silently dropping a
// file the validators need would let an incomplete archive be reported as
// validated content.
func createBoundedTarGz(sourceDir, canonical string, budget *archiveBudget) ([]byte, error) {
	var buf bytes.Buffer
	bounded := &boundedWriter{
		dst:       &buf,
		remaining: &budget.remainingCompressed,
		what:      "the total compressed archive size",
		limit:     budget.remainingCompressed,
	}
	gzWriter := gzip.NewWriter(bounded)
	tarWriter := tar.NewWriter(gzWriter)

	walkErr := filepath.Walk(sourceDir, func(entryPath string, info os.FileInfo, err error) error { //nolint:gosec // sourceDir is validated by resolveArtifactPath before this call
		if err != nil {
			return err
		}

		relPath, relErr := filepath.Rel(sourceDir, entryPath)
		if relErr != nil {
			return fmt.Errorf("failed to get relative path: %w", relErr)
		}
		if relPath == "." {
			return nil
		}

		if budget.remainingEntries <= 0 {
			return fmt.Errorf("%w: too many archive entries", errArtifactLimitExceeded)
		}
		budget.remainingEntries--

		memberName := filepath.ToSlash(relPath)
		if budget.maxMemberPathBytes > 0 && int64(len(memberName)) > budget.maxMemberPathBytes {
			return fmt.Errorf("%w: archive member path %q is longer than %d bytes",
				errArtifactLimitExceeded, memberName, budget.maxMemberPathBytes)
		}

		mode := info.Mode()
		switch {
		case mode.IsDir(), mode.IsRegular():
			// supported
		case mode&os.ModeSymlink != 0:
			return fmt.Errorf(
				"local artifact %q contains the symlink %q; read-only validation cannot transport symlinks, so remove or dereference it rather than validating an archive with the file missing",
				canonical, memberName)
		default:
			return fmt.Errorf(
				"local artifact %q contains %q, which is not a regular file or directory (mode %s) and cannot be validated",
				canonical, memberName, mode)
		}

		header, headerErr := tar.FileInfoHeader(info, "")
		if headerErr != nil {
			return fmt.Errorf("failed to create tar header: %w", headerErr)
		}
		// Same layout as the normal build archive: entries are named relative to
		// the artifact directory's own root.
		header.Name = relPath

		if writeErr := tarWriter.WriteHeader(header); writeErr != nil {
			return fmt.Errorf("failed to write tar header: %w", writeErr)
		}

		if !mode.IsRegular() {
			return nil
		}

		file, openErr := os.Open(filepath.Clean(entryPath))
		if openErr != nil {
			return fmt.Errorf("failed to read %q inside local artifact %q: %w", memberName, canonical, openErr)
		}
		defer file.Close()

		copied, copyErr := io.Copy(&boundedWriter{
			dst:       tarWriter,
			remaining: &budget.remainingExtracted,
			what:      "the total uncompressed artifact size",
			limit:     budget.remainingExtracted,
		}, file)
		if copyErr != nil {
			return fmt.Errorf("failed to read %q inside local artifact %q: %w", memberName, canonical, copyErr)
		}

		// Abort rather than substitute content: if the file changed between the
		// header being written and the bytes being read, the archive no longer
		// describes anything that exists on disk.
		after, statErr := os.Lstat(entryPath)
		if statErr != nil {
			return fmt.Errorf("failed to re-read %q inside local artifact %q: %w", memberName, canonical, statErr)
		}
		if copied != info.Size() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
			return fmt.Errorf(
				"%q inside local artifact %q changed while the archive was being written; aborting rather than validating content that was never on disk",
				memberName, canonical)
		}

		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
