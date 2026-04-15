package deploymentcell

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var workflowResumeCmd = &cobra.Command{
	Use:   "resume [deployment-cell-id] [workflow-id]",
	Short: "Resume a paused deployment cell workflow",
	Long:  "Resume a paused deployment cell workflow execution.",
	Args:  cobra.ExactArgs(2),
	RunE:  resumeDeploymentCellWorkflow,
}

func resumeDeploymentCellWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	deploymentCellID := args[0]
	workflowID := args[1]

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.ResumeDeploymentCellWorkflow(ctx, token, deploymentCellID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to resume deployment cell workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s resumed successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []any{result})
}
