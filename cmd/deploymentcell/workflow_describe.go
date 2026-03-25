package deploymentcell

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var workflowDescribeCmd = &cobra.Command{
	Use:   "describe [deployment-cell-id] [workflow-id]",
	Short: "Describe a specific deployment cell workflow",
	Long:  "Get detailed information about a specific deployment cell workflow.",
	Args:  cobra.ExactArgs(2),
	RunE:  describeDeploymentCellWorkflow,
}

func describeDeploymentCellWorkflow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	deploymentCellID := args[0]
	workflowID := args[1]

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.DescribeDeploymentCellWorkflow(ctx, token, deploymentCellID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to describe deployment cell workflow: %w", err)
	}

	if result == nil {
		return fmt.Errorf("workflow not found")
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result.Workflow})
}
