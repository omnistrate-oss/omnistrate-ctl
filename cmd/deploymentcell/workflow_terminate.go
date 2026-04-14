package deploymentcell

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var workflowTerminateCmd = &cobra.Command{
	Use:   "terminate [deployment-cell-id] [workflow-id]",
	Short: "Terminate a running deployment cell workflow",
	Long:  "Terminate a running deployment cell workflow. This will stop the workflow execution.",
	Args:  cobra.ExactArgs(2),
	RunE:  terminateDeploymentCellWorkflow,
}

func init() {
	workflowTerminateCmd.Flags().BoolP("confirm", "y", false, "Skip confirmation prompt")
}

func terminateDeploymentCellWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	deploymentCellID := args[0]
	workflowID := args[1]
	skipConfirm, _ := cmd.Flags().GetBool("confirm")

	if !skipConfirm {
		fmt.Printf("Are you sure you want to terminate workflow %s? (y/N): ", workflowID)
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			fmt.Println("Operation cancelled")
			return err
		}
		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.TerminateDeploymentCellWorkflow(ctx, token, deploymentCellID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to terminate deployment cell workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s terminated successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []any{result})
}
