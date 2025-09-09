package operations

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var deploymentCellHealthCmd = &cobra.Command{
	Use:   "deployment-cell-health",
	Short: "Get deployment cell health details",
	Long:  "Get detailed health information for deployment cells, including host clusters and services.",
	RunE:  runDeploymentCellHealth,
}

func init() {
	deploymentCellHealthCmd.Flags().String("host-cluster-id", "", "Host cluster ID to get health for")
	deploymentCellHealthCmd.Flags().StringP("service-id", "s", "", "Service ID to get health for")
	deploymentCellHealthCmd.Flags().StringP("environment-id", "e", "", "Service environment ID to get health for")
}

func runDeploymentCellHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	hostClusterID, _ := cmd.Flags().GetString("host-cluster-id")
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	opts := &dataaccess.DeploymentCellHealthOptions{
		HostClusterID:        hostClusterID,
		ServiceID:            serviceID,
		ServiceEnvironmentID: environmentID,
	}

	result, err := dataaccess.GetDeploymentCellHealth(ctx, token, opts)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}