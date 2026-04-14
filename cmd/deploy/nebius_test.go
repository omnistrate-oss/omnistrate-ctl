package deploy

import (
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCloudProviderAndRegion_Nebius(t *testing.T) {
	t.Run("defaults_to_first_nebius_region_from_offering", func(t *testing.T) {
		offering := openapiclient.ServiceOffering{
			CloudProviders: []string{"nebius"},
			NebiusRegions:  []string{"eu-north1", "eu-west1"},
		}

		cloudProvider, region, err := resolveCloudProviderAndRegion(offering, "", "")
		require.NoError(t, err)
		assert.Equal(t, "nebius", cloudProvider)
		assert.Equal(t, "eu-north1", region)
	})

	t.Run("infers_nebius_from_region", func(t *testing.T) {
		offering := openapiclient.ServiceOffering{
			CloudProviders: []string{"aws", "nebius"},
			AwsRegions:     []string{"us-east-1"},
			NebiusRegions:  []string{"eu-north1"},
		}

		cloudProvider, region, err := resolveCloudProviderAndRegion(offering, "", "eu-north1")
		require.NoError(t, err)
		assert.Equal(t, "nebius", cloudProvider)
		assert.Equal(t, "eu-north1", region)
	})

	t.Run("rejects_nebius_without_region_metadata_or_explicit_region", func(t *testing.T) {
		offering := openapiclient.ServiceOffering{
			CloudProviders: []string{"nebius"},
		}

		_, _, err := resolveCloudProviderAndRegion(offering, "nebius", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "region is required for cloud provider 'nebius'")
	})

	t.Run("rejects_unsupported_nebius_region", func(t *testing.T) {
		offering := openapiclient.ServiceOffering{
			CloudProviders: []string{"nebius"},
			NebiusRegions:  []string{"eu-north1"},
		}

		_, _, err := resolveCloudProviderAndRegion(offering, "nebius", "eu-west1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "region 'eu-west1' is not supported for cloud provider 'nebius'")
	})
}

func TestHasReadyNebiusBindingForRegion(t *testing.T) {
	account := &openapiclient.DescribeAccountConfigResult{
		Status:         "READY",
		NebiusTenantID: ptr("tenant-1"),
		NebiusBindings: []openapiclient.NebiusAccountBindingResult{
			{Region: "eu-west1", Status: ptr("READY")},
			{Region: "eu-north1", Status: ptr("FAILED")},
		},
	}

	assert.True(t, hasReadyNebiusBindingForRegion(account, "eu-west1"))
	assert.False(t, hasReadyNebiusBindingForRegion(account, "eu-north1"))
	assert.False(t, hasReadyNebiusBindingForRegion(account, "us-east-1"))
}

func TestPromptForCloudCredentials_NebiusUnsupported(t *testing.T) {
	_, err := promptForCloudCredentials("nebius")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nebius account onboarding from deploy is not supported")
}

func ptr[T any](v T) *T {
	return &v
}
