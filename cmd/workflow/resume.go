package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume [workflow-id]",
	Short: "Resume a paused workflow",
	Long:  "Resume a paused workflow execution.",
	Args:  cobra.ExactArgs(1),
	RunE:  resumeWorkflow,
}

func init() {
	resumeCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	resumeCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = resumeCmd.MarkFlagRequired("service-id")
	_ = resumeCmd.MarkFlagRequired("environment-id")
}

func resumeWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.ResumeWorkflow(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to resume workflow: %w", err)
	}

	if result == nil {
		fmt.Printf("Workflow %s resumed successfully\n", workflowID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
