package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var terminateCmd = &cobra.Command{
	Use:   "terminate [workflow-id]",
	Short: "Terminate a running workflow",
	Long:  "Terminate a running workflow. This will stop the workflow execution.",
	Args:  cobra.ExactArgs(1),
	RunE:  terminateWorkflow,
}

func init() {
	terminateCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	terminateCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")
	terminateCmd.Flags().BoolP("confirm", "y", false, "Skip confirmation prompt")

	_ = terminateCmd.MarkFlagRequired("service-id")
	_ = terminateCmd.MarkFlagRequired("environment-id")
}

func terminateWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
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

	result, err := dataaccess.TerminateWorkflow(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to terminate workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s terminated successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
