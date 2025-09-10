package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events [workflow-id]",
	Short: "Get events for a specific workflow",
	Long:  "Retrieve all events and resource details for a specific workflow.",
	Args:  cobra.ExactArgs(1),
	RunE:  getWorkflowEvents,
}

func init() {
	eventsCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	eventsCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = eventsCmd.MarkFlagRequired("service-id")
	_ = eventsCmd.MarkFlagRequired("environment-id")
}

func getWorkflowEvents(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.GetWorkflowEvents(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow events: %w", err)
	}

	if result == nil {
		fmt.Println("No workflow events found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
