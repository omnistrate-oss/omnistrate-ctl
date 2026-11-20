package deploymentcell

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
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

func TestManagedReleaseVersionDisplayDoesNotClaimAutoUpgradeWhenNotReported(t *testing.T) {
	require.Equal(t, "not reported by API", managedReleaseVersionDisplay(nil))
	require.Equal(t, model.ManagedReleaseVersionAutoUpgrade, managedReleaseVersionDisplay(utils.ToPtr(model.ManagedReleaseVersionAutoUpgrade)))
	require.Equal(t, "r0000020", managedReleaseVersionDisplay(utils.ToPtr("r0000020")))
}
