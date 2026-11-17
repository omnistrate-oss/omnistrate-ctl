package artifactsync

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/require"
)

func TestLoadPlanAcceptsManagedSyncEnvelope(t *testing.T) {
	t.Parallel()
	plan, err := LoadPlan(strings.NewReader(`{"status":"MANUAL","manualCopyPlan":{"destinationRegistry":"123.dkr.ecr.us-west-2.amazonaws.com","destinationRegion":"us-west-2","destinationRepositoryPrefix":"artifacts/system","artifacts":[{"artifactKey":"image/agent","version":"v1","copySourceRef":"registry.example.com/agent:v1","sourceDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","relativePath":"images/agent"}]}}`))
	require.NoError(t, err)
	require.Equal(t, "artifacts/system", plan.DestinationRepositoryPrefix)
	require.Len(t, plan.Artifacts, 1)
}

func TestValidatePlanRejectsTraversalAndMissingDigest(t *testing.T) {
	t.Parallel()
	base := model.ManagedArtifactManualCopyPlan{
		DestinationRegistry: "123.dkr.ecr.us-west-2.amazonaws.com", DestinationRegion: "us-west-2", DestinationRepositoryPrefix: "artifacts/system",
		Artifacts: []model.ManagedArtifactManualCopyArtifact{{ArtifactKey: "image/agent", Version: "v1", CopySourceRef: "registry.example.com/agent:v1", SourceDigest: "sha256:abc", RelativePath: "../agent"}},
	}
	require.ErrorContains(t, ValidatePlan(base), "invalid relative path")
	base.Artifacts[0].RelativePath = "images/agent"
	base.Artifacts[0].SourceDigest = ""
	require.ErrorContains(t, ValidatePlan(base), "missing key, source, digest, or version")
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
	require.NoError(t, os.WriteFile(hookPath, []byte(hook), 0o700))
	t.Setenv("HOOK_INPUT_PATH", inputPath)
	t.Setenv("HOOK_ENVIRONMENT_PATH", environmentPath)

	err := executeHook(context.Background(), hookPath, "customer vetter", HookInput{
		ArtifactKey:       "image/agent",
		SourceRef:         "registry.example.com/agent:v1",
		DestinationRef:    "123.dkr.ecr.us-west-2.amazonaws.com/artifacts/agent:manual-stage",
		DestinationDigest: "sha256:" + strings.Repeat("a", 64),
		DestinationRegion: "us-west-2",
		AWSProfile:        "customer-production",
	})
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
