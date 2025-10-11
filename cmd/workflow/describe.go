package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [workflow-id]",
	Short: "Describe a specific workflow",
	Long:  "Get detailed information about a specific workflow.",
	Args:  cobra.ExactArgs(1),
	RunE:  describeWorkflow,
}

func init() {
	describeCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	describeCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = describeCmd.MarkFlagRequired("service-id")
	_ = describeCmd.MarkFlagRequired("environment-id")
}

func describeWorkflow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.DescribeWorkflow(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to describe workflow: %w", err)
	}

	if result == nil {
		return fmt.Errorf("workflow not found")
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result.Workflow})
}
