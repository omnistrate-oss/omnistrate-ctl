package model

import (
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

func TestDeploymentCellTemplateYAMLPreservesManagedIdentities(t *testing.T) {
	data := []byte(`
managedIdentities:
  - identifier: queue-writer
    description: Allows workloads to publish queue messages.
    permissions:
      policies:
        aws: |
          {"Statement":[{"Action":["sqs:SendMessage"],"Effect":"Allow","Resource":"*"}]}
    bindings:
      - serviceAccount:
          namespace: queue-system
          name: queue-writer
`)

	var template DeploymentCellTemplate
	err := yaml.Unmarshal(data, &template)
	require.NoError(t, err)

	require.Len(t, template.ManagedIdentities, 1)
	identity := template.ManagedIdentities[0]
	require.Equal(t, "queue-writer", identity.Identifier)
	require.NotNil(t, identity.Description)
	require.Equal(t, "Allows workloads to publish queue messages.", *identity.Description)
	require.NotNil(t, identity.Permissions)
	require.Contains(t, identity.Permissions.Policies["aws"], "sqs:SendMessage")
	require.Len(t, identity.Bindings, 1)
	require.Equal(t, "queue-system", identity.Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, "queue-writer", identity.Bindings[0].ServiceAccount.Name)
}
