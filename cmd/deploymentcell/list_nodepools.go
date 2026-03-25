package deploymentcell

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var listNodepoolsCmd = &cobra.Command{
	Use:          "list-nodepools",
	Short:        "List all nodepools in a deployment cell",
	Long:         `List all nodepools in a deployment cell with their details.`,
	RunE:         runListNodepools,
	SilenceUsage: true,
}

func init() {
	listNodepoolsCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	_ = listNodepoolsCmd.MarkFlagRequired("id")
}

func runListNodepools(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("id")
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

	nodepools, rawEntities, err := dataaccess.ListNodepools(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print output in requested format
	if output == "json" {
		// For JSON, use raw API response
		err = utils.PrintTextTableJsonArrayOutput(output, rawEntities)
	} else {
		// For table/text, use formatted view
		err = utils.PrintTextTableJsonArrayOutput(output, nodepools)
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
