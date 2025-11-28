package deploymentcell

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var workflowEventsCmd = &cobra.Command{
	Use:   "events [deployment-cell-id] [workflow-id]",
	Short: "Get events for a deployment cell workflow",
	Long:  "Retrieve debug events for a specific deployment cell workflow, organized by workflow steps.",
	Args:  cobra.ExactArgs(2),
	RunE:  getDeploymentCellWorkflowEvents,
}

type WorkflowEventItem struct {
	StepName  string `json:"stepName" table:"Step Name"`
	EventTime string `json:"eventTime" table:"Event Time"`
	EventType string `json:"eventType" table:"Event Type"`
	Message   string `json:"message" table:"Message"`
	Error     string `json:"error,omitempty" table:"Error"`
	Metadata  string `json:"metadata,omitempty" table:"Metadata"`
}

func getDeploymentCellWorkflowEvents(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	deploymentCellID := args[0]
	workflowID := args[1]

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	result, err := dataaccess.GetDeploymentCellWorkflowEvents(ctx, token, deploymentCellID, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell workflow events: %w", err)
	}

	if result == nil {
		return fmt.Errorf("workflow events not found")
	}

	outputFormat, _ := cmd.Flags().GetString("output")

	// For JSON output, return the raw result
	if outputFormat == "json" {
		return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
	}

	// For table and text output, format events with separators
	if len(result.EventsPerWorkflowStep) == 0 {
		fmt.Println("No events found for this workflow")
		return nil
	}

	events := formatWorkflowEventsWithSeparators(result)

	if len(events) == 0 {
		fmt.Println("No events found for this workflow")
		return nil
	}

	return utils.PrintTextTableJsonArrayOutput(outputFormat, events)
}

func formatWorkflowEventsWithSeparators(result *openapiclientfleet.GetDeploymentCellWorkflowEventsResult) []interface{} {
	var events []interface{}

	// Sort steps by the earliest event time in each step
	sortedSteps := sortStepsByTime(result.EventsPerWorkflowStep)

	for i, step := range sortedSteps {
		stepName := step.GetStepName()

		// Add separator row before each step (except the first one)
		if i > 0 {
			events = append(events, WorkflowEventItem{
				StepName:  strings.Repeat("─", 32),
				EventTime: strings.Repeat("─", 20),
				EventType: strings.Repeat("─", 21),
				Message:   strings.Repeat("─", 70),
				Error:     strings.Repeat("─", 40),
				Metadata:  strings.Repeat("─", 40),
			})
		}

		// Add events for this step
		for _, event := range step.GetEvents() {
			// Format metadata if present
			metadata := formatMetadata(event.AdditionalProperties)

			// Get error if present
			errorMsg := ""
			if event.Error != nil && *event.Error != "" {
				errorMsg = *event.Error
			}

			events = append(events, WorkflowEventItem{
				StepName:  stepName,
				EventTime: event.GetEventTime(),
				EventType: event.GetEventType(),
				Message:   event.GetMessage(),
				Error:     errorMsg,
				Metadata:  metadata,
			})
		}
	}

	return events
}

func sortStepsByTime(steps []openapiclientfleet.DeploymentCellEventsPerWorkflowStep) []openapiclientfleet.DeploymentCellEventsPerWorkflowStep {
	// Create a copy to avoid modifying the original
	sortedSteps := make([]openapiclientfleet.DeploymentCellEventsPerWorkflowStep, len(steps))
	copy(sortedSteps, steps)

	// Sort by the earliest event time in each step
	sort.Slice(sortedSteps, func(i, j int) bool {
		timeI := getEarliestEventTime(sortedSteps[i])
		timeJ := getEarliestEventTime(sortedSteps[j])

		if timeI.IsZero() || timeJ.IsZero() {
			return false
		}

		return timeI.Before(timeJ)
	})

	return sortedSteps
}

func getEarliestEventTime(step openapiclientfleet.DeploymentCellEventsPerWorkflowStep) time.Time {
	events := step.GetEvents()
	if len(events) == 0 {
		return time.Time{}
	}

	earliestTime, err := time.Parse(time.RFC3339, events[0].GetEventTime())
	if err != nil {
		return time.Time{}
	}

	for _, event := range events[1:] {
		eventTime, err := time.Parse(time.RFC3339, event.GetEventTime())
		if err != nil {
			continue
		}
		if eventTime.Before(earliestTime) {
			earliestTime = eventTime
		}
	}

	return earliestTime
}

func formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}

	// Convert metadata to JSON string for display
	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return string(jsonBytes)
}
