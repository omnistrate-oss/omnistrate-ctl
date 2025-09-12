package workflow

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
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
	eventsCmd.Flags().String("resource-id", "", "Filter results by resource ID")
	eventsCmd.Flags().String("resource-key", "", "Filter results by resource key")

	_ = eventsCmd.MarkFlagRequired("service-id")
	_ = eventsCmd.MarkFlagRequired("environment-id")
}

func getWorkflowEvents(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	workflowID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	resourceID, _ := cmd.Flags().GetString("resource-id")
	resourceKey, _ := cmd.Flags().GetString("resource-key")

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

	// Apply resource filtering if specified
	if resourceID != "" || resourceKey != "" {
		filteredResult, err := filterWorkflowEventsByResource(result, resourceID, resourceKey)
		if err != nil {
			return fmt.Errorf("failed to apply resource filter: %w", err)
		}
		result = filteredResult
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func filterWorkflowEventsByResource(result *fleet.GetWorkflowEventsResult, resourceID, resourceKey string) (*fleet.GetWorkflowEventsResult, error) {
	if result == nil || result.Resources == nil {
		return result, nil
	}

	// Create a filtered copy of the result
	filteredResult := *result
	var filteredResources []fleet.EventsPerResource

	// Filter resources based on resourceID or resourceKey
	for _, resource := range result.Resources {
		includeResource := false

		// Check resource ID filter
		if resourceID != "" && resource.ResourceId == resourceID {
			includeResource = true
		}

		// Check resource key filter
		if resourceKey != "" && resource.ResourceKey == resourceKey {
			includeResource = true
		}

		// If no filters are specified, include all resources (this shouldn't happen based on calling logic)
		if resourceID == "" && resourceKey == "" {
			includeResource = true
		}

		// If both filters are specified, the resource must match at least one
		if includeResource {
			filteredResources = append(filteredResources, resource)
		}
	}

	filteredResult.Resources = filteredResources

	// Add filtering metadata as additional properties
	if filteredResult.AdditionalProperties == nil {
		filteredResult.AdditionalProperties = make(map[string]interface{})
	}

	filterInfo := map[string]interface{}{}
	if resourceID != "" {
		filterInfo["resourceId"] = resourceID
	}
	if resourceKey != "" {
		filterInfo["resourceKey"] = resourceKey
	}
	filteredResult.AdditionalProperties["appliedFilters"] = filterInfo

	// Add count information
	filteredResult.AdditionalProperties["totalResources"] = len(result.Resources)
	filteredResult.AdditionalProperties["filteredResources"] = len(filteredResources)

	return &filteredResult, nil
}
