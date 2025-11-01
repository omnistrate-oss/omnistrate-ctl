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

var configureNodepoolCmd = &cobra.Command{
	Use:          "configure-nodepool",
	Short:        "Configure a nodepool in a deployment cell",
	Long:         `Configure properties of a nodepool in a deployment cell, such as max size.`,
	RunE:         runConfigureNodepool,
	SilenceUsage: true,
}

func init() {
	configureNodepoolCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	configureNodepoolCmd.Flags().StringP("nodepool", "n", "", "Nodepool name (required)")
	configureNodepoolCmd.Flags().Int64P("max-size", "m", 0, "Maximum size for the nodepool (required)")
	_ = configureNodepoolCmd.MarkFlagRequired("id")
	_ = configureNodepoolCmd.MarkFlagRequired("nodepool")
	_ = configureNodepoolCmd.MarkFlagRequired("max-size")
}

func runConfigureNodepool(cmd *cobra.Command, args []string) error {
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

	maxSize, err := cmd.Flags().GetInt64("max-size")
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

	err = dataaccess.ConfigureNodepool(ctx, token, deploymentCellID, nodepoolName, maxSize)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully configured nodepool '%s' in deployment cell '%s'\n", nodepoolName, deploymentCellID)
	return nil
}
