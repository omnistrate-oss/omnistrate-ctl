package dataaccess

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestConvertTemplateToOpenAPIFormatPreservesDependsOn(t *testing.T) {
	helmType := "Helm"
	template := model.DeploymentCellTemplate{
		CustomAmenities: []model.Amenity{
			{
				Name:      "dependent",
				Type:      utils.ToPtr(helmType),
				DependsOn: []string{"namespace", "crds"},
			},
		},
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.Len(t, configs["aws"].Amenities, 1)
	require.Equal(t, []string{"namespace", "crds"}, configs["aws"].Amenities[0].DependsOn)
}

func TestConvertTemplateToOpenAPIFormatPreservesDisable(t *testing.T) {
	helmType := "Helm"
	disable := `$sys.deploymentCell.isImported`
	template := model.DeploymentCellTemplate{
		ManagedAmenities: []model.Amenity{
			{
				Name:        "External DNS",
				Type:        utils.ToPtr(helmType),
				Description: utils.ToPtr("External DNS"),
				Disable:     utils.ToPtr(disable),
			},
		},
		CustomAmenities: []model.Amenity{
			{
				Name:        "Custom DNS",
				Type:        utils.ToPtr(helmType),
				Description: utils.ToPtr("Custom DNS"),
				Disable:     utils.ToPtr(disable),
			},
		},
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.Len(t, configs["aws"].Amenities, 2)
	require.NotNil(t, configs["aws"].Amenities[0].Disable)
	require.Equal(t, disable, *configs["aws"].Amenities[0].Disable)
	require.NotNil(t, configs["aws"].Amenities[1].Disable)
	require.Equal(t, disable, *configs["aws"].Amenities[1].Disable)
}

func TestConvertTemplateToOpenAPIFormatPreservesWorkloadIdentities(t *testing.T) {
	description := "Allows workloads to publish queue messages"
	roleType := "Default"
	template := model.DeploymentCellTemplate{
		WorkloadIdentities: []model.ManagedWorkloadIdentity{
			{
				Identifier:  "queue-writer",
				Description: &description,
				Bindings: []model.ManagedWorkloadIdentityBinding{
					{
						ServiceAccount: &model.ManagedWorkloadIdentityServiceAccount{
							Namespace: "queue-system",
							Name:      "queue-writer",
						},
					},
				},
				Permissions: &model.ManagedWorkloadIdentityPermissions{
					Policies: map[string]string{
						"aws": `{"Statement":[]}`,
					},
					Roles: map[string][]model.ManagedWorkloadIdentityRole{
						"gcp": {
							{
								Name: "roles/pubsub.publisher",
								Type: &roleType,
							},
						},
					},
					Permissions: map[string][]string{
						"oci": {"manage queues"},
					},
				},
			},
		},
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.Contains(t, configs["aws"].AdditionalProperties, "WorkloadIdentities")

	workloadIdentities, ok := configs["aws"].AdditionalProperties["WorkloadIdentities"].([]model.ManagedWorkloadIdentity)
	require.True(t, ok)
	require.Len(t, workloadIdentities, 1)
	require.Equal(t, "queue-writer", workloadIdentities[0].Identifier)
	require.Equal(t, "queue-system", workloadIdentities[0].Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, `{"Statement":[]}`, workloadIdentities[0].Permissions.Policies["aws"])
}

func TestConvertTemplateToOpenAPIFormatPreservesExplicitEmptyWorkloadIdentities(t *testing.T) {
	template := model.DeploymentCellTemplate{
		WorkloadIdentities: []model.ManagedWorkloadIdentity{},
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs["aws"].AdditionalProperties, "WorkloadIdentities")

	workloadIdentities, ok := configs["aws"].AdditionalProperties["WorkloadIdentities"].([]model.ManagedWorkloadIdentity)
	require.True(t, ok)
	require.Empty(t, workloadIdentities)
}

func TestConvertTemplateToOpenAPIFormatOmitsNilWorkloadIdentities(t *testing.T) {
	converted := convertTemplateToOpenAPIFormat(model.DeploymentCellTemplate{}, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.NotContains(t, configs["aws"].AdditionalProperties, "WorkloadIdentities")
}

func TestConvertToDeploymentCellTemplatePreservesWorkloadIdentities(t *testing.T) {
	description := "Allows workloads to publish queue messages"
	template, err := ConvertToDeploymentCellTemplate(map[string]interface{}{
		"Amenities": []model.InternalAmenity{
			{
				Name:      "External DNS",
				Type:      utils.ToPtr("Helm"),
				IsManaged: utils.ToPtr(true),
			},
		},
		"WorkloadIdentities": []model.ManagedWorkloadIdentity{
			{
				Identifier:  "queue-writer",
				Description: &description,
				Bindings: []model.ManagedWorkloadIdentityBinding{
					{
						ServiceAccount: &model.ManagedWorkloadIdentityServiceAccount{
							Namespace: "queue-system",
							Name:      "queue-writer",
						},
					},
				},
				Permissions: &model.ManagedWorkloadIdentityPermissions{
					Permissions: map[string][]string{
						"oci": {"manage queues"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, template.ManagedAmenities, 1)
	require.Len(t, template.WorkloadIdentities, 1)
	require.Equal(t, "queue-writer", template.WorkloadIdentities[0].Identifier)
	require.Equal(t, "queue-system", template.WorkloadIdentities[0].Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, "manage queues", template.WorkloadIdentities[0].Permissions.Permissions["oci"][0])
}
