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
