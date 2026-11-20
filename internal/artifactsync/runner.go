package artifactsync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	helmChartArtifactType     = "helm-chart"
	helmChartContentMediaType = "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
	maxManifestBytes          = 4 << 20
)

var (
	privateECRRegistryPattern = regexp.MustCompile(`^([0-9]{12})\.dkr\.ecr(?:-fips)?\.([a-z0-9-]+)\.amazonaws\.com(?:\.cn)?$`)
	repositoryNamePattern     = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)
	tagPattern                = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)
	sha256DigestPattern       = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

type Options struct {
	AWSProfile string
	Vetter     string
	Signer     string
}

type HookInput struct {
	ArtifactKey       string `json:"artifactKey"`
	SourceRef         string `json:"sourceRef"`
	DestinationRef    string `json:"destinationRef"`
	DestinationDigest string `json:"destinationDigest"`
	DestinationRegion string `json:"destinationRegion"`
	AWSProfile        string `json:"awsProfile,omitempty"`
}

type ecrAPI interface {
	DescribeRepositories(context.Context, *ecr.DescribeRepositoriesInput, ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	CreateRepository(context.Context, *ecr.CreateRepositoryInput, ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
	BatchDeleteImage(context.Context, *ecr.BatchDeleteImageInput, ...func(*ecr.Options)) (*ecr.BatchDeleteImageOutput, error)
}

type runner struct {
	plan            model.ManagedArtifactManualCopyPlan
	registryID      string
	options         Options
	ecr             ecrAPI
	openSource      func(copySourceRef string) (oras.ReadOnlyGraphTarget, error)
	openDestination func(repositoryName string) (oras.Target, error)
	warnings        io.Writer
}

type stagedArtifact struct {
	artifact       model.ManagedArtifactManualCopyArtifact
	repositoryName string
	destination    oras.Target
	descriptor     ocispec.Descriptor
	stagingTag     string
	published      bool
}

func LoadPlan(reader io.Reader) (model.ManagedArtifactManualCopyPlan, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return model.ManagedArtifactManualCopyPlan{}, errors.Wrap(err, "read manual copy plan")
	}
	var sync model.ManagedArtifactSync
	if err = json.Unmarshal(payload, &sync); err == nil && sync.ManualCopyPlan != nil {
		return *sync.ManualCopyPlan, nil
	}
	var plan model.ManagedArtifactManualCopyPlan
	if err = json.Unmarshal(payload, &plan); err != nil {
		return model.ManagedArtifactManualCopyPlan{}, errors.Wrap(err, "decode manual copy plan")
	}
	return plan, nil
}

// Run copies, vets, signs, and publishes every artifact in the plan. Every
// artifact is staged before any hook runs, and final tags are published only
// after both hooks succeed for every artifact.
func Run(ctx context.Context, plan model.ManagedArtifactManualCopyPlan, options Options) error {
	plan, registryID, err := normalizePlan(plan)
	if err != nil {
		return err
	}
	options.AWSProfile = strings.TrimSpace(options.AWSProfile)
	loadOptions := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(plan.DestinationRegion)}
	if options.AWSProfile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(options.AWSProfile))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return errors.Wrap(err, "load AWS destination credentials")
	}
	ecrClient := ecr.NewFromConfig(cfg)
	destinationAuth, err := destinationAuthClient(ctx, ecrClient, plan.DestinationRegistry)
	if err != nil {
		return err
	}
	sourceAuth := sourceAuthClient()
	r := &runner{
		plan:       plan,
		registryID: registryID,
		options:    options,
		ecr:        ecrClient,
		warnings:   os.Stderr,
		openSource: func(copySourceRef string) (oras.ReadOnlyGraphTarget, error) {
			reference, err := registry.ParseReference(strings.TrimPrefix(copySourceRef, "oci://"))
			if err != nil {
				return nil, errors.Wrap(err, "parse copy source reference")
			}
			source, err := remote.NewRepository(reference.Registry + "/" + reference.Repository)
			if err != nil {
				return nil, errors.Wrap(err, "open source repository")
			}
			source.Client = sourceAuth
			return source, nil
		},
		openDestination: func(repositoryName string) (oras.Target, error) {
			destination, err := remote.NewRepository(plan.DestinationRegistry + "/" + repositoryName)
			if err != nil {
				return nil, errors.Wrap(err, "open destination repository")
			}
			destination.Client = destinationAuth
			return destination, nil
		},
	}
	return r.run(ctx)
}

func ValidatePlan(plan model.ManagedArtifactManualCopyPlan) error {
	_, _, err := normalizePlan(plan)
	return err
}

func normalizePlan(plan model.ManagedArtifactManualCopyPlan) (model.ManagedArtifactManualCopyPlan, string, error) {
	plan.DestinationRegistry = strings.TrimRight(strings.TrimSpace(plan.DestinationRegistry), "/")
	plan.DestinationRegion = strings.TrimSpace(plan.DestinationRegion)
	plan.DestinationRepositoryPrefix = strings.Trim(strings.TrimSpace(plan.DestinationRepositoryPrefix), "/")
	if plan.DestinationRegistry == "" || plan.DestinationRegion == "" || plan.DestinationRepositoryPrefix == "" {
		return plan, "", errors.New("destination registry, region, and repository prefix are required")
	}
	match := privateECRRegistryPattern.FindStringSubmatch(plan.DestinationRegistry)
	if match == nil {
		return plan, "", errors.Errorf("destination registry %q is not a private ECR registry", plan.DestinationRegistry)
	}
	if match[2] != plan.DestinationRegion {
		return plan, "", errors.Errorf("destination registry %q is not in destination region %q", plan.DestinationRegistry, plan.DestinationRegion)
	}
	if !repositoryNamePattern.MatchString(plan.DestinationRepositoryPrefix) {
		return plan, "", errors.Errorf("destination repository prefix %q is invalid", plan.DestinationRepositoryPrefix)
	}
	if len(plan.Artifacts) == 0 {
		return plan, "", errors.New("manual copy plan contains no artifacts")
	}
	artifacts := make([]model.ManagedArtifactManualCopyArtifact, 0, len(plan.Artifacts))
	for _, artifact := range plan.Artifacts {
		artifact.ArtifactKey = strings.TrimSpace(artifact.ArtifactKey)
		artifact.CopySourceRef = strings.TrimSpace(artifact.CopySourceRef)
		artifact.SourceDigest = strings.TrimSpace(artifact.SourceDigest)
		artifact.Version = strings.TrimSpace(artifact.Version)
		artifact.RelativePath = strings.Trim(strings.TrimSpace(artifact.RelativePath), "/")
		if artifact.ArtifactKey == "" || artifact.CopySourceRef == "" || artifact.SourceDigest == "" || artifact.Version == "" {
			return plan, "", errors.Errorf("artifact %q is missing key, source, digest, or version", artifact.ArtifactKey)
		}
		if strings.HasPrefix(artifact.CopySourceRef, "http://") || strings.HasPrefix(artifact.CopySourceRef, "https://") {
			return plan, "", errors.Errorf("artifact %q has a non-OCI source %q; manual copy supports OCI registry sources only", artifact.ArtifactKey, artifact.CopySourceRef)
		}
		if !sha256DigestPattern.MatchString(artifact.SourceDigest) {
			return plan, "", errors.Errorf("artifact %q has an invalid source digest %q", artifact.ArtifactKey, artifact.SourceDigest)
		}
		if !tagPattern.MatchString(artifact.Version) {
			return plan, "", errors.Errorf("artifact %q has an invalid version tag %q", artifact.ArtifactKey, artifact.Version)
		}
		if !repositoryNamePattern.MatchString(artifact.RelativePath) {
			return plan, "", errors.Errorf("artifact %q has an invalid relative path %q", artifact.ArtifactKey, artifact.RelativePath)
		}
		artifacts = append(artifacts, artifact)
	}
	plan.Artifacts = artifacts
	return plan, match[1], nil
}

func (r *runner) run(ctx context.Context) error {
	staged := make([]*stagedArtifact, 0, len(r.plan.Artifacts))
	for _, artifact := range r.plan.Artifacts {
		item, err := r.stage(ctx, artifact)
		if err != nil {
			return errors.Wrapf(err, "stage managed artifact %s", artifact.ArtifactKey)
		}
		staged = append(staged, item)
	}
	for _, hook := range []struct{ name, executable string }{
		{name: "customer vetter", executable: r.options.Vetter},
		{name: "customer signer", executable: r.options.Signer},
	} {
		if strings.TrimSpace(hook.executable) == "" {
			continue
		}
		for _, item := range staged {
			if item.published {
				continue
			}
			if err := executeHook(ctx, hook.executable, hook.name, r.hookInput(item)); err != nil {
				return errors.Wrapf(err, "managed artifact %s", item.artifact.ArtifactKey)
			}
		}
	}
	for _, item := range staged {
		if err := r.publish(ctx, item); err != nil {
			return errors.Wrapf(err, "publish managed artifact %s", item.artifact.ArtifactKey)
		}
	}
	return nil
}

func (r *runner) stage(ctx context.Context, artifact model.ManagedArtifactManualCopyArtifact) (*stagedArtifact, error) {
	repositoryName := r.plan.DestinationRepositoryPrefix + "/" + artifact.RelativePath
	if err := r.ensureRepository(ctx, repositoryName); err != nil {
		return nil, err
	}
	destination, err := r.openDestination(repositoryName)
	if err != nil {
		return nil, err
	}
	item := &stagedArtifact{artifact: artifact, repositoryName: repositoryName, destination: destination, stagingTag: stagingTag(artifact)}
	// Artifacts already published by a previous run are not copied or re-hooked.
	published, found, err := resolveTag(ctx, destination, artifact.Version)
	if err != nil {
		return nil, err
	}
	if found && published.Digest.String() == artifact.SourceDigest {
		item.descriptor = published
		item.published = true
		return item, nil
	}
	descriptor, found, err := resolveTag(ctx, destination, item.stagingTag)
	if err != nil {
		return nil, err
	}
	if !found || descriptor.Digest.String() != artifact.SourceDigest {
		source, err := r.openSource(artifact.CopySourceRef)
		if err != nil {
			return nil, err
		}
		descriptor, err = oras.ExtendedCopy(ctx, source, artifact.SourceDigest, destination, item.stagingTag, oras.DefaultExtendedCopyOptions)
		if err != nil {
			return nil, errors.Wrap(err, "copy artifact to staging tag")
		}
		if descriptor.Digest.String() != artifact.SourceDigest {
			return nil, errors.Errorf("copied digest mismatch: expected %s, got %s", artifact.SourceDigest, descriptor.Digest)
		}
	}
	if artifact.Type == helmChartArtifactType && strings.TrimSpace(artifact.SourceChecksum) != "" {
		if err = verifyHelmChartChecksum(ctx, destination, descriptor, artifact.SourceChecksum); err != nil {
			return nil, err
		}
	}
	item.descriptor = descriptor
	return item, nil
}

func (r *runner) ensureRepository(ctx context.Context, repositoryName string) error {
	_, err := r.ecr.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{RegistryId: aws.String(r.registryID), RepositoryNames: []string{repositoryName}})
	if err == nil {
		return nil
	}
	var notFound *types.RepositoryNotFoundException
	if !errors.As(err, &notFound) {
		return errors.Wrapf(err, "describe destination repository %s", repositoryName)
	}
	_, err = r.ecr.CreateRepository(ctx, &ecr.CreateRepositoryInput{RegistryId: aws.String(r.registryID), RepositoryName: aws.String(repositoryName)})
	var exists *types.RepositoryAlreadyExistsException
	if err != nil && !errors.As(err, &exists) {
		return errors.Wrapf(err, "create destination repository %s", repositoryName)
	}
	return nil
}

func (r *runner) publish(ctx context.Context, item *stagedArtifact) error {
	for _, tag := range []string{immutableAliasTag(item.artifact.Version, item.artifact.SourceDigest), item.artifact.Version} {
		if tag == "" {
			continue
		}
		descriptor, found, err := resolveTag(ctx, item.destination, tag)
		if err != nil {
			return err
		}
		if found && descriptor.Digest == item.descriptor.Digest {
			continue
		}
		if err = item.destination.Tag(ctx, item.descriptor, tag); err != nil {
			return errors.Wrapf(err, "publish tag %s", tag)
		}
	}
	r.removeStagingTag(ctx, item)
	return nil
}

// removeStagingTag untags the staging reference; the image stays reachable
// through its published tags.
func (r *runner) removeStagingTag(ctx context.Context, item *stagedArtifact) {
	output, err := r.ecr.BatchDeleteImage(ctx, &ecr.BatchDeleteImageInput{
		RegistryId:     aws.String(r.registryID),
		RepositoryName: aws.String(item.repositoryName),
		ImageIds:       []types.ImageIdentifier{{ImageTag: aws.String(item.stagingTag)}},
	})
	if err == nil && output != nil {
		for _, failure := range output.Failures {
			if failure.FailureCode != types.ImageFailureCodeImageNotFound && failure.FailureCode != types.ImageFailureCodeImageTagDoesNotMatchDigest {
				err = errors.Errorf("%s: %s", failure.FailureCode, aws.ToString(failure.FailureReason))
				break
			}
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(r.warnings, "warning: failed to remove staging tag %s from %s: %v\n", item.stagingTag, item.repositoryName, err)
	}
}

func (r *runner) hookInput(item *stagedArtifact) HookInput {
	return HookInput{
		ArtifactKey:       item.artifact.ArtifactKey,
		SourceRef:         item.artifact.SourceRef,
		DestinationRef:    r.plan.DestinationRegistry + "/" + item.repositoryName + ":" + item.stagingTag,
		DestinationDigest: item.descriptor.Digest.String(),
		DestinationRegion: r.plan.DestinationRegion,
		AWSProfile:        r.options.AWSProfile,
	}
}

func resolveTag(ctx context.Context, target oras.ReadOnlyTarget, tag string) (ocispec.Descriptor, bool, error) {
	descriptor, err := target.Resolve(ctx, tag)
	if errors.Is(err, errdef.ErrNotFound) {
		return ocispec.Descriptor{}, false, nil
	}
	if err != nil {
		return ocispec.Descriptor{}, false, errors.Wrapf(err, "resolve destination tag %s", tag)
	}
	return descriptor, true, nil
}

func stagingTag(artifact model.ManagedArtifactManualCopyArtifact) string {
	hash := sha256.Sum256([]byte(artifact.ArtifactKey + "\x00" + artifact.SourceDigest))
	return "manual-" + hex.EncodeToString(hash[:12])
}

func verifyHelmChartChecksum(ctx context.Context, target oras.ReadOnlyTarget, manifestDescriptor ocispec.Descriptor, expectedChecksum string) error {
	expectedChecksum = strings.TrimSpace(expectedChecksum)
	if !sha256DigestPattern.MatchString(expectedChecksum) {
		return errors.Errorf("invalid SHA-256 Helm chart checksum %q", expectedChecksum)
	}
	manifestReader, err := target.Fetch(ctx, manifestDescriptor)
	if err != nil {
		return errors.Wrap(err, "fetch Helm chart OCI manifest")
	}
	defer manifestReader.Close()
	var manifest ocispec.Manifest
	if err = json.NewDecoder(io.LimitReader(manifestReader, maxManifestBytes)).Decode(&manifest); err != nil {
		return errors.Wrap(err, "decode Helm chart OCI manifest")
	}
	var chartLayer *ocispec.Descriptor
	for index := range manifest.Layers {
		if manifest.Layers[index].MediaType == helmChartContentMediaType {
			if chartLayer != nil {
				return errors.New("Helm chart OCI manifest contains multiple chart layers")
			}
			chartLayer = &manifest.Layers[index]
		}
	}
	if chartLayer == nil {
		return errors.New("Helm chart OCI manifest is missing its chart layer")
	}
	layerReader, err := target.Fetch(ctx, *chartLayer)
	if err != nil {
		return errors.Wrap(err, "fetch Helm chart OCI layer")
	}
	defer layerReader.Close()
	hasher := sha256.New()
	if _, err = io.Copy(hasher, layerReader); err != nil {
		return errors.Wrap(err, "calculate Helm chart checksum")
	}
	expected, _ := hex.DecodeString(strings.TrimPrefix(expectedChecksum, "sha256:"))
	if actual := hasher.Sum(nil); !bytes.Equal(actual, expected) {
		return errors.Errorf("Helm chart checksum mismatch: expected %s, got sha256:%x", expectedChecksum, actual)
	}
	return nil
}

func sourceAuthClient() *auth.Client {
	client := &auth.Client{Client: retry.DefaultClient, Cache: auth.NewCache(), Credential: credentials.Credential(credentials.NewMemoryStore())}
	if store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{}); err == nil {
		client.Credential = credentials.Credential(store)
	}
	return client
}

func destinationAuthClient(ctx context.Context, client *ecr.Client, registryHost string) (*auth.Client, error) {
	output, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, errors.Wrap(err, "get destination ECR authorization token")
	}
	if len(output.AuthorizationData) == 0 || output.AuthorizationData[0].AuthorizationToken == nil {
		return nil, errors.New("destination ECR returned no authorization token")
	}
	decoded, err := base64.StdEncoding.DecodeString(aws.ToString(output.AuthorizationData[0].AuthorizationToken))
	if err != nil {
		return nil, errors.Wrap(err, "decode destination ECR authorization token")
	}
	username, password, found := strings.Cut(string(decoded), ":")
	if !found || username == "" || password == "" {
		return nil, errors.New("destination ECR returned an invalid authorization token")
	}
	return &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: auth.StaticCredential(registryHost, auth.Credential{Username: username, Password: password}),
	}, nil
}

// executeHook writes hook output to stderr so it never mixes with command output.
func executeHook(ctx context.Context, executable, name string, input HookInput) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return errors.Wrap(err, "encode customer hook input")
	}
	command := exec.CommandContext(ctx, executable) // #nosec G204 -- the executable is an explicit operator-provided hook; no shell is involved.
	command.Stdin = bytes.NewReader(payload)
	command.Stdout = os.Stderr
	command.Stderr = os.Stderr
	command.Env = append(os.Environ(), "AWS_REGION="+input.DestinationRegion, "AWS_DEFAULT_REGION="+input.DestinationRegion)
	if input.AWSProfile != "" {
		command.Env = append(command.Env, "AWS_PROFILE="+input.AWSProfile)
	}
	if err = command.Run(); err != nil {
		return errors.Wrapf(err, "%s failed", name)
	}
	return nil
}

func immutableAliasTag(version, digest string) string {
	digestHex := strings.TrimPrefix(strings.TrimSpace(digest), "sha256:")
	version = strings.TrimSpace(version)
	if version == "" || digestHex == "" {
		return ""
	}
	if len(digestHex) > 16 {
		digestHex = digestHex[:16]
	}
	suffix := "-sha256-" + digestHex
	if len(version) > 128-len(suffix) {
		version = version[:128-len(suffix)]
	}
	return version + suffix
}
