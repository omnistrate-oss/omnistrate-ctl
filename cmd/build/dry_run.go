package build

// The read-only route for `omnistrate-ctl build --dry-run`, per
// dry-run-implementation-specs/04-local-artifacts-and-ctl.md §"CTL execution
// algorithm" and §"Output and logging".
//
// This route is entered from runBuildWithOptions after the token is acquired
// and the spec has been preprocessed, and BEFORE FindOrCreateServiceHierarchy —
// the first business write on the legacy path. From here the only server
// interaction permitted is POST /2022-09-01-00/service/spec/validate. Prepare,
// either normal build endpoint, artifact upload, release, environment creation
// and promotion are all unreachable, and there is no fallback to them for any
// server response, including 404/405.
//
// Errors are returned rather than routed through utils.PrintError, because
// PrintError calls os.Exit(1) and would kill the process before cobra could
// turn an INVALID/INCOMPLETE result into a nonzero exit status.

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const validationArtifactEncoding = dataaccess.ValidationArtifactEncoding

// dryRunInput carries everything the validation route needs. It is assembled in
// runBuildWithOptions from the already-resolved flags and the already-rendered
// specification bytes; nothing here is re-read or re-rendered between the two
// requests.
type dryRunInput struct {
	specType string
	name     string
	fileData []byte
	cwd      string
	output   string

	description                   *string
	serviceLogoURL                *string
	environment                   *string
	environmentType               *string
	release                       bool
	releaseAsPreferred            bool
	releaseVersionName            *string
	forceCreateServicePlanVersion bool
}

// validationSpecTypeFor maps the CLI spec type onto the wire enum accepted by
// ValidateServiceSpecRequest2.specType.
func validationSpecTypeFor(specType string) (string, error) {
	switch specType {
	case ServicePlanSpecType:
		return dataaccess.ValidationSpecTypeServicePlan, nil
	case DockerComposeSpecType:
		return dataaccess.ValidationSpecTypeCompose, nil
	default:
		return "", fmt.Errorf("read-only validation does not support spec type %q", specType)
	}
}

// runDryRunValidation executes the two-request validation flow and prints
// exactly one result. It returns nil only for a VALID outcome.
func runDryRunValidation(ctx context.Context, token string, in dryRunInput) error {
	switch in.output {
	case "text", "table", "json":
	default:
		return fmt.Errorf("unsupported output format: %s", in.output)
	}

	wireSpecType, err := validationSpecTypeFor(in.specType)
	if err != nil {
		return err
	}

	limits := defaultValidationLimits()
	if int64(len(in.fileData)) > limits.MaxTotalSpecBytes {
		return fmt.Errorf("%w: the specification is %d bytes and read-only validation accepts at most %d",
			errArtifactLimitExceeded, len(in.fileData), limits.MaxTotalSpecBytes)
	}

	request := openapiclient.ValidateServiceSpecRequest2{
		Name:                             in.name,
		SpecType:                         wireSpecType,
		FileContent:                      base64.StdEncoding.EncodeToString(in.fileData),
		Description:                      in.description,
		ServiceLogoURL:                   in.serviceLogoURL,
		Environment:                      in.environment,
		EnvironmentType:                  in.environmentType,
		Release:                          utils.ToPtr(in.release),
		ReleaseAsPreferred:               utils.ToPtr(in.releaseAsPreferred),
		ReleaseVersionName:               in.releaseVersionName,
		ForceCreateNewServicePlanVersion: utils.ToPtr(in.forceCreateServicePlanVersion),
	}

	// Compose configs and secrets travel in the envelope, exactly as the normal
	// build sends them. They are rejected by the server for service-plan specs.
	if in.specType == DockerComposeSpecType {
		composeFileData, configs, secrets, composeErr := prepareComposeValidationContent(ctx, in.fileData)
		if composeErr != nil {
			return composeErr
		}
		request.FileContent = base64.StdEncoding.EncodeToString(composeFileData)
		request.Configs = configs
		request.Secrets = secrets
	}

	// Request one: discovery, with no artifacts.
	first, err := sendValidationRequest(ctx, token, request, limits)
	if err != nil {
		return err
	}
	limits = limits.withServerLimits(first.Limits)

	result := first
	var suppliedArtifacts []openapiclient.ValidationArtifactInput

	if shouldSupplyArtifacts(first) {
		allowed := collectDeclaredArtifactPaths(in.specType, in.fileData, limits.MaxArchiveMemberPathBytes)
		suppliedArtifacts, err = packageValidationArtifacts(in.cwd, first.RequiredArtifacts, allowed, limits)
		if err != nil {
			return err
		}

		// Request two: the identical envelope plus the requested content. This
		// is the only retry; there is no loop and no mutation fallback.
		request.Artifacts = suppliedArtifacts
		second, secondErr := sendValidationRequest(ctx, token, request, limits)
		if secondErr != nil {
			return secondErr
		}

		if second.InputDigest != first.InputDigest {
			return fmt.Errorf(
				"the validation server described a different input for the second request (%s) than for the first (%s); aborting instead of trusting the result",
				second.InputDigest, first.InputDigest)
		}
		result = second
	}

	printValidationResult(in.output, result)

	if len(suppliedArtifacts) > 0 && len(result.RequiredArtifacts) > 0 {
		return fmt.Errorf(
			"the validation server asked for more local content after the final request (%s); read-only validation does not retry again",
			strings.Join(requiredArtifactPaths(result), ", "))
	}

	switch result.Status {
	case dataaccess.ValidationStatusValid:
		if err := verifyValidatedArtifacts(first, result, suppliedArtifacts); err != nil {
			return err
		}
		return nil
	case dataaccess.ValidationStatusInvalid:
		return errors.New("validation failed: the specification is not valid")
	case dataaccess.ValidationStatusIncomplete:
		return errors.New("validation incomplete: not every required check could be completed")
	default:
		return fmt.Errorf("validation returned an unknown status %q", result.Status)
	}
}

// shouldSupplyArtifacts implements step 4: package and retry once only when the
// server reported concrete requirements and did not already know the spec is
// invalid.
func shouldSupplyArtifacts(result *openapiclient.ValidateServiceSpecResult) bool {
	if result.Status == dataaccess.ValidationStatusInvalid {
		return false
	}
	return len(result.RequiredArtifacts) > 0
}

// sendValidationRequest performs one request under the effective deadline and
// translates an unsupported-server response into an upgrade message.
func sendValidationRequest(
	ctx context.Context,
	token string,
	request openapiclient.ValidateServiceSpecRequest2,
	limits validationLimits,
) (*openapiclient.ValidateServiceSpecResult, error) {
	requestCtx := ctx
	if limits.RequestDeadlineSeconds > 0 {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, time.Duration(limits.RequestDeadlineSeconds)*time.Second)
		defer cancel()
	}

	result, err := dataaccess.ValidateServiceSpec(requestCtx, token, request)
	if err != nil {
		var validationErr *dataaccess.ValidateServiceSpecError
		if errors.As(err, &validationErr) && validationErr.ServerLacksValidationEndpoint() {
			return nil, fmt.Errorf(
				"this Omnistrate server does not support read-only build validation; it requires an upgrade before 'build --dry-run' can be used. No build, prepare or upload request was made")
		}
		return nil, err
	}
	return result, nil
}

// verifyValidatedArtifacts confirms a VALID result really acknowledges the
// supplied bytes and every use that required them.
func verifyValidatedArtifacts(
	first *openapiclient.ValidateServiceSpecResult,
	final *openapiclient.ValidateServiceSpecResult,
	supplied []openapiclient.ValidationArtifactInput,
) error {
	if len(supplied) == 0 {
		return nil
	}

	validated := make(map[string]openapiclient.ValidatedArtifact, len(final.ValidatedArtifacts))
	for _, artifact := range final.ValidatedArtifacts {
		validated[artifact.LogicalPath] = artifact
	}

	for _, artifact := range supplied {
		acknowledged, ok := validated[artifact.LogicalPath]
		if !ok {
			return fmt.Errorf(
				"validation reported VALID but never acknowledged the content supplied for %q; treating the result as untrustworthy",
				artifact.LogicalPath)
		}
		if acknowledged.Sha256 != artifact.Sha256 {
			return fmt.Errorf(
				"validation acknowledged different content for %q (%s) than was supplied (%s)",
				artifact.LogicalPath, acknowledged.Sha256, artifact.Sha256)
		}
	}

	// Every use that made the content required must appear among the uses whose
	// validators consumed it.
	for _, requirement := range first.RequiredArtifacts {
		canonical, err := canonicalValidationPath(requirement.LogicalPath, 0)
		if err != nil {
			continue
		}
		acknowledged, ok := validated[canonical]
		if !ok {
			continue
		}
		acknowledgedUses := make(map[string]struct{}, len(acknowledged.Uses))
		for _, use := range acknowledged.Uses {
			acknowledgedUses[artifactUseKey(use)] = struct{}{}
		}
		for _, use := range requirement.Uses {
			if _, ok := acknowledgedUses[artifactUseKey(use)]; !ok {
				return fmt.Errorf(
					"validation reported VALID but did not check %q for resource %q (%s)",
					canonical, use.ResourceKey, use.Kind)
			}
		}
	}

	return nil
}

func artifactUseKey(use openapiclient.ValidationArtifactUse) string {
	provider := ""
	if use.Provider != nil {
		provider = *use.Provider
	}
	onPrem := ""
	if use.OnPremPlatform != nil {
		onPrem = *use.OnPremPlatform
	}
	return strings.Join([]string{use.ResourceKey, use.Kind, provider, onPrem, use.Path}, "\x00")
}

func requiredArtifactPaths(result *openapiclient.ValidateServiceSpecResult) []string {
	out := make([]string, 0, len(result.RequiredArtifacts))
	for _, requirement := range result.RequiredArtifacts {
		out = append(out, requirement.LogicalPath)
	}
	return out
}

// ---------------------------------------------------------------------------
// Preprocessing that never writes into the user's source tree
// ---------------------------------------------------------------------------

// renderFileForValidation is the read-only counterpart of RenderFile.
//
// RenderFile writes "<rootDir>/<basename>.tmp" for `docker compose config` and
// removes it only on success, so it destroys a pre-existing file of that name
// and can leave rendered spec content behind in the user's input root. The
// validation route must not touch the source tree at all, so the temporary file
// is created in an owned temp directory outside it and removed unconditionally,
// while --project-directory preserves the original project directory for
// relative env-file, config and secret resolution.
func renderFileForValidation(fileData []byte, rootDir string, file string) ([]byte, error) {
	rendered, err := renderFileReferences(fileData, file, nil, nil)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(string(rendered), "env_file:") {
		return rendered, nil
	}

	return renderEnvFileForValidation(rendered, rootDir, file)
}

// renderEnvFileForValidation mirrors renderEnvFileAndInterpolateVariables in
// build_from_repo.go, including its `$` escaping convention and its numeric-cpus
// requoting, but keeps every byte it writes outside the user's project.
//
// TestDryRunValidationRenderingMatchesRenderFile pins the two implementations to
// the same output so they cannot drift.
func renderEnvFileForValidation(fileData []byte, rootDir string, file string) ([]byte, error) {
	// Replace `$` with `$$` to avoid interpolation. Do not replace for `${...}`
	// since it's used to specify variable interpolations.
	escaped := strings.ReplaceAll(string(fileData), "$", "$$")
	escaped = strings.ReplaceAll(escaped, "$${", "${")
	escaped = strings.ReplaceAll(escaped, "${{ secrets.GitHubPAT }}", "$${{ secrets.GitHubPAT }}")

	tempDir, err := os.MkdirTemp("", "omnistrate-validate-")
	if err != nil {
		return nil, fmt.Errorf("failed to create a temporary directory for validation rendering: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	baseName := "omnistrate-compose.yaml"
	if strings.TrimSpace(file) != "" {
		baseName = filepath.Base(file)
	}
	tempFile := filepath.Join(tempDir, baseName+".tmp")
	if err = os.WriteFile(tempFile, []byte(escaped), 0600); err != nil {
		return nil, fmt.Errorf("failed to write the temporary validation spec: %w", err)
	}

	projectDirectory := rootDir
	if strings.TrimSpace(projectDirectory) == "" {
		projectDirectory = "."
	}

	// --project-directory keeps relative env_file/config/secret references
	// resolving against the user's project even though -f points outside it.
	renderCmd := exec.Command("docker", "compose", "--project-directory", projectDirectory, "-f", tempFile, "config")
	cmdOut := &bytes.Buffer{}
	cmdErr := &bytes.Buffer{}
	renderCmd.Stdout = cmdOut
	renderCmd.Stderr = cmdErr

	if err = renderCmd.Run(); err != nil {
		detail := strings.TrimSpace(cmdErr.String())
		if detail == "" {
			return nil, fmt.Errorf("failed to render the compose spec for validation: %w", err)
		}
		return nil, fmt.Errorf("failed to render the compose spec for validation: %w\n%s", err, detail)
	}

	// docker compose config escapes `$` by doubling it; undo that.
	rendered := strings.ReplaceAll(cmdOut.String(), "$$", "$")

	// Quote numeric cpus values in deploy.resources.
	re := regexp.MustCompile(`(?m)(^\s*cpus:\s*)([0-9.]+)\s*$`)
	rendered = re.ReplaceAllString(rendered, `$1"$2"`)

	return []byte(rendered), nil
}

// prepareComposeValidationContent reproduces the compose preprocessing that
// BuildService performs before sending a real build request: volumes that point
// at local files become configs, the project is re-marshalled if that changed
// it, and config/secret files are read from their declared relative paths and
// base64 encoded.
//
// It is a local, read-only operation. TestDryRunComposeEnvelopeMatchesRealBuild
// asserts byte equality with what the legacy build endpoint receives, so the two
// cannot drift.
func prepareComposeValidationContent(ctx context.Context, fileData []byte) (
	[]byte, *map[string]string, *map[string]string, error) {
	parsedYaml, err := loader.ParseYAML(fileData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse YAML content: %w", err)
	}

	project, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{{Config: parsedYaml}},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid compose: %w", err)
	}

	project, modified, err := convertVolumesToConfigs(project)
	if err != nil {
		return nil, nil, nil, err
	}
	if modified {
		marshalled, marshalErr := project.MarshalYAML()
		if marshalErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to marshal project to YAML: %w", marshalErr)
		}
		fileData = marshalled
	}

	var configs *map[string]string
	if project.Configs != nil {
		configsTemp := make(map[string]string)
		for configName, config := range project.Configs {
			content, readErr := os.ReadFile(filepath.Clean(config.File))
			if readErr != nil {
				return nil, nil, nil, readErr
			}
			configsTemp[configName] = base64.StdEncoding.EncodeToString(content)
		}
		configs = &configsTemp
	}

	var secrets *map[string]string
	if project.Secrets != nil {
		secretsTemp := make(map[string]string)
		for secretName, secret := range project.Secrets {
			content, readErr := os.ReadFile(filepath.Clean(secret.File))
			if readErr != nil {
				return nil, nil, nil, readErr
			}
			secretsTemp[secretName] = base64.StdEncoding.EncodeToString(content)
		}
		secrets = &secretsTemp
	}

	return fileData, configs, secrets, nil
}

// ---------------------------------------------------------------------------
// Output
// ---------------------------------------------------------------------------

type dryRunCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type dryRunDiagnostic struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	ResourceKey string `json:"resourceKey,omitempty"`
}

type dryRunArtifactUse struct {
	ResourceKey    string `json:"resourceKey"`
	Kind           string `json:"kind"`
	Provider       string `json:"provider,omitempty"`
	OnPremPlatform string `json:"onPremPlatform,omitempty"`
	Path           string `json:"path,omitempty"`
}

type dryRunArtifactRequirement struct {
	LogicalPath string              `json:"logicalPath"`
	Uses        []dryRunArtifactUse `json:"uses"`
}

type dryRunValidatedArtifact struct {
	LogicalPath         string              `json:"logicalPath"`
	Sha256              string              `json:"sha256"`
	CompressedSizeBytes int64               `json:"compressedSizeBytes"`
	Uses                []dryRunArtifactUse `json:"uses"`
}

type dryRunExistingTarget struct {
	ServiceID            string `json:"serviceID,omitempty"`
	ServiceEnvironmentID string `json:"serviceEnvironmentID,omitempty"`
	ProductTierID        string `json:"productTierID,omitempty"`
}

// dryRunValidationOutput is the single structured document printed on stdout. It
// mirrors the wire result but always allocates the four required arrays, so a
// consumer never has to distinguish null from [].
//
// It deliberately carries no service plan version, release URL or "built"
// wording: a dry run changes no service configuration and must not look as
// though it did.
type dryRunValidationOutput struct {
	Status             string                      `json:"status"`
	ValidationVersion  string                      `json:"validationVersion"`
	InputDigest        string                      `json:"inputDigest"`
	ObservedAt         string                      `json:"observedAt,omitempty"`
	ExistingTarget     *dryRunExistingTarget       `json:"existingTarget,omitempty"`
	Checks             []dryRunCheck               `json:"checks"`
	Diagnostics        []dryRunDiagnostic          `json:"diagnostics"`
	RequiredArtifacts  []dryRunArtifactRequirement `json:"requiredArtifacts"`
	ValidatedArtifacts []dryRunValidatedArtifact   `json:"validatedArtifacts"`
}

func newDryRunValidationOutput(result *openapiclient.ValidateServiceSpecResult) dryRunValidationOutput {
	out := dryRunValidationOutput{
		Status:             result.Status,
		ValidationVersion:  result.ValidationVersion,
		InputDigest:        result.InputDigest,
		Checks:             make([]dryRunCheck, 0, len(result.Checks)),
		Diagnostics:        make([]dryRunDiagnostic, 0, len(result.Diagnostics)),
		RequiredArtifacts:  make([]dryRunArtifactRequirement, 0, len(result.RequiredArtifacts)),
		ValidatedArtifacts: make([]dryRunValidatedArtifact, 0, len(result.ValidatedArtifacts)),
	}
	if result.ObservedAt != nil {
		out.ObservedAt = result.ObservedAt.UTC().Format(time.RFC3339)
	}
	if result.ExistingTarget != nil {
		target := dryRunExistingTarget{}
		if result.ExistingTarget.ServiceID != nil {
			target.ServiceID = *result.ExistingTarget.ServiceID
		}
		if result.ExistingTarget.ServiceEnvironmentID != nil {
			target.ServiceEnvironmentID = *result.ExistingTarget.ServiceEnvironmentID
		}
		if result.ExistingTarget.ProductTierID != nil {
			target.ProductTierID = *result.ExistingTarget.ProductTierID
		}
		out.ExistingTarget = &target
	}
	for _, check := range result.Checks {
		out.Checks = append(out.Checks, dryRunCheck{Name: check.Name, Status: check.Status})
	}
	for _, diagnostic := range result.Diagnostics {
		entry := dryRunDiagnostic{
			Code:     diagnostic.Code,
			Severity: diagnostic.Severity,
			Message:  diagnostic.Message,
		}
		if diagnostic.Path != nil {
			entry.Path = *diagnostic.Path
		}
		if diagnostic.ResourceKey != nil {
			entry.ResourceKey = *diagnostic.ResourceKey
		}
		out.Diagnostics = append(out.Diagnostics, entry)
	}
	for _, requirement := range result.RequiredArtifacts {
		out.RequiredArtifacts = append(out.RequiredArtifacts, dryRunArtifactRequirement{
			LogicalPath: requirement.LogicalPath,
			Uses:        convertArtifactUses(requirement.Uses),
		})
	}
	for _, artifact := range result.ValidatedArtifacts {
		out.ValidatedArtifacts = append(out.ValidatedArtifacts, dryRunValidatedArtifact{
			LogicalPath:         artifact.LogicalPath,
			Sha256:              artifact.Sha256,
			CompressedSizeBytes: artifact.CompressedSizeBytes,
			Uses:                convertArtifactUses(artifact.Uses),
		})
	}
	return out
}

func convertArtifactUses(uses []openapiclient.ValidationArtifactUse) []dryRunArtifactUse {
	out := make([]dryRunArtifactUse, 0, len(uses))
	for _, use := range uses {
		entry := dryRunArtifactUse{
			ResourceKey: use.ResourceKey,
			Kind:        use.Kind,
			Path:        use.Path,
		}
		if use.Provider != nil {
			entry.Provider = *use.Provider
		}
		if use.OnPremPlatform != nil {
			entry.OnPremPlatform = *use.OnPremPlatform
		}
		out = append(out, entry)
	}
	return out
}

// printValidationResult writes exactly one document. In JSON mode that is a
// single JSON object with no spinner, prompt or success footer around it.
func printValidationResult(output string, result *openapiclient.ValidateServiceSpecResult) {
	payload := newDryRunValidationOutput(result)

	if output == "json" {
		// PrintTextTableJsonOutput only fails on an unsupported format, which is
		// rejected before any request is sent.
		_ = utils.PrintTextTableJsonOutput(output, payload)
		return
	}

	printValidationText(payload)
}

func printValidationText(payload dryRunValidationOutput) {
	switch payload.Status {
	case dataaccess.ValidationStatusValid:
		utils.PrintSuccess("Validation passed; no service configuration changed")
	case dataaccess.ValidationStatusInvalid:
		utils.PrintWarning("Validation failed")
	case dataaccess.ValidationStatusIncomplete:
		utils.PrintWarning("Validation incomplete")
	default:
		utils.PrintWarning("Validation returned an unknown status: " + payload.Status)
	}

	if len(payload.Checks) > 0 {
		fmt.Println()
		fmt.Println("Checks:")
		for _, check := range payload.Checks {
			fmt.Printf("  %-22s %s\n", check.Name, check.Status)
		}
	}

	if len(payload.Diagnostics) > 0 {
		fmt.Println()
		fmt.Println("Diagnostics:")
		for _, diagnostic := range payload.Diagnostics {
			location := diagnostic.Path
			if diagnostic.ResourceKey != "" {
				if location == "" {
					location = "resource " + diagnostic.ResourceKey
				} else {
					location += " (resource " + diagnostic.ResourceKey + ")"
				}
			}
			if location == "" {
				fmt.Printf("  [%s] %s: %s\n", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
				continue
			}
			fmt.Printf("  [%s] %s at %s: %s\n", diagnostic.Severity, diagnostic.Code, location, diagnostic.Message)
		}
	}

	if len(payload.RequiredArtifacts) > 0 {
		fmt.Println()
		fmt.Println("Local content still required:")
		for _, requirement := range payload.RequiredArtifacts {
			fmt.Printf("  %s\n", requirement.LogicalPath)
			for _, use := range requirement.Uses {
				fmt.Printf("      used by resource %s (%s)%s\n", use.ResourceKey, use.Kind, formatUseTarget(use))
			}
		}
	}

	if len(payload.ValidatedArtifacts) > 0 {
		fmt.Println()
		fmt.Println("Local content validated:")
		for _, artifact := range payload.ValidatedArtifacts {
			fmt.Printf("  %s (sha256 %s, %d bytes compressed)\n",
				artifact.LogicalPath, shortDigest(artifact.Sha256), artifact.CompressedSizeBytes)
		}
	}

	fmt.Println()
	switch payload.Status {
	case dataaccess.ValidationStatusValid:
		fmt.Println("No service, environment, plan, version or artifact was created, updated, released or promoted.")
	case dataaccess.ValidationStatusInvalid:
		fmt.Println("Fix the diagnostics above and run the validation again. Nothing was changed.")
	case dataaccess.ValidationStatusIncomplete:
		fmt.Println("Validation could not finish every required check, so the specification is not confirmed valid. Nothing was changed.")
	}
}

func formatUseTarget(use dryRunArtifactUse) string {
	switch {
	case use.Provider != "":
		return " on " + use.Provider
	case use.OnPremPlatform != "":
		return " on " + use.OnPremPlatform
	default:
		return ""
	}
}

func shortDigest(digest string) string {
	if len(digest) <= 12 {
		return digest
	}
	return digest[:12]
}
