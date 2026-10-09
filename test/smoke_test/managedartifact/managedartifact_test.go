package managedartifact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedArtifactCommands(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.Background()
	// Cleanup callbacks run in reverse order, so keep authentication until policy restoration finishes.
	t.Cleanup(testutils.Cleanup)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{
		"login",
		fmt.Sprintf("--email=%s", testEmail),
		fmt.Sprintf("--password=%s", testPassword),
	})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	var originalPolicy model.ManagedArtifactReleasePolicy
	runManagedArtifactJSON(t, ctx, &originalPolicy,
		"managed-artifact", "policy", "describe", "--environment-type", "dev", "--cloud-provider", "aws")
	require.Equal(t, "DEV", originalPolicy.EnvironmentType)
	require.Equal(t, "aws", originalPolicy.CloudProvider)
	require.NotEmpty(t, originalPolicy.EffectiveBundleVersion)

	var releases model.ManagedArtifactReleaseList
	runManagedArtifactJSON(t, ctx, &releases,
		"managed-artifact", "release", "list", "--limit", "1")
	require.NotEmpty(t, releases.Releases)

	var release model.ManagedArtifactRelease
	runManagedArtifactJSON(t, ctx, &release,
		"managed-artifact", "release", "describe", "--bundle-version", releases.Releases[0].BundleVersion)
	assert.Equal(t, releases.Releases[0].BundleVersion, release.BundleVersion)
	assert.Len(t, release.Artifacts, release.ArtifactCount)

	var syncs model.ManagedArtifactSyncList
	runManagedArtifactJSON(t, ctx, &syncs,
		"managed-artifact", "sync", "list", "--limit", "1")
	require.NotEmpty(t, syncs.Syncs)

	var sync model.ManagedArtifactSync
	runManagedArtifactJSON(t, ctx, &sync,
		"managed-artifact", "sync", "describe", "--id", syncs.Syncs[0].ID)
	assert.Equal(t, syncs.Syncs[0].ID, sync.ID)
	assert.Equal(t, syncs.Syncs[0].BundleVersion, sync.BundleVersion)
	assert.Len(t, sync.Artifacts, sync.ArtifactCount)

	for _, args := range [][]string{
		{"managed-artifact", "policy", "describe", "--environment-type", "dev", "--cloud-provider", "aws"},
		{"managed-artifact", "release", "list", "--limit", "1"},
		{"managed-artifact", "release", "describe", "--bundle-version", release.BundleVersion},
		{"managed-artifact", "sync", "list", "--limit", "1"},
		{"managed-artifact", "sync", "describe", "--id", sync.ID},
	} {
		cmd.RootCmd.SetArgs(append(args, "--output", "table"))
		require.NoError(t, cmd.RootCmd.ExecuteContext(ctx), "table command failed: %v", args)
	}

	restored := false
	restorePolicy := func() {
		args := []string{
			"managed-artifact", "policy", "update",
			"--environment-type", "dev",
			"--cloud-provider", "aws",
			fmt.Sprintf("--auto-upgrade=%t", originalPolicy.AutoUpgrade),
		}
		if !originalPolicy.AutoUpgrade && originalPolicy.PreferredBundleVersion != "" {
			args = append(args, "--preferred-bundle-version", originalPolicy.PreferredBundleVersion)
		}
		var restoredPolicy model.ManagedArtifactReleasePolicy
		runManagedArtifactJSON(t, ctx, &restoredPolicy, args...)
		require.Equal(t, originalPolicy.AutoUpgrade, restoredPolicy.AutoUpgrade)
		require.Equal(t, originalPolicy.PreferredBundleVersion, restoredPolicy.PreferredBundleVersion)
	}
	t.Cleanup(func() {
		if !restored {
			restorePolicy()
		}
	})

	var pinnedPolicy model.ManagedArtifactReleasePolicy
	runManagedArtifactJSON(t, ctx, &pinnedPolicy,
		"managed-artifact", "policy", "update",
		"--environment-type", "dev",
		"--cloud-provider", "aws",
		"--auto-upgrade=false",
		"--preferred-bundle-version", originalPolicy.EffectiveBundleVersion)
	assert.False(t, pinnedPolicy.AutoUpgrade)
	assert.Equal(t, originalPolicy.EffectiveBundleVersion, pinnedPolicy.PreferredBundleVersion)
	assert.Equal(t, originalPolicy.EffectiveBundleVersion, pinnedPolicy.EffectiveBundleVersion)
	var persistedPinnedPolicy model.ManagedArtifactReleasePolicy
	runManagedArtifactJSON(t, ctx, &persistedPinnedPolicy,
		"managed-artifact", "policy", "describe", "--environment-type", "dev", "--cloud-provider", "aws")
	require.Equal(t, pinnedPolicy, persistedPinnedPolicy)

	restorePolicy()
	var restoredPolicy model.ManagedArtifactReleasePolicy
	runManagedArtifactJSON(t, ctx, &restoredPolicy,
		"managed-artifact", "policy", "describe", "--environment-type", "dev", "--cloud-provider", "aws")
	require.Equal(t, originalPolicy.AutoUpgrade, restoredPolicy.AutoUpgrade)
	require.Equal(t, originalPolicy.PreferredBundleVersion, restoredPolicy.PreferredBundleVersion)
	restored = true
}

func TestManagedArtifactPublicECRCommands(t *testing.T) {
	testutils.SmokeTest(t)

	accountID := os.Getenv("MANAGED_ARTIFACT_PUBLIC_ECR_TEST_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("set MANAGED_ARTIFACT_PUBLIC_ECR_TEST_ACCOUNT_ID to a provider account with public artifact access and an existing publication")
	}
	require.Regexp(t, `^[0-9]{12}$`, accountID)
	t.Cleanup(testutils.Cleanup)
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	ctx := context.Background()
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	listArgs := []string{"managed-artifact", "sync", "list", "--registry-type", "public_ecr", "--destination-account-id", accountID, "--limit", "1"}
	var publications model.ManagedArtifactSyncList
	runManagedArtifactJSON(t, ctx, &publications, listArgs...)
	require.NotEmpty(t, publications.Syncs, "the configured account must contain a public publication; an empty list cannot prove filter support")
	publication := publications.Syncs[0]
	require.Equal(t, "PUBLIC_ECR", publication.RegistryType)
	require.True(t, strings.HasPrefix(publication.ID, "sppap-"))
	require.NotNil(t, publication.Destination)
	require.Equal(t, accountID, publication.Destination.AccountID)

	var detail model.ManagedArtifactSync
	runManagedArtifactJSON(t, ctx, &detail, "managed-artifact", "sync", "describe", "--id", publication.ID)
	assert.Equal(t, publication.ID, detail.ID)
	assert.Equal(t, "PUBLIC_ECR", detail.RegistryType)
	require.NotNil(t, detail.Destination)
	assert.Equal(t, accountID, detail.Destination.AccountID)
	assert.Equal(t, publication.BundleVersion, detail.BundleVersion)
	assert.Len(t, detail.Artifacts, detail.ArtifactCount)

	if publications.NextPageToken != "" {
		var page model.ManagedArtifactSyncList
		runManagedArtifactJSON(t, ctx, &page, append(listArgs, "--next-page-token", publications.NextPageToken)...)
		for _, item := range page.Syncs {
			assert.Equal(t, "PUBLIC_ECR", item.RegistryType)
			require.NotNil(t, item.Destination)
			assert.Equal(t, accountID, item.Destination.AccountID)
		}
	}
	for _, args := range [][]string{listArgs, {"managed-artifact", "sync", "describe", "--id", publication.ID}} {
		utils.LastPrintedString = ""
		cmd.RootCmd.SetArgs(append(args, "--output", "table"))
		require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
		require.NotEmpty(t, utils.LastPrintedString)
	}
}

func runManagedArtifactJSON(t *testing.T, ctx context.Context, result any, args ...string) {
	t.Helper()

	utils.LastPrintedString = ""
	cmd.RootCmd.SetArgs(append(args, "--output", "json"))
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx), "command failed: %v", args)
	require.NotEmpty(t, utils.LastPrintedString)
	require.NoError(t, json.Unmarshal([]byte(utils.LastPrintedString), result))
}
