package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events [workflow-id]",
	Short: "Get workflow execution status and events",
	Long: `Get workflow execution status showing resources, steps, and their status.

By default, shows a summary with:
- Resources involved in the workflow
- Workflow steps for each resource  
- Status of each step (success, failed, running, etc.)

Use --detail to see full event details for each step.`,
	Args: cobra.ExactArgs(1),
	RunE: getWorkflowEvents,
	Example: `  # Show workflow summary with step statuses
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id>

  # Show detailed events for all steps
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail

  # Filter to specific resource and show details
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --resource-key mydb --detail

  # Show only specific steps
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --step-names Bootstrap,Deployment

  # Show events from a specific time period
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --since 2024-01-15T10:00:00Z`,
}

func init() {
	eventsCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	eventsCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")
	eventsCmd.Flags().String("resource-id", "", "Filter to specific resource by ID")
	eventsCmd.Flags().String("resource-key", "", "Filter to specific resource by key/name")
	eventsCmd.Flags().StringSlice("step-names", []string{}, "Filter by step names (e.g., Bootstrap, Compute, Deployment, Network, Storage, Monitoring)")
	eventsCmd.Flags().Bool("detail", false, "Show detailed events for each step (default: show step summary with status only)")
	eventsCmd.Flags().String("since", "", "Show events after this time (RFC3339 format, e.g. 2024-01-15T10:00:00Z)")
	eventsCmd.Flags().String("until", "", "Show events before this time (RFC3339 format, e.g. 2024-01-15T11:00:00Z)")

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
	stepNames, _ := cmd.Flags().GetStringSlice("step-names")
	detail, _ := cmd.Flags().GetBool("detail")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")

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

	// Apply filtering and formatting
	filterOptions := WorkflowEventFilterOptions{
		ResourceID:  resourceID,
		ResourceKey: resourceKey,
		StepNames:   stepNames,
		Detail:      detail,
		Since:       since,
		Until:       until,
	}

	filteredResult, err := filterWorkflowEvents(result, filterOptions)
	if err != nil {
		return fmt.Errorf("failed to apply filters: %w", err)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{filteredResult})
}

// WorkflowEventFilterOptions contains filtering options for workflow events
type WorkflowEventFilterOptions struct {
	ResourceID  string
	ResourceKey string
	StepNames   []string
	Detail      bool
	Since       string
	Until       string
}

// WorkflowStepSummary represents a simplified view of a workflow step with status
type WorkflowStepSummary struct {
	StepName     string                       `json:"stepName"`
	Status       string                       `json:"status"`
	StartTime    string                       `json:"startTime,omitempty"`
	EndTime      string                       `json:"endTime,omitempty"`
	EventCount   int                          `json:"eventCount"`
	DetailedStep *fleet.EventsPerWorkflowStep `json:"detailedStep,omitempty"`
}

// WorkflowResourceSummary represents a simplified view of resource workflow execution
type WorkflowResourceSummary struct {
	ResourceId   string                `json:"resourceId"`
	ResourceKey  string                `json:"resourceKey"`
	ResourceName string                `json:"resourceName"`
	Steps        []WorkflowStepSummary `json:"steps"`
}

// WorkflowEventsSummary represents the simplified workflow events response
type WorkflowEventsSummary struct {
	WorkflowId     string                    `json:"workflowId"`
	EnvironmentId  string                    `json:"environmentId"`
	ServiceId      string                    `json:"serviceId"`
	Resources      []WorkflowResourceSummary `json:"resources"`
	FilteringStats map[string]interface{}    `json:"filteringStats,omitempty"`
	AppliedFilters map[string]interface{}    `json:"appliedFilters,omitempty"`
}

func filterWorkflowEvents(result *fleet.GetWorkflowEventsResult, options WorkflowEventFilterOptions) (*WorkflowEventsSummary, error) {
	if result == nil || result.Resources == nil {
		return &WorkflowEventsSummary{
			WorkflowId:    result.Id,
			EnvironmentId: result.EnvironmentId,
			ServiceId:     result.ServiceId,
			Resources:     []WorkflowResourceSummary{},
		}, nil
	}

	// Parse time filters if provided
	var sinceTime, untilTime *time.Time
	if options.Since != "" {
		t, err := time.Parse(time.RFC3339, options.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since time format: %w", err)
		}
		sinceTime = &t
	}
	if options.Until != "" {
		t, err := time.Parse(time.RFC3339, options.Until)
		if err != nil {
			return nil, fmt.Errorf("invalid until time format: %w", err)
		}
		untilTime = &t
	}

	var summaryResources []WorkflowResourceSummary
	originalTotalSteps := 0
	originalTotalEvents := 0
	filteredTotalSteps := 0
	filteredTotalEvents := 0

	// Process each resource
	for _, resource := range result.Resources {
		// Check resource-level filtering
		if !matchesResourceFilter(resource, options.ResourceID, options.ResourceKey) {
			originalTotalSteps += len(resource.WorkflowSteps)
			for _, step := range resource.WorkflowSteps {
				originalTotalEvents += len(step.Events)
			}
			continue
		}

		var summarySteps []WorkflowStepSummary

		// Process each workflow step
		for _, step := range resource.WorkflowSteps {
			originalTotalSteps++
			originalTotalEvents += len(step.Events)

			// Check step name filtering
			if !matchesStepNameFilter(step.StepName, options.StepNames) {
				continue
			}

			// Filter events by time if specified
			var relevantEvents []fleet.WorkflowEvent
			for _, event := range step.Events {
				if matchesTimeFilter(event, sinceTime, untilTime) {
					relevantEvents = append(relevantEvents, event)
				}
			}

			// Only include step if it has relevant events
			if len(relevantEvents) > 0 {
				stepSummary := WorkflowStepSummary{
					StepName:   step.StepName,
					Status:     determineStepStatus(relevantEvents),
					EventCount: len(relevantEvents),
				}

				// Add timing information
				if len(relevantEvents) > 0 {
					stepSummary.StartTime = relevantEvents[0].EventTime
					stepSummary.EndTime = relevantEvents[len(relevantEvents)-1].EventTime
				}

				// Include detailed step info if --detail flag is used
				if options.Detail {
					detailedStep := step
					detailedStep.Events = relevantEvents
					stepSummary.DetailedStep = &detailedStep
				}

				summarySteps = append(summarySteps, stepSummary)
				filteredTotalSteps++
				filteredTotalEvents += len(relevantEvents)
			}
		}

		// Only include resource if it has steps
		if len(summarySteps) > 0 {
			summaryResources = append(summaryResources, WorkflowResourceSummary{
				ResourceId:   resource.ResourceId,
				ResourceKey:  resource.ResourceKey,
				ResourceName: resource.ResourceName,
				Steps:        summarySteps,
			})
		}
	}

	// Create summary result
	summary := &WorkflowEventsSummary{
		WorkflowId:    result.Id,
		EnvironmentId: result.EnvironmentId,
		ServiceId:     result.ServiceId,
		Resources:     summaryResources,
	}

	// Add filtering metadata
	if options.ResourceID != "" || options.ResourceKey != "" || len(options.StepNames) > 0 ||
		options.Since != "" || options.Until != "" || options.Detail {
		appliedFilters := map[string]interface{}{}

		if options.ResourceID != "" {
			appliedFilters["resourceId"] = options.ResourceID
		}
		if options.ResourceKey != "" {
			appliedFilters["resourceKey"] = options.ResourceKey
		}
		if len(options.StepNames) > 0 {
			appliedFilters["stepNames"] = options.StepNames
		}
		if options.Since != "" {
			appliedFilters["since"] = options.Since
		}
		if options.Until != "" {
			appliedFilters["until"] = options.Until
		}
		if options.Detail {
			appliedFilters["detail"] = true
		}
		summary.AppliedFilters = appliedFilters

		// Add statistics
		summary.FilteringStats = map[string]interface{}{
			"totalResources":    len(result.Resources),
			"filteredResources": len(summaryResources),
			"totalSteps":        originalTotalSteps,
			"filteredSteps":     filteredTotalSteps,
			"totalEvents":       originalTotalEvents,
			"filteredEvents":    filteredTotalEvents,
		}
	}

	return summary, nil
}

func matchesResourceFilter(resource fleet.EventsPerResource, resourceID, resourceKey string) bool {
	if resourceID != "" && resource.ResourceId != resourceID {
		return false
	}
	if resourceKey != "" && resource.ResourceKey != resourceKey {
		return false
	}
	return true
}

func matchesStepNameFilter(stepName string, stepNames []string) bool {
	if len(stepNames) == 0 {
		return true // No filter means include all
	}

	for _, filterName := range stepNames {
		if strings.EqualFold(stepName, filterName) {
			return true
		}
	}
	return false
}

func matchesTimeFilter(event fleet.WorkflowEvent, sinceTime, untilTime *time.Time) bool {
	if sinceTime != nil || untilTime != nil {
		eventTime, err := time.Parse(time.RFC3339, event.EventTime)
		if err != nil {
			// If we can't parse the time, include it to be safe
			return true
		}

		if sinceTime != nil && eventTime.Before(*sinceTime) {
			return false
		}
		if untilTime != nil && eventTime.After(*untilTime) {
			return false
		}
	}
	return true
}

// determineStepStatus analyzes events to determine the overall step status based on workflow event types
func determineStepStatus(events []fleet.WorkflowEvent) string {
	if len(events) == 0 {
		return "unknown"
	}

	hasStarted := false
	hasCompleted := false
	hasFailed := false

	// Look for specific workflow event types to determine status
	for _, event := range events {
		switch event.EventType {
		case "WorkflowStepStarted":
			hasStarted = true
		case "WorkflowStepCompleted":
			hasCompleted = true
		case "WorkflowStepFailed":
			hasFailed = true
		}
	}

	// Determine status based on priority: failed > completed > started > unknown
	if hasFailed {
		return "failed"
	}
	if hasCompleted {
		return "success"
	}
	if hasStarted {
		return "in-progress"
	}

	return "unknown"
}
