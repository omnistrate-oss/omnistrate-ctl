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
