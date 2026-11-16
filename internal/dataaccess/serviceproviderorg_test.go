package dataaccess

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestConvertTemplateToOpenAPIFormatPreservesManagedReleaseVersion(t *testing.T) {
	template := model.DeploymentCellTemplate{
		ManagedReleaseVersion: utils.ToPtr("r0000020"),
	}

	converted := convertTemplateToOpenAPIFormat(template, "aws")
	configs := converted.GetDeploymentCellConfigurationPerCloudProvider()
	require.Contains(t, configs, "aws")
	require.NotNil(t, configs["aws"].ManagedReleaseVersion)
	require.Equal(t, "r0000020", *configs["aws"].ManagedReleaseVersion)
}

func TestConvertTemplateToOpenAPIFormatManagedReleaseVersionPolicy(t *testing.T) {
	tests := []struct {
		name      string
		requested *string
		want      *string
	}{
		{name: "omitted keeps current policy", requested: nil, want: nil},
		{name: "empty keeps current policy", requested: utils.ToPtr("  "), want: nil},
		{name: "autoUpgrade selects automatic adoption", requested: utils.ToPtr(" AUTOUPGRADE "), want: utils.ToPtr(model.ManagedReleaseVersionAutoUpgrade)},
		{name: "bundle version pins release", requested: utils.ToPtr(" r0000020 "), want: utils.ToPtr("r0000020")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converted := convertTemplateToOpenAPIFormat(model.DeploymentCellTemplate{ManagedReleaseVersion: test.requested}, "aws")
			require.Equal(t, test.want, converted.GetDeploymentCellConfigurationPerCloudProvider()["aws"].ManagedReleaseVersion)
		})
	}
}

func TestConvertDeploymentCellConfigurationFromOpenAPIReportsAutomaticAsAutoUpgrade(t *testing.T) {
	for _, reported := range []string{"", "autoUpgrade", "autoupgrade"} {
		template := ConvertDeploymentCellConfigurationFromOpenAPI(openapiclient.DeploymentCellConfiguration{
			ManagedReleaseVersion: utils.ToPtr(reported),
		})

		require.Equal(t, utils.ToPtr(model.ManagedReleaseVersionAutoUpgrade), template.ManagedReleaseVersion, reported)
	}
}

func TestUpdateServiceProviderOrganizationSendsAutoUpgradeWithoutReleaseLookup(t *testing.T) {
	var releaseLookedUp bool
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/managed-artifact/releases/") {
			releaseLookedUp = true
			w.WriteHeader(http.StatusNotFound)
			return
		}
		payload, _ := io.ReadAll(r.Body)
		body = string(payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	err := UpdateServiceProviderOrganization(context.Background(), "test-token", model.DeploymentCellTemplate{
		ManagedReleaseVersion: utils.ToPtr("AutoUpgrade"),
	}, "GLOBAL", "aws")

	require.NoError(t, err)
	require.False(t, releaseLookedUp)
	require.Contains(t, body, `"ManagedReleaseVersion":"autoUpgrade"`)
}

func TestUpdateServiceProviderOrganizationRejectsUnknownManagedReleaseVersion(t *testing.T) {
	var organizationUpdated bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/2022-09-01-00/managed-artifact/releases/r9999999" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"name":"not_found","message":"managed artifact release r9999999 was not found"}`))
			return
		}
		organizationUpdated = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	err := UpdateServiceProviderOrganization(context.Background(), "test-token", model.DeploymentCellTemplate{
		ManagedReleaseVersion: utils.ToPtr("r9999999"),
	}, "GLOBAL", "aws")

	require.ErrorContains(t, err, `managed release version "r9999999" was not found`)
	require.False(t, organizationUpdated)
}

func TestConvertDeploymentCellConfigurationFromOpenAPIPreservesManagedReleaseVersion(t *testing.T) {
	template := ConvertDeploymentCellConfigurationFromOpenAPI(openapiclient.DeploymentCellConfiguration{
		ManagedReleaseVersion: utils.ToPtr("r0000020"),
	})

	require.NotNil(t, template.ManagedReleaseVersion)
	require.Equal(t, "r0000020", *template.ManagedReleaseVersion)
}
