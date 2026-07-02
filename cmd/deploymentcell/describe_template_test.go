package deploymentcell

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestCreateDeploymentCellTemplatePreservesDisable(t *testing.T) {
	const disable = `$sys.deploymentCell.isImported`
	deploymentCell := &fleet.HostCluster{
		Amenities: []fleet.Amenity{
			{
				Name:        utils.ToPtr("External DNS"),
				Description: utils.ToPtr("External DNS"),
				Type:        utils.ToPtr("Helm"),
				IsManaged:   utils.ToPtr(true),
				Disable:     utils.ToPtr(disable),
			},
		},
	}

	template := createDeploymentCellTemplate(deploymentCell)

	require.Len(t, template.ManagedAmenities, 1)
	require.NotNil(t, template.ManagedAmenities[0].Disable)
	require.Equal(t, disable, *template.ManagedAmenities[0].Disable)
}

func TestCreateDeploymentCellTemplatePreservesWorkloadIdentities(t *testing.T) {
	description := "Allows workloads to publish queue messages"
	deploymentCell := &fleet.HostCluster{
		Amenities: []fleet.Amenity{
			{
				Name:        utils.ToPtr("External DNS"),
				Description: utils.ToPtr("External DNS"),
				Type:        utils.ToPtr("Helm"),
				IsManaged:   utils.ToPtr(true),
			},
		},
		AdditionalProperties: map[string]interface{}{
			"WorkloadIdentities": []interface{}{
				map[string]interface{}{
					"Identifier":  "queue-writer",
					"Description": description,
					"Bindings": []interface{}{
						map[string]interface{}{
							"ServiceAccount": map[string]interface{}{
								"Namespace": "queue-system",
								"Name":      "queue-writer",
							},
						},
					},
					"Permissions": map[string]interface{}{
						"Permissions": map[string]interface{}{
							"oci": []interface{}{"manage queues"},
						},
					},
				},
			},
		},
	}

	template := createDeploymentCellTemplate(deploymentCell)

	require.Len(t, template.ManagedAmenities, 1)
	require.Len(t, template.WorkloadIdentities, 1)
	require.Equal(t, "queue-writer", template.WorkloadIdentities[0].Identifier)
	require.Equal(t, "queue-system", template.WorkloadIdentities[0].Bindings[0].ServiceAccount.Namespace)
	require.Equal(t, "manage queues", template.WorkloadIdentities[0].Permissions.Permissions["oci"][0])
}
