package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get workflow summary for a service environment",
	Long:  "Get a summary of all workflows for a specific service and environment.",
	RunE:  describeWorkflowSummary,
}

func init() {
	summaryCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	summaryCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = summaryCmd.MarkFlagRequired("service-id")
	_ = summaryCmd.MarkFlagRequired("environment-id")
}

func describeWorkflowSummary(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.DescribeWorkflowSummary(ctx, token, serviceID, environmentID)
	if err != nil {
		return fmt.Errorf("failed to get workflow summary: %w", err)
	}

	if result == nil {
		fmt.Println("No workflow summary found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}