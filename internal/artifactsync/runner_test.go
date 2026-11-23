package artifactsync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
)

func TestLoadPlanAcceptsManagedSyncEnvelope(t *testing.T) {
	t.Parallel()
	plan, err := LoadPlan(strings.NewReader(`{"status":"MANUAL","manualCopyPlan":{"destinationRegistry":"123.dkr.ecr.us-west-2.amazonaws.com","destinationRegion":"us-west-2","destinationRepositoryPrefix":"artifacts/system","artifacts":[{"artifactKey":"image/agent","version":"v1","copySourceRef":"registry.example.com/agent:v1","sourceDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","relativePath":"images/agent"}]}}`))
	require.NoError(t, err)
	require.Equal(t, "artifacts/system", plan.DestinationRepositoryPrefix)
	require.Len(t, plan.Artifacts, 1)
}

func TestValidatePlanRejectsUnsafePlans(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		mutate func(*model.ManagedArtifactManualCopyPlan)
		err    string
	}{
		{name: "parent path", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.Artifacts[0].RelativePath = ".." }, err: "invalid relative path"},
		{name: "path collapsing to prefix", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.Artifacts[0].RelativePath = "images/.." }, err: "invalid relative path"},
		{name: "nested traversal", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.Artifacts[0].RelativePath = "a/../../x" }, err: "invalid relative path"},
		{name: "missing digest", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.Artifacts[0].SourceDigest = "" }, err: "missing key, source, digest, or version"},
		{name: "malformed digest", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.Artifacts[0].SourceDigest = "sha256:abc" }, err: "invalid source digest"},
		{name: "non-ECR registry", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.DestinationRegistry = "registry.example.com" }, err: "not a private ECR registry"},
		{name: "registry outside region", mutate: func(plan *model.ManagedArtifactManualCopyPlan) { plan.DestinationRegion = "us-east-1" }, err: "not in destination region"},
		{name: "HTTP source", mutate: func(plan *model.ManagedArtifactManualCopyPlan) {
			plan.Artifacts[0].CopySourceRef = "https://charts.example.com/agent-1.0.0.tgz"
		}, err: "non-OCI source"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := testPlan("sha256:" + strings.Repeat("a", 64))
			tt.mutate(&plan)
			require.ErrorContains(t, ValidatePlan(plan), tt.err)
		})
	}
	plan := testPlan("sha256:" + strings.Repeat("a", 64))
	plan.DestinationRegistry += "/"
	require.NoError(t, ValidatePlan(plan))
}

func TestImmutableAliasTag(t *testing.T) {
	t.Parallel()
	require.Equal(t, "v1.2.3-sha256-aaaaaaaaaaaaaaaa", immutableAliasTag("v1.2.3", "sha256:"+strings.Repeat("a", 64)))
}

func TestExecuteHookPassesArtifactAndAWSContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test hook uses a POSIX shell script")
	}
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.json")
	environmentPath := filepath.Join(tempDir, "environment.txt")
	hookPath := filepath.Join(tempDir, "hook")
	hook := "#!/bin/sh\ncat > \"$HOOK_INPUT_PATH\"\nprintf '%s\\n%s\\n%s\\n' \"$AWS_PROFILE\" \"$AWS_REGION\" \"$AWS_DEFAULT_REGION\" > \"$HOOK_ENVIRONMENT_PATH\"\n"
	require.NoError(t, os.WriteFile(hookPath, []byte(hook), 0o700)) // #nosec G306 -- the test hook must be executable by its owner.
	t.Setenv("HOOK_INPUT_PATH", inputPath)
	t.Setenv("HOOK_ENVIRONMENT_PATH", environmentPath)

	err := executeHook(context.Background(), hookPath, "customer vetter", HookInput{
		ArtifactKey:       "image/agent",
		SourceRef:         "registry.example.com/agent:v1",
		DestinationRef:    "123.dkr.ecr.us-west-2.amazonaws.com/artifacts/agent:manual-stage",
		DestinationDigest: "sha256:" + strings.Repeat("a", 64),
		DestinationRegion: "us-west-2",
		AWSProfile:        "customer-production",
	}, nil)
	require.NoError(t, err)
	payload, err := os.ReadFile(inputPath)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"artifactKey":"image/agent",
		"sourceRef":"registry.example.com/agent:v1",
		"destinationRef":"123.dkr.ecr.us-west-2.amazonaws.com/artifacts/agent:manual-stage",
		"destinationDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"destinationRegion":"us-west-2",
		"awsProfile":"customer-production"
	}`, string(payload))
	environment, err := os.ReadFile(environmentPath)
	require.NoError(t, err)
	require.Equal(t, "customer-production\nus-west-2\nus-west-2\n", string(environment))
}

func TestExecuteHookReplacesInheritedAWSCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test hook uses a POSIX shell script")
	}
	tempDir := t.TempDir()
	environmentPath := filepath.Join(tempDir, "environment.txt")
	hookPath := filepath.Join(tempDir, "hook")
	hook := "#!/bin/sh\nprintf '%s|%s|%s|%s|%s|%s\\n' \"$AWS_PROFILE\" \"$AWS_ACCESS_KEY_ID\" \"$AWS_SECRET_ACCESS_KEY\" \"${AWS_SESSION_TOKEN-unset}\" \"${AWS_ROLE_ARN-unset}\" \"${AWS_CREDENTIAL_EXPIRATION-unset}\" > \"$HOOK_ENVIRONMENT_PATH\"\n"
	require.NoError(t, os.WriteFile(hookPath, []byte(hook), 0o700)) // #nosec G306 -- the test hook must be executable by its owner.
	t.Setenv("HOOK_ENVIRONMENT_PATH", environmentPath)
	t.Setenv("AWS_ACCESS_KEY_ID", "inherited-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "inherited-secret")
	t.Setenv("AWS_SESSION_TOKEN", "inherited-token")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::111111111111:role/inherited")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CREDENTIAL_EXPIRATION", "")
	require.NoError(t, os.Unsetenv("AWS_CREDENTIAL_EXPIRATION"))
	for _, tt := range []struct {
		name        string
		credentials *aws.Credentials
		want        string
	}{
		{
			name: "inherited credentials without a selected profile",
			want: "|inherited-key|inherited-secret|inherited-token|arn:aws:iam::111111111111:role/inherited|unset\n",
		},
		{
			name:        "static profile credentials",
			credentials: &aws.Credentials{AccessKeyID: "profile-key", SecretAccessKey: "profile-secret"},
			want:        "customer-production|profile-key|profile-secret|unset|unset|unset\n",
		},
		{
			name: "assumed role profile credentials",
			credentials: &aws.Credentials{
				AccessKeyID: "role-key", SecretAccessKey: "role-secret", SessionToken: "role-token",
				CanExpire: true, Expires: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			},
			want: "customer-production|role-key|role-secret|role-token|unset|2026-01-02T03:04:05Z\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := HookInput{DestinationRegion: "us-west-2"}
			if tt.credentials != nil {
				input.AWSProfile = "customer-production"
			}
			require.NoError(t, executeHook(context.Background(), hookPath, "customer signer", input, tt.credentials))
			environment, err := os.ReadFile(environmentPath)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(environment))
		})
	}
}

func TestRunStagesEveryArtifactBeforeHooksAndPublishesLast(t *testing.T) {
	t.Parallel()
	skipWithoutPOSIXShell(t)
	env := newRunEnvironment(t, "image/a", "image/b")
	env.ecr.existing["artifacts/system/images/a"] = true
	logPath := filepath.Join(t.TempDir(), "hooks.log")
	env.runner.options = Options{Vetter: writeHook(t, "vetter", logPath, ""), Signer: writeHook(t, "signer", logPath, "")}

	require.NoError(t, env.runner.run(context.Background()))

	require.Equal(t, []string{"vetter image/a", "vetter image/b", "signer image/a", "signer image/b"}, readHookLog(t, logPath))
	require.Equal(t, []string{"artifacts/system/images/b"}, env.ecr.created)
	for index, key := range env.keys {
		destination := env.destinations[env.repositories[index]]
		requireTag(t, destination, "v1", env.digests[key])
		requireTag(t, destination, immutableAliasTag("v1", env.digests[key]), env.digests[key])
	}
	require.ElementsMatch(t, []string{stagingTag(env.runner.plan.Artifacts[0]), stagingTag(env.runner.plan.Artifacts[1])}, env.ecr.untagged)
}

func TestRunHookFailurePublishesNoArtifact(t *testing.T) {
	t.Parallel()
	skipWithoutPOSIXShell(t)
	env := newRunEnvironment(t, "image/a", "image/b")
	logPath := filepath.Join(t.TempDir(), "hooks.log")
	env.runner.options = Options{Vetter: writeHook(t, "vetter", logPath, "image/b"), Signer: writeHook(t, "signer", logPath, "")}

	require.ErrorContains(t, env.runner.run(context.Background()), "customer vetter failed")

	require.Equal(t, []string{"vetter image/a", "vetter image/b"}, readHookLog(t, logPath))
	for index, artifact := range env.runner.plan.Artifacts {
		destination := env.destinations[env.repositories[index]]
		requireTag(t, destination, stagingTag(artifact), env.digests[artifact.ArtifactKey])
		_, found, err := resolveTag(context.Background(), destination, "v1")
		require.NoError(t, err)
		require.False(t, found)
	}
	require.Empty(t, env.ecr.untagged)
}

func TestRunSkipsArtifactsAlreadyPublished(t *testing.T) {
	t.Parallel()
	skipWithoutPOSIXShell(t)
	env := newRunEnvironment(t, "image/a")
	require.NoError(t, env.runner.run(context.Background()))

	env.sourceOpens = 0
	env.runner.options = Options{Vetter: writeHook(t, "vetter", filepath.Join(t.TempDir(), "hooks.log"), "image/a")}
	require.NoError(t, env.runner.run(context.Background()))
	require.Zero(t, env.sourceOpens)
}

func TestRunRejectsHelmChartChecksumMismatch(t *testing.T) {
	t.Parallel()
	env := newRunEnvironment(t, "chart/a")
	env.runner.plan.Artifacts[0].Type = helmChartArtifactType
	env.runner.plan.Artifacts[0].SourceChecksum = "sha256:" + strings.Repeat("0", 64)

	require.ErrorContains(t, env.runner.run(context.Background()), "Helm chart checksum mismatch")
	_, found, err := resolveTag(context.Background(), env.destinations[env.repositories[0]], "v1")
	require.NoError(t, err)
	require.False(t, found)
}

type fakeECR struct {
	mu       sync.Mutex
	existing map[string]bool
	created  []string
	untagged []string
}

func (f *fakeECR) DescribeRepositories(_ context.Context, input *ecr.DescribeRepositoriesInput, _ ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if aws.ToString(input.RegistryId) != "123456789012" {
		return nil, fmt.Errorf("unexpected registry %q", aws.ToString(input.RegistryId))
	}
	if !f.existing[input.RepositoryNames[0]] {
		return nil, &types.RepositoryNotFoundException{}
	}
	return &ecr.DescribeRepositoriesOutput{}, nil
}

func (f *fakeECR) CreateRepository(_ context.Context, input *ecr.CreateRepositoryInput, _ ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.existing[aws.ToString(input.RepositoryName)] = true
	f.created = append(f.created, aws.ToString(input.RepositoryName))
	return &ecr.CreateRepositoryOutput{}, nil
}

func (f *fakeECR) BatchDeleteImage(_ context.Context, input *ecr.BatchDeleteImageInput, _ ...func(*ecr.Options)) (*ecr.BatchDeleteImageOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.untagged = append(f.untagged, aws.ToString(input.ImageIds[0].ImageTag))
	return &ecr.BatchDeleteImageOutput{}, nil
}

type runEnvironment struct {
	runner       *runner
	ecr          *fakeECR
	keys         []string
	repositories []string
	digests      map[string]string
	destinations map[string]*memory.Store
	sourceOpens  int
}

func newRunEnvironment(t *testing.T, keys ...string) *runEnvironment {
	t.Helper()
	ctx := context.Background()
	source := memory.New()
	env := &runEnvironment{ecr: &fakeECR{existing: map[string]bool{}}, keys: keys, digests: map[string]string{}, destinations: map[string]*memory.Store{}}
	plan := model.ManagedArtifactManualCopyPlan{DestinationRegistry: "123456789012.dkr.ecr.us-west-2.amazonaws.com", DestinationRegion: "us-west-2", DestinationRepositoryPrefix: "artifacts/system"}
	for _, key := range keys {
		layer := content.NewDescriptorFromBytes("application/vnd.cncf.helm.chart.content.v1.tar+gzip", []byte(key))
		require.NoError(t, source.Push(ctx, layer, strings.NewReader(key)))
		manifest, err := oras.PackManifest(ctx, source, oras.PackManifestVersion1_1, "application/vnd.test", oras.PackManifestOptions{Layers: []ocispec.Descriptor{layer}})
		require.NoError(t, err)
		// The in-memory store resolves tags only, so expose each manifest under its digest.
		require.NoError(t, source.Tag(ctx, manifest, manifest.Digest.String()))
		env.digests[key] = manifest.Digest.String()
		relativePath := "images/" + path.Base(key)
		env.repositories = append(env.repositories, plan.DestinationRepositoryPrefix+"/"+relativePath)
		plan.Artifacts = append(plan.Artifacts, model.ManagedArtifactManualCopyArtifact{
			ArtifactKey: key, Type: "container-image", Version: "v1", SourceRef: "registry.example.com/" + key + ":v1",
			CopySourceRef: "registry.example.com/" + key + ":v1", SourceDigest: manifest.Digest.String(), RelativePath: relativePath,
		})
	}
	env.runner = &runner{
		plan: plan, registryID: "123456789012", ecr: env.ecr, warnings: io.Discard,
		openSource: func(string) (oras.ReadOnlyGraphTarget, error) {
			env.sourceOpens++
			return source, nil
		},
		openDestination: func(repositoryName string) (oras.Target, error) {
			if destination, ok := env.destinations[repositoryName]; ok {
				return destination, nil
			}
			destination := memory.New()
			env.destinations[repositoryName] = destination
			return destination, nil
		},
	}
	return env
}

func testPlan(digest string) model.ManagedArtifactManualCopyPlan {
	return model.ManagedArtifactManualCopyPlan{
		DestinationRegistry: "123456789012.dkr.ecr.us-west-2.amazonaws.com", DestinationRegion: "us-west-2", DestinationRepositoryPrefix: "artifacts/system",
		Artifacts: []model.ManagedArtifactManualCopyArtifact{{ArtifactKey: "image/agent", Version: "v1", CopySourceRef: "registry.example.com/agent:v1", SourceDigest: digest, RelativePath: "images/agent"}},
	}
}

func skipWithoutPOSIXShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test hook uses a POSIX shell script")
	}
}

// writeHook creates a hook that logs "<name> <artifactKey>" and fails for failKey.
func writeHook(t *testing.T, name, logPath, failKey string) string {
	t.Helper()
	hookPath := filepath.Join(t.TempDir(), name)
	script := fmt.Sprintf(`#!/bin/sh
input=$(cat)
key=$(printf '%%s' "$input" | sed -e 's/.*"artifactKey":"\([^"]*\)".*/\1/')
printf '%%s %%s\n' %q "$key" >> %q
[ "$key" != %q ]
`, name, logPath, failKey)
	require.NoError(t, os.WriteFile(hookPath, []byte(script), 0o700)) // #nosec G306 -- the test hook must be executable by its owner.
	return hookPath
}

func readHookLog(t *testing.T, logPath string) []string {
	t.Helper()
	payload, err := os.ReadFile(logPath) // #nosec G304 -- test-owned temporary file.
	require.NoError(t, err)
	return strings.Split(strings.TrimSpace(string(payload)), "\n")
}

func requireTag(t *testing.T, target oras.ReadOnlyTarget, tag, digest string) {
	t.Helper()
	descriptor, found, err := resolveTag(context.Background(), target, tag)
	require.NoError(t, err)
	require.True(t, found, "tag %s is missing", tag)
	require.Equal(t, digest, descriptor.Digest.String())
}
