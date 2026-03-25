package deploymentcell

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var scaleDownNodepoolCmd = &cobra.Command{
	Use:   "scale-down-nodepool",
	Short: "Scale down a nodepool to zero nodes",
	Long: `Scale down a nodepool to zero nodes by setting max size to 0.

This will evict all nodes in the nodepool and can be used as a cost saving measure.
The nodepool will remain configured but will have no running nodes.`,
	RunE:         runScaleDownNodepool,
	SilenceUsage: true,
}

func init() {
	scaleDownNodepoolCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	scaleDownNodepoolCmd.Flags().StringP("nodepool", "n", "", "Nodepool name (required)")
	_ = scaleDownNodepoolCmd.MarkFlagRequired("id")
	_ = scaleDownNodepoolCmd.MarkFlagRequired("nodepool")
}

func runScaleDownNodepool(cmd *cobra.Command, args []string) error {
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

	// Scale down to 0 nodes
	err = dataaccess.ConfigureNodepool(ctx, token, deploymentCellID, nodepoolName, 0)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully scaled down nodepool '%s' in deployment cell '%s' to 0 nodes", nodepoolName, deploymentCellID))
	return nil
}
