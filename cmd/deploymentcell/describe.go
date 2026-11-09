package deploymentcell

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [deployment-cell-id]",
	Short: "Describe a deployment cell",
	Long:  "Describe a deployment cell by ID, including its current managed artifact version.",
	Example: `# Describe a deployment cell
omnistrate-ctl deployment-cell describe hc-12345678

# Describe a deployment cell with JSON output
omnistrate-ctl deployment-cell describe hc-12345678 --output json`,
	Args:         cobra.ExactArgs(1),
	RunE:         runDescribe,
	SilenceUsage: true,
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID := args[0]
	if strings.TrimSpace(deploymentCellID) == "" {
		return fmt.Errorf("deployment cell ID is required")
	}

	output, err := cmd.Flags().GetString(common.OutputFlag)
	if err != nil {
		return err
	}
	if output != "text" && output != "table" && output != "json" {
		return fmt.Errorf("unsupported output format: %s", output)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	hostCluster, err := dataaccess.DescribeHostCluster(cmd.Context(), token, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to describe deployment cell %s: %w", deploymentCellID, err)
	}

	deploymentCell := formatDeploymentCell(hostCluster)
	if output == "table" {
		return utils.PrintTextTableJsonOutput(output, deploymentCell.ToTableView())
	}

	return utils.PrintTextTableJsonOutput(output, deploymentCell)
}
