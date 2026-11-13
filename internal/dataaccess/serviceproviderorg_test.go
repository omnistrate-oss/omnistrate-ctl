package dataaccess

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
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

func TestConvertTemplateToOpenAPIFormatPreservesManagedIdentities(t *testing.T) {
	description := "Allows workloads to publish queue messages."
	template := model.DeploymentCellTemplate{
		ManagedIdentities: []model.ManagedWorkloadIdentity{
			{
				Identifier:  "queue-writer",
				Description: utils.ToPtr(description),
				Permissions: &model.ManagedWorkloadIdentityPermissions{
					Policies: map[string]string{
						"aws": `{"Statement":[{"Action":["sqs:SendMessage"],"Effect":"Allow","Resource":"*"}]}`,
					},
				},
				Bindings: []model.ManagedWorkloadIdentityBinding{
					{
						ServiceAccount: model.ManagedWorkloadIdentityServiceAccount{
							Namespace: "queue-system",
							Name:      "queue-writer",
						},
					},
				},
			},
		},
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.Len(t, configs["aws"].WorkloadIdentities, 1)

	identity := configs["aws"].WorkloadIdentities[0]
	require.Equal(t, "queue-writer", identity.Identifier)
	require.Equal(t, description, *identity.Description)
	require.NotNil(t, identity.Permissions)
	require.Contains(t, (*identity.Permissions.Policies)["aws"], "sqs:SendMessage")
	require.Len(t, identity.Bindings, 1)
	require.Equal(t, "queue-system", identity.Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, "queue-writer", identity.Bindings[0].ServiceAccount.Name)
}

func TestConvertTemplateToOpenAPIFormatOmitsManagedReleaseVersion(t *testing.T) {
	template := model.DeploymentCellTemplate{
		ManagedReleaseVersion: utils.ToPtr("r0000020"),
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.Nil(t, configs["aws"].ManagedReleaseVersion)
}

func TestConvertDeploymentCellConfigurationFromOpenAPIPreservesManagedReleaseVersion(t *testing.T) {
	template := ConvertDeploymentCellConfigurationFromOpenAPI(openapiclient.DeploymentCellConfiguration{
		ManagedReleaseVersion: utils.ToPtr("r0000020"),
	})

	require.NotNil(t, template.ManagedReleaseVersion)
	require.Equal(t, "r0000020", *template.ManagedReleaseVersion)
}
