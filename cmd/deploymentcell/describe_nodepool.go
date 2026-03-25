package deploymentcell

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var describeNodepoolCmd = &cobra.Command{
	Use:          "describe-nodepool",
	Short:        "Describe a nodepool in a deployment cell",
	Long:         `Get detailed information about a specific nodepool in a deployment cell.`,
	RunE:         runDescribeNodepool,
	SilenceUsage: true,
}

func init() {
	describeNodepoolCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	describeNodepoolCmd.Flags().StringP("nodepool", "n", "", "Nodepool name (required)")
	_ = describeNodepoolCmd.MarkFlagRequired("id")
	_ = describeNodepoolCmd.MarkFlagRequired("nodepool")
}

func runDescribeNodepool(cmd *cobra.Command, args []string) error {
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

	nodepool, rawEntity, err := dataaccess.DescribeNodepool(ctx, token, deploymentCellID, nodepoolName)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print output in requested format
	if output == "json" {
		// For JSON, use raw API response
		err = utils.PrintTextTableJsonOutput(output, rawEntity)
	} else {
		// For table/text, use formatted view
		err = utils.PrintTextTableJsonOutput(output, nodepool)
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
