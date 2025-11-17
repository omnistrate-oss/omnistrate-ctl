package workflow

import (
	"crypto/sha256"
	"encoding/hex"
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

Use --detail to see full event details for each step. Duplicate events are automatically deduplicated to reduce output size.
Use --max-events to limit the number of unique events shown per event type (default: 3, use 0 for unlimited).`,
	Args: cobra.ExactArgs(1),
	RunE: getWorkflowEvents,
	Example: `  # Show workflow summary with step statuses
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id>

  # Show detailed events for all steps (with deduplication, max 3 per event type)
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail

  # Show detailed events with up to 5 events per event type
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail --max-events 5

  # Show all events without limiting by type
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail --max-events 0

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
	eventsCmd.Flags().Int("max-events", 3, "Maximum number of events to show per event type within each step (0 = unlimited)")
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
	maxEvents, _ := cmd.Flags().GetInt("max-events")
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
		MaxEvents:   maxEvents,
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
	MaxEvents   int
	Since       string
	Until       string
}

// DedupedWorkflowEvent represents a workflow event with deduplication info
type DedupedWorkflowEvent struct {
	EventTime   string `json:"eventTime"`
	EventType   string `json:"eventType"`
	Message     string `json:"message"`
	Occurrences int    `json:"occurrences,omitempty"`
	FirstSeen   string `json:"firstSeen,omitempty"`
	LastSeen    string `json:"lastSeen,omitempty"`
}

// DetailedStepWithDedupedEvents contains a step with deduplicated events
type DetailedStepWithDedupedEvents struct {
	StepName string                 `json:"stepName"`
	Events   []DedupedWorkflowEvent `json:"events"`
}

// WorkflowStepSummary represents a simplified view of a workflow step with status
type WorkflowStepSummary struct {
	StepName     string                         `json:"stepName"`
	Status       string                         `json:"status"`
	StartTime    string                         `json:"startTime,omitempty"`
	EndTime      string                         `json:"endTime,omitempty"`
	EventCount   int                            `json:"eventCount"`
	DetailedStep *DetailedStepWithDedupedEvents `json:"detailedStep,omitempty"`
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
				// Apply deduplication to reduce output size
				if options.Detail {
					dedupedEvents := deduplicateEvents(relevantEvents, options.MaxEvents)
					stepSummary.DetailedStep = &DetailedStepWithDedupedEvents{
						StepName: step.StepName,
						Events:   dedupedEvents,
					}
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

// deduplicateEvents takes a list of workflow events and deduplicates consecutive similar events
// Events are considered similar if they have the same eventType and message content
// maxEventsPerType limits the number of unique events per eventType, keeping the latest events (0 = unlimited)
func deduplicateEvents(events []fleet.WorkflowEvent, maxEventsPerType int) []DedupedWorkflowEvent {
	if len(events) == 0 {
		return []DedupedWorkflowEvent{}
	}

	// First pass: deduplicate events
	eventMap := make(map[string]*DedupedWorkflowEvent)
	eventOrder := []string{}

	for _, event := range events {
		// Create a hash key based on eventType and message content
		hashKey := hashEventContent(event.EventType, event.Message)

		if existing, found := eventMap[hashKey]; found {
			// Update existing event with last seen time and increment count
			existing.Occurrences++
			existing.LastSeen = event.EventTime
		} else {
			// New unique event
			dedupedEvent := DedupedWorkflowEvent{
				EventTime:   event.EventTime,
				EventType:   event.EventType,
				Message:     event.Message,
				Occurrences: 1,
				FirstSeen:   event.EventTime,
			}
			eventMap[hashKey] = &dedupedEvent
			eventOrder = append(eventOrder, hashKey)
		}
	}

	// Second pass: if maxEventsPerType is set, keep only the latest N events per type
	var dedupedEvents []DedupedWorkflowEvent
	if maxEventsPerType > 0 {
		// Group events by type and keep track of their order
		eventsByType := make(map[string][]struct {
			key   string
			index int
		})
		for i, key := range eventOrder {
			event := eventMap[key]
			eventsByType[event.EventType] = append(eventsByType[event.EventType], struct {
				key   string
				index int
			}{key, i})
		}

		// For each type, keep only the latest N events (last N in the order)
		keysToInclude := make(map[string]bool)
		for _, eventsOfType := range eventsByType {
			startIdx := 0
			if len(eventsOfType) > maxEventsPerType {
				startIdx = len(eventsOfType) - maxEventsPerType
			}
			for i := startIdx; i < len(eventsOfType); i++ {
				keysToInclude[eventsOfType[i].key] = true
			}
		}

		// Build result in original order, only including selected events
		for _, key := range eventOrder {
			if keysToInclude[key] {
				event := eventMap[key]
				if event.Occurrences > 1 {
					dedupedEvents = append(dedupedEvents, *event)
				} else {
					// For single occurrences, don't include the dedup fields
					dedupedEvents = append(dedupedEvents, DedupedWorkflowEvent{
						EventTime: event.EventTime,
						EventType: event.EventType,
						Message:   event.Message,
					})
				}
			}
		}
	} else {
		// No limit, include all events in original order
		for _, key := range eventOrder {
			event := eventMap[key]
			if event.Occurrences > 1 {
				dedupedEvents = append(dedupedEvents, *event)
			} else {
				// For single occurrences, don't include the dedup fields
				dedupedEvents = append(dedupedEvents, DedupedWorkflowEvent{
					EventTime: event.EventTime,
					EventType: event.EventType,
					Message:   event.Message,
				})
			}
		}
	}

	return dedupedEvents
}

// hashEventContent creates a hash of event type and message for deduplication
func hashEventContent(eventType, message string) string {
	h := sha256.New()
	h.Write([]byte(eventType + "|" + message))
	return hex.EncodeToString(h.Sum(nil))
}
