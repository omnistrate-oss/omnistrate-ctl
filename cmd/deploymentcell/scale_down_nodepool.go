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
	Short: "Scale down a nodepool to minimum size",
	Long: fmt.Sprintf(`Scale down a nodepool by setting max size to %d.

This can be used as a cost saving measure to reduce the nodepool capacity.
When set to 0, the nodepool will remain configured but will have no running nodes.`, DefaultMinNodepoolSize),
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

	// Scale down to minimum size
	err = dataaccess.ConfigureNodepool(ctx, token, deploymentCellID, nodepoolName, DefaultMinNodepoolSize)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully scaled down nodepool '%s' in deployment cell '%s' to %d nodes", nodepoolName, deploymentCellID, DefaultMinNodepoolSize))
	return nil
}
