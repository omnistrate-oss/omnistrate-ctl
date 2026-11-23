package dataaccess

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedArtifactReadOnlyOperations(t *testing.T) {
	testutils.IntegrationTest(t)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	ctx := context.Background()
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)

	policy, err := dataaccess.DescribeManagedArtifactReleasePolicy(ctx, login.JWTToken, "DEV", "aws")
	require.NoError(t, err)
	assert.Equal(t, "DEV", policy.EnvironmentType)
	assert.Equal(t, "aws", policy.CloudProvider)

	releases, err := dataaccess.ListManagedArtifactReleases(ctx, login.JWTToken, dataaccess.ListManagedArtifactReleasesOptions{Limit: 1})
	require.NoError(t, err)
	if len(releases.Releases) > 0 {
		release, describeErr := dataaccess.DescribeManagedArtifactRelease(ctx, login.JWTToken, releases.Releases[0].BundleVersion)
		require.NoError(t, describeErr)
		assert.Equal(t, releases.Releases[0].BundleVersion, release.BundleVersion)
	}

	syncs, err := dataaccess.ListManagedArtifactSyncs(ctx, login.JWTToken, dataaccess.ListManagedArtifactSyncsOptions{Limit: 1})
	require.NoError(t, err)
	if len(syncs.Syncs) > 0 {
		sync, describeErr := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, syncs.Syncs[0].ID)
		require.NoError(t, describeErr)
		assert.Equal(t, syncs.Syncs[0].ID, sync.ID)
	}

	_, err = dataaccess.ListManagedArtifactReleases(ctx, "invalid-token", dataaccess.ListManagedArtifactReleasesOptions{Limit: 1})
	assert.Error(t, err)
}

func TestManagedArtifactPublicECRReadOnlyOperations(t *testing.T) {
	testutils.IntegrationTest(t)

	accountID := os.Getenv("MANAGED_ARTIFACT_PUBLIC_ECR_TEST_ACCOUNT_ID")
	if accountID == "" {
		t.Skip("set MANAGED_ARTIFACT_PUBLIC_ECR_TEST_ACCOUNT_ID to a provider account with public artifact access and an existing publication")
	}
	require.Regexp(t, `^[0-9]{12}$`, accountID)
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	ctx := context.Background()
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)

	options := dataaccess.ListManagedArtifactSyncsOptions{RegistryType: "PUBLIC_ECR", DestinationAccountID: accountID, Limit: 1}
	publications, err := dataaccess.ListManagedArtifactSyncs(ctx, login.JWTToken, options)
	require.NoError(t, err)
	require.NotEmpty(t, publications.Syncs, "the configured account must contain a public publication; an empty list cannot prove filter support")
	publication := publications.Syncs[0]
	require.Equal(t, "PUBLIC_ECR", publication.RegistryType)
	require.True(t, strings.HasPrefix(publication.ID, "sppap-"))
	require.NotNil(t, publication.Destination)
	require.Equal(t, accountID, publication.Destination.AccountID)

	detail, err := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, publication.ID)
	require.NoError(t, err)
	assert.Equal(t, publication.ID, detail.ID)
	assert.Equal(t, "PUBLIC_ECR", detail.RegistryType)
	require.NotNil(t, detail.Destination)
	assert.Equal(t, accountID, detail.Destination.AccountID)
	assert.Equal(t, publication.BundleVersion, detail.BundleVersion)
	assert.Len(t, detail.Artifacts, detail.ArtifactCount)

	if publications.NextPageToken != "" {
		options.NextPageToken = publications.NextPageToken
		page, pageErr := dataaccess.ListManagedArtifactSyncs(ctx, login.JWTToken, options)
		require.NoError(t, pageErr)
		for _, item := range page.Syncs {
			assert.Equal(t, "PUBLIC_ECR", item.RegistryType)
			require.NotNil(t, item.Destination)
			assert.Equal(t, accountID, item.Destination.AccountID)
		}
	}
	options.NextPageToken = ""
	options.BundleVersion, options.Status = publication.BundleVersion, publication.Status
	if publication.Target != nil {
		options.TargetID = publication.Target.ID
	}
	filtered, err := dataaccess.ListManagedArtifactSyncs(ctx, login.JWTToken, options)
	require.NoError(t, err)
	for _, item := range filtered.Syncs {
		assert.Equal(t, "PUBLIC_ECR", item.RegistryType)
		require.NotNil(t, item.Destination)
		assert.Equal(t, accountID, item.Destination.AccountID)
		assert.Equal(t, options.BundleVersion, item.BundleVersion)
		assert.Equal(t, options.Status, item.Status)
		if options.TargetID != "" {
			require.NotNil(t, item.Target)
			assert.Equal(t, options.TargetID, item.Target.ID)
		}
	}
}

func TestManagedArtifactSyncActionsRejectInvalidToken(t *testing.T) {
	testutils.IntegrationTest(t)

	ctx := context.Background()
	for name, call := range map[string]func(context.Context, string, string) (*model.ManagedArtifactSync, error){
		"refresh status": dataaccess.RefreshManagedArtifactSyncStatus,
		"reconcile":      dataaccess.ReconcileManagedArtifactSync,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := call(ctx, "invalid-token", "spabs-invalid-token-check")
			require.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

// MANAGED_ARTIFACT_MANUAL_SYNC_TEST_ID must name a private MANUAL sync dedicated
// to these tests, with an intentionally incomplete destination and automatic
// private publication disabled for its target. Refreshing it then leaves it
// MANUAL, and reconciling it is rejected without starting a workflow.
func TestManagedArtifactManualSyncActions(t *testing.T) {
	testutils.IntegrationTest(t)

	syncID := os.Getenv("MANAGED_ARTIFACT_MANUAL_SYNC_TEST_ID")
	if syncID == "" {
		t.Skip("set MANAGED_ARTIFACT_MANUAL_SYNC_TEST_ID to a dedicated incomplete MANUAL private sync whose target disables automatic publication")
	}
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	ctx := context.Background()
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)

	before, err := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, syncID)
	require.NoError(t, err)
	require.Equal(t, "MANUAL", before.Status, "the fixture must start MANUAL")

	refreshed, err := dataaccess.RefreshManagedArtifactSyncStatus(ctx, login.JWTToken, syncID)
	require.NoError(t, err)
	assert.Equal(t, syncID, refreshed.ID)
	assert.Equal(t, before.BundleVersion, refreshed.BundleVersion)
	assert.Equal(t, "MANUAL", refreshed.Status, "an incomplete destination must remain MANUAL")
	assert.Len(t, refreshed.Artifacts, refreshed.ArtifactCount)

	_, err = dataaccess.ReconcileManagedArtifactSync(ctx, login.JWTToken, syncID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_state")

	after, err := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, syncID)
	require.NoError(t, err)
	assert.Equal(t, "MANUAL", after.Status, "a rejected reconcile must not change the sync")

	ready, err := dataaccess.ListManagedArtifactSyncs(ctx, login.JWTToken, dataaccess.ListManagedArtifactSyncsOptions{Status: "ready", Limit: 1})
	require.NoError(t, err)
	if len(ready.Syncs) == 0 {
		t.Log("no READY private sync is available to verify that refresh-status rejects non-MANUAL syncs")
		return
	}
	readySync := ready.Syncs[0]
	require.Equal(t, "READY", readySync.Status)
	_, err = dataaccess.RefreshManagedArtifactSyncStatus(ctx, login.JWTToken, readySync.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_state")
	unchanged, err := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, readySync.ID)
	require.NoError(t, err)
	assert.Equal(t, "READY", unchanged.Status)
}

// MANAGED_ARTIFACT_RECONCILE_SYNC_TEST_ID must name a private READY sync
// dedicated to this test whose target allows automatic private publication.
// Reconcile republishes it in the background while it remains READY, and a
// failed republish leaves the record unchanged.
func TestManagedArtifactReconcileReadySync(t *testing.T) {
	testutils.IntegrationTest(t)

	syncID := os.Getenv("MANAGED_ARTIFACT_RECONCILE_SYNC_TEST_ID")
	if syncID == "" {
		t.Skip("set MANAGED_ARTIFACT_RECONCILE_SYNC_TEST_ID to a dedicated READY private sync whose target allows automatic publication")
	}
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	ctx := context.Background()
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)

	before, err := dataaccess.DescribeManagedArtifactSync(ctx, login.JWTToken, syncID)
	require.NoError(t, err)
	require.Equal(t, "READY", before.Status, "the fixture must start READY")

	reconciled, err := dataaccess.ReconcileManagedArtifactSync(ctx, login.JWTToken, syncID)
	require.NoError(t, err)
	assert.Equal(t, syncID, reconciled.ID)
	assert.Equal(t, before.BundleVersion, reconciled.BundleVersion)
	assert.Equal(t, "READY", reconciled.Status, "a READY republish must remain READY")
}

func TestManagedArtifactRequiredParameters(t *testing.T) {
	testutils.IntegrationTest(t)

	ctx := context.Background()
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "missing policy environment",
			call: func() error {
				_, err := dataaccess.DescribeManagedArtifactReleasePolicy(ctx, "unused", "", "aws")
				return err
			},
			want: "environment type is required",
		},
		{
			name: "missing policy cloud provider",
			call: func() error {
				_, err := dataaccess.UpdateManagedArtifactReleasePolicy(ctx, "unused", "DEV", "", model.UpdateManagedArtifactReleasePolicyRequest{})
				return err
			},
			want: "cloud provider is required",
		},
		{
			name: "missing release bundle version",
			call: func() error {
				_, err := dataaccess.DescribeManagedArtifactRelease(ctx, "unused", "")
				return err
			},
			want: "bundle version is required",
		},
		{
			name: "missing sync ID",
			call: func() error {
				_, err := dataaccess.DescribeManagedArtifactSync(ctx, "unused", "")
				return err
			},
			want: "sync ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.call(), tt.want)
		})
	}
}
