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
