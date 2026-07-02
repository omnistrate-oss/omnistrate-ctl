package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDeploymentCellTemplateYAMLPreservesDisable(t *testing.T) {
	const disable = `$sys.deploymentCell.isImported`
	data := []byte(`
managedAmenities:
  - name: External DNS
    description: External DNS
    type: Helm
    disable: $sys.deploymentCell.isImported
customAmenities:
  - name: Custom DNS
    description: Custom DNS
    type: Helm
    disable: $sys.deploymentCell.isImported
`)

	var template DeploymentCellTemplate
	err := yaml.Unmarshal(data, &template)
	require.NoError(t, err)

	require.Len(t, template.ManagedAmenities, 1)
	require.NotNil(t, template.ManagedAmenities[0].Disable)
	require.Equal(t, disable, *template.ManagedAmenities[0].Disable)

	require.Len(t, template.CustomAmenities, 1)
	require.NotNil(t, template.CustomAmenities[0].Disable)
	require.Equal(t, disable, *template.CustomAmenities[0].Disable)
}

func TestDeploymentCellTemplateYAMLPreservesWorkloadIdentities(t *testing.T) {
	data := []byte(`
workloadIdentities:
  - identifier: queue-writer
    description: Allows workloads to publish queue messages
    bindings:
      - serviceAccount:
          namespace: queue-system
          name: queue-writer
    permissions:
      policies:
        aws: '{"Statement":[]}'
      roles:
        gcp:
          - name: roles/pubsub.publisher
            type: Default
      permissions:
        oci:
          - manage queues
`)

	var template DeploymentCellTemplate
	err := yaml.Unmarshal(data, &template)
	require.NoError(t, err)

	require.Len(t, template.WorkloadIdentities, 1)
	require.Equal(t, "queue-writer", template.WorkloadIdentities[0].Identifier)
	require.NotNil(t, template.WorkloadIdentities[0].Description)
	require.Equal(t, "Allows workloads to publish queue messages", *template.WorkloadIdentities[0].Description)
	require.Len(t, template.WorkloadIdentities[0].Bindings, 1)
	require.Equal(t, "queue-system", template.WorkloadIdentities[0].Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, `{"Statement":[]}`, template.WorkloadIdentities[0].Permissions.Policies["aws"])
	require.Equal(t, "roles/pubsub.publisher", template.WorkloadIdentities[0].Permissions.Roles["gcp"][0].Name)
	require.Equal(t, "manage queues", template.WorkloadIdentities[0].Permissions.Permissions["oci"][0])

	output, err := yaml.Marshal(template)
	require.NoError(t, err)
	require.Contains(t, string(output), "workloadIdentities:")
	require.Contains(t, string(output), "identifier: queue-writer")
}

func TestDeploymentCellTemplateYAMLPreservesExplicitEmptyWorkloadIdentities(t *testing.T) {
	data := []byte(`
workloadIdentities: []
`)

	var template DeploymentCellTemplate
	err := yaml.Unmarshal(data, &template)
	require.NoError(t, err)

	require.NotNil(t, template.WorkloadIdentities)
	require.Empty(t, template.WorkloadIdentities)

	output, err := yaml.Marshal(template)
	require.NoError(t, err)
	require.Contains(t, string(output), "workloadIdentities: []")
}

func TestDeploymentCellTemplateJSONUsesConsistentTopLevelFieldNames(t *testing.T) {
	description := "Allows workloads to publish queue messages"
	template := DeploymentCellTemplate{
		WorkloadIdentities: []ManagedWorkloadIdentity{
			{
				Identifier:  "queue-writer",
				Description: &description,
			},
		},
	}

	output, err := json.Marshal(template)
	require.NoError(t, err)
	require.Contains(t, string(output), `"workload_identities"`)
	require.NotContains(t, string(output), `"WorkloadIdentities"`)
}
