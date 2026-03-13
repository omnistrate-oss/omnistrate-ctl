package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry [workflow-id]",
	Short: "Retry a failed workflow",
	Long:  "Retry a failed workflow execution.",
	Args:  cobra.ExactArgs(1),
	RunE:  retryWorkflow,
}

func init() {
	retryCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	retryCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = retryCmd.MarkFlagRequired("service-id")
	_ = retryCmd.MarkFlagRequired("environment-id")
}

func retryWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.RetryWorkflow(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to retry workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s retry requested successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []any{result})
}
