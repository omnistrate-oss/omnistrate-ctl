package deploymentcell

import (
	"context"
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var deleteNodepoolCmd = &cobra.Command{
	Use:          "delete-nodepool",
	Short:        "Delete a nodepool from a deployment cell",
	Long:         `Delete a nodepool from a deployment cell.`,
	RunE:         runDeleteNodepool,
	SilenceUsage: true,
}

func init() {
	deleteNodepoolCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	deleteNodepoolCmd.Flags().StringP("nodepool", "n", "", "Nodepool name (required)")
	_ = deleteNodepoolCmd.MarkFlagRequired("id")
	_ = deleteNodepoolCmd.MarkFlagRequired("nodepool")
}

func runDeleteNodepool(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	nodepoolName, err := cmd.Flags().GetString("nodepool")
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

	// Show spinner while deleting (can take up to 10 minutes)
	sm := ysmrr.NewSpinnerManager()
	spinner := sm.AddSpinner(fmt.Sprintf("Deleting nodepool '%s' from deployment cell '%s'...", nodepoolName, deploymentCellID))
	utils.StartSpinnerWithCleanup(sm)

	err = dataaccess.DeleteNodepool(ctx, token, deploymentCellID, nodepoolName)

	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully deleted nodepool '%s' from deployment cell '%s'", nodepoolName, deploymentCellID))
	return nil
}
