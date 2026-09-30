package dataaccess

import (
	"context"
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
