package deploymentcell

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/require"
)

func TestDeploymentCellTemplateForCloudProviderReturnsErrorForMissingDefaults(t *testing.T) {
	_, err := deploymentCellTemplateForCloudProvider(nil, " BYOC-ONPREM ")
	require.ErrorContains(t, err, "no default deployment cell configurations found in service provider organization")
}

func TestDeploymentCellTemplateForCloudProviderReturnsErrorForMissingBYOCOnPremConfig(t *testing.T) {
	configs := map[string]openapiclient.DeploymentCellConfiguration{
		"aws": {
			Amenities: []openapiclient.Amenity{
				apiAmenity("AWS Amenity"),
			},
		},
	}

	_, err := deploymentCellTemplateForCloudProvider(&configs, "byoc-onprem")
	require.ErrorContains(t, err, "no configuration found for cloud provider 'byoc-onprem'")
}

func TestDeploymentCellTemplateForCloudProviderNormalizesAndSelectsExistingConfig(t *testing.T) {
	configs := map[string]openapiclient.DeploymentCellConfiguration{
		"byoc-onprem": {
			Amenities: []openapiclient.Amenity{
				apiAmenity("Custom BYOC Template Amenity"),
			},
		},
	}

	template, err := deploymentCellTemplateForCloudProvider(&configs, " BYOC-ONPREM ")
	require.NoError(t, err)
	requireManagedAmenity(t, template, "Custom BYOC Template Amenity")
	require.NotContains(t, managedAmenityNames(template), "Headlamp")
}

func TestDeploymentCellTemplateForCloudProviderPreservesDisable(t *testing.T) {
	const disable = `$sys.deploymentCell.isImported`
	amenity := apiAmenity("External DNS")
	amenity.Disable = utils.ToPtr(disable)
	configs := map[string]openapiclient.DeploymentCellConfiguration{
		"aws": {
			Amenities: []openapiclient.Amenity{amenity},
		},
	}

	template, err := deploymentCellTemplateForCloudProvider(&configs, "aws")
	require.NoError(t, err)

	require.Len(t, template.ManagedAmenities, 1)
	require.NotNil(t, template.ManagedAmenities[0].Disable)
	require.Equal(t, disable, *template.ManagedAmenities[0].Disable)
}

func TestDeploymentCellTemplateForCloudProviderPreservesManagedIdentities(t *testing.T) {
	description := "Allows workloads to publish queue messages."
	configs := map[string]openapiclient.DeploymentCellConfiguration{
		"aws": {
			WorkloadIdentities: []openapiclient.ManagedWorkloadIdentity{
				{
					Identifier:  "queue-writer",
					Description: utils.ToPtr(description),
					Permissions: &openapiclient.ManagedWorkloadIdentityPermissions{
						Policies: &map[string]string{
							"aws": `{"Statement":[{"Action":["sqs:SendMessage"],"Effect":"Allow","Resource":"*"}]}`,
						},
					},
					Bindings: []openapiclient.ManagedWorkloadIdentityBinding{
						{
							ServiceAccount: openapiclient.ManagedWorkloadIdentityServiceAccount{
								Namespace: "queue-system",
								Name:      "queue-writer",
							},
						},
					},
				},
			},
		},
	}

	template, err := deploymentCellTemplateForCloudProvider(&configs, "aws")
	require.NoError(t, err)
	require.Len(t, template.ManagedIdentities, 1)

	identity := template.ManagedIdentities[0]
	require.Equal(t, "queue-writer", identity.Identifier)
	require.Equal(t, description, *identity.Description)
	require.NotNil(t, identity.Permissions)
	require.Contains(t, identity.Permissions.Policies["aws"], "sqs:SendMessage")
	require.Len(t, identity.Bindings, 1)
	require.Equal(t, "queue-system", identity.Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, "queue-writer", identity.Bindings[0].ServiceAccount.Name)
}

func TestDeploymentCellTemplateForCloudProviderReturnsErrorForMissingNonBYOCConfig(t *testing.T) {
	configs := map[string]openapiclient.DeploymentCellConfiguration{
		"aws": {
			Amenities: []openapiclient.Amenity{
				apiAmenity("AWS Amenity"),
			},
		},
	}

	_, err := deploymentCellTemplateForCloudProvider(&configs, "gcp")
	require.ErrorContains(t, err, "no configuration found for cloud provider 'gcp'")
}

func apiAmenity(name string) openapiclient.Amenity {
	return openapiclient.Amenity{
		Name:        utils.ToPtr(name),
		Description: utils.ToPtr(name),
		Type:        utils.ToPtr("Helm"),
		IsManaged:   utils.ToPtr(true),
	}
}

func requireManagedAmenity(t *testing.T, template model.DeploymentCellTemplate, name string) {
	t.Helper()
	require.Contains(t, managedAmenityNames(template), name)
}

func managedAmenityNames(template model.DeploymentCellTemplate) map[string]struct{} {
	names := make(map[string]struct{}, len(template.ManagedAmenities))
	for _, amenity := range template.ManagedAmenities {
		names[amenity.Name] = struct{}{}
	}
	return names
}
