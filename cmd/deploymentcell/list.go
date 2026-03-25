package deploymentcell

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var listCmd = &cobra.Command{
	Use:          "list",
	Short:        "List all deployment cells",
	Long:         `List all deployment cells with their details.`,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringP("account-config-id", "a", "", "Filter by account config ID")
	listCmd.Flags().StringP("region-id", "r", "", "Filter by region ID")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountConfigID, err := cmd.Flags().GetString("account-config-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	regionID, err := cmd.Flags().GetString("region-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var accountConfigIDPtr, regionIDPtr *string
	if accountConfigID != "" {
		accountConfigIDPtr = &accountConfigID
	}
	if regionID != "" {
		regionIDPtr = &regionID
	}

	hostClusters, err := dataaccess.ListHostClusters(ctx, token, accountConfigIDPtr, regionIDPtr)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Convert to model structure
	var deploymentCells []model.DeploymentCell
	for _, cluster := range hostClusters.GetHostClusters() {
		deploymentCell := formatDeploymentCell(&cluster)
		deploymentCells = append(deploymentCells, deploymentCell)
	}

	// Print output in requested format
	if output == "table" {
		// Use table view for better readability
		var tableViews []model.DeploymentCellTableView
		for _, cell := range deploymentCells {
			tableViews = append(tableViews, cell.ToTableView())
		}
		err = utils.PrintTextTableJsonArrayOutput(output, tableViews)
	} else {
		// Use full model for JSON and text output
		err = utils.PrintTextTableJsonArrayOutput(output, deploymentCells)
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
