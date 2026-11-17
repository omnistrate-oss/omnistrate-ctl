package artifactsync

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/pkg/errors"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
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

func Run(ctx context.Context, plan model.ManagedArtifactManualCopyPlan, options Options) error {
	if err := ValidatePlan(plan); err != nil {
		return err
	}
	loadOptions := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(plan.DestinationRegion)}
	if strings.TrimSpace(options.AWSProfile) != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(strings.TrimSpace(options.AWSProfile)))
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
	for _, artifact := range plan.Artifacts {
		if err = copyArtifact(ctx, ecrClient, sourceAuth, destinationAuth, plan, artifact, options); err != nil {
			return errors.Wrapf(err, "copy managed artifact %s", artifact.ArtifactKey)
		}
	}
	return nil
}

func ValidatePlan(plan model.ManagedArtifactManualCopyPlan) error {
	if strings.TrimSpace(plan.DestinationRegistry) == "" || strings.TrimSpace(plan.DestinationRegion) == "" || strings.Trim(strings.TrimSpace(plan.DestinationRepositoryPrefix), "/") == "" {
		return errors.New("destination registry, region, and repository prefix are required")
	}
	if len(plan.Artifacts) == 0 {
		return errors.New("manual copy plan contains no artifacts")
	}
	for _, artifact := range plan.Artifacts {
		if strings.TrimSpace(artifact.ArtifactKey) == "" || strings.TrimSpace(artifact.CopySourceRef) == "" || strings.TrimSpace(artifact.SourceDigest) == "" || strings.TrimSpace(artifact.Version) == "" {
			return errors.Errorf("artifact %q is missing key, source, digest, or version", artifact.ArtifactKey)
		}
		relativePath := strings.Trim(strings.TrimSpace(artifact.RelativePath), "/")
		if relativePath == "" || relativePath == "." || strings.HasPrefix(relativePath, "../") || strings.Contains(relativePath, "/../") {
			return errors.Errorf("artifact %q has an invalid relative path", artifact.ArtifactKey)
		}
	}
	return nil
}

func copyArtifact(ctx context.Context, ecrClient *ecr.Client, sourceAuth, destinationAuth *auth.Client, plan model.ManagedArtifactManualCopyPlan, artifact model.ManagedArtifactManualCopyArtifact, options Options) error {
	repositoryName := path.Join(strings.Trim(plan.DestinationRepositoryPrefix, "/"), strings.Trim(artifact.RelativePath, "/"))
	if _, err := ecrClient.CreateRepository(ctx, &ecr.CreateRepositoryInput{RepositoryName: &repositoryName}); err != nil {
		var exists *types.RepositoryAlreadyExistsException
		if !errors.As(err, &exists) {
			return errors.Wrap(err, "create destination repository")
		}
	}
	sourceReference, err := registry.ParseReference(strings.TrimPrefix(artifact.CopySourceRef, "oci://"))
	if err != nil {
		return errors.Wrap(err, "parse copy source reference")
	}
	source, err := remote.NewRepository(sourceReference.Registry + "/" + sourceReference.Repository)
	if err != nil {
		return errors.Wrap(err, "open source repository")
	}
	source.Client = sourceAuth
	destination, err := remote.NewRepository(strings.TrimRight(plan.DestinationRegistry, "/") + "/" + repositoryName)
	if err != nil {
		return errors.Wrap(err, "open destination repository")
	}
	destination.Client = destinationAuth
	stagingHash := sha256.Sum256([]byte(artifact.ArtifactKey + "\x00" + artifact.SourceDigest))
	stagingTag := "manual-" + hex.EncodeToString(stagingHash[:12])
	descriptor, err := oras.ExtendedCopy(ctx, source, artifact.SourceDigest, destination, stagingTag, oras.DefaultExtendedCopyOptions)
	if err != nil {
		return errors.Wrap(err, "copy artifact to staging tag")
	}
	digest := descriptor.Digest.String()
	if digest != artifact.SourceDigest {
		return errors.Errorf("copied digest mismatch: expected %s, got %s", artifact.SourceDigest, digest)
	}
	stagingRef := strings.TrimRight(plan.DestinationRegistry, "/") + "/" + repositoryName + ":" + stagingTag
	hookInput := HookInput{
		ArtifactKey: artifact.ArtifactKey, SourceRef: artifact.SourceRef, DestinationRef: stagingRef,
		DestinationDigest: digest, DestinationRegion: plan.DestinationRegion, AWSProfile: strings.TrimSpace(options.AWSProfile),
	}
	if strings.TrimSpace(options.Vetter) != "" {
		if err = executeHook(ctx, options.Vetter, "customer vetter", hookInput); err != nil {
			return err
		}
	}
	if strings.TrimSpace(options.Signer) != "" {
		if err = executeHook(ctx, options.Signer, "customer signer", hookInput); err != nil {
			return err
		}
	}
	if err = destination.Tag(ctx, descriptor, artifact.Version); err != nil {
		return errors.Wrap(err, "publish canonical version tag")
	}
	if alias := immutableAliasTag(artifact.Version, digest); alias != "" {
		if err = destination.Tag(ctx, descriptor, alias); err != nil {
			return errors.Wrap(err, "publish immutable alias tag")
		}
	}
	return nil
}

func sourceAuthClient() *auth.Client {
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return &auth.Client{Client: http.DefaultClient, Cache: auth.NewCache(), Credential: credentials.Credential(credentials.NewMemoryStore())}
	}
	return &auth.Client{Client: http.DefaultClient, Cache: auth.NewCache(), Credential: credentials.Credential(store)}
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
		Client: http.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(strings.TrimSpace(registryHost), auth.Credential{
			Username: username,
			Password: password,
		}),
	}, nil
}

func executeHook(ctx context.Context, executable, name string, input HookInput) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return errors.Wrap(err, "encode customer hook input")
	}
	command := exec.CommandContext(ctx, executable) // #nosec G204 -- the executable is an explicit operator-provided hook; no shell is involved.
	command.Stdin = strings.NewReader(string(payload))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = os.Environ()
	command.Env = append(command.Env, "AWS_REGION="+input.DestinationRegion, "AWS_DEFAULT_REGION="+input.DestinationRegion)
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
	return fmt.Sprintf("%s%s", version, suffix)
}
