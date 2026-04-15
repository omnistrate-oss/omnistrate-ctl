package deploymentcell

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var workflowRetryCmd = &cobra.Command{
	Use:   "retry [deployment-cell-id] [workflow-id]",
	Short: "Retry a failed deployment cell workflow",
	Long:  "Retry a failed deployment cell workflow execution.",
	Args:  cobra.ExactArgs(2),
	RunE:  retryDeploymentCellWorkflow,
}

func retryDeploymentCellWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	deploymentCellID := args[0]
	workflowID := args[1]

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.RetryDeploymentCellWorkflow(ctx, token, deploymentCellID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to retry deployment cell workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s retry requested successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []any{result})
}
