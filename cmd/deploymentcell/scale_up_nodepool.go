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

var scaleUpNodepoolCmd = &cobra.Command{
	Use:   "scale-up-nodepool",
	Short: "Scale up a nodepool to the default maximum size",
	Long: fmt.Sprintf(`Scale up a nodepool to the default maximum size of %d nodes.

This restores the nodepool to its default capacity after being scaled down.
Nodes will be provisioned as needed by the autoscaler.`, DefaultMaxNodepoolSize),
	RunE:         runScaleUpNodepool,
	SilenceUsage: true,
}

func init() {
	scaleUpNodepoolCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	scaleUpNodepoolCmd.Flags().StringP("nodepool", "n", "", "Nodepool name (required)")
	_ = scaleUpNodepoolCmd.MarkFlagRequired("id")
	_ = scaleUpNodepoolCmd.MarkFlagRequired("nodepool")
}

func runScaleUpNodepool(cmd *cobra.Command, args []string) error {
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

	// Scale up to default max size
	err = dataaccess.ConfigureNodepool(ctx, token, deploymentCellID, nodepoolName, DefaultMaxNodepoolSize)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully scaled up nodepool '%s' in deployment cell '%s' to %d max nodes", nodepoolName, deploymentCellID, DefaultMaxNodepoolSize))
	return nil
}
