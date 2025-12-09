package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

// ResourceSpinner holds spinner information for a resource
type ResourceSpinner struct {
	ResourceId   string
	ResourceKey  string
	ResourceName string
	Spinner      *ysmrr.Spinner
}

// displayWorkflowResourceDataWithSpinners creates individual spinners for each resource and updates them dynamically
func DisplayWorkflowResourceDataWithSpinners(ctx context.Context, token, instanceID, actionType string) error {
	// Search for the instance to get service details
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		return fmt.Errorf("instance not found")
	}

	instance := searchRes.ResourceInstanceResults[0]

	// Initialize spinner manager
	sm := ysmrr.NewSpinnerManager()

	// Track resource spinners
	resourceSpinners := make(map[string]ResourceSpinner)

	// Function to create or update spinners for each resource
	createOrUpdateSpinners := func(resourcesData []dataaccess.ResourceWorkflowDebugEvents) {
		// If this is the first time, create spinners for each resource

		for _, resourceData := range resourcesData {
			if _, exists := resourceSpinners[resourceData.ResourceID]; !exists {
				spinner := sm.AddSpinner(fmt.Sprintf("%s: Initializing...", resourceData.ResourceName))
				resourceSpinners[resourceData.ResourceID] = ResourceSpinner{
					ResourceId:   resourceData.ResourceID,
					ResourceKey:  resourceData.ResourceKey,
					ResourceName: resourceData.ResourceName,
					Spinner:      spinner,
				}
			}
		}
		if !sm.Running() {
			sm.Start()
		}

		// Update each resource spinner with current status
		for _, resourceData := range resourcesData {
			// Dynamically get all available workflow steps and their status
			workflowStepStatuses := getDynamicWorkflowStepStatuses(resourceData.EventsByWorkflowStep)

			// Build dynamic message with available workflow steps
			var messageParts []string
			for _, workflowStepStatus := range workflowStepStatuses {
				messageParts = append(messageParts, fmt.Sprintf("%s: %s", workflowStepStatus.Name, workflowStepStatus.Icon))
			}

			// Create dynamic message for this resource
			message := fmt.Sprintf("%s - %s", resourceData.ResourceName, strings.Join(messageParts, " | "))

			// Update spinner message
			resourceSpinners[resourceData.ResourceID].Spinner.UpdateMessage(message)

			// Use getResourceStatusFromEvents for status
			resourceStatus := getResourceStatusFromEvents(resourceData.EventsByWorkflowStep)
			switch resourceStatus {
			case model.ResourceStatusFailed:
				resourceSpinners[resourceData.ResourceID].Spinner.Error()
			case model.ResourceStatusCompleted:
				resourceSpinners[resourceData.ResourceID].Spinner.Complete()
			}
		}
	}

	// Function to complete spinners when deployment is done
	completeSpinners := func(resourcesData []dataaccess.ResourceWorkflowDebugEvents, workflowInfo *dataaccess.WorkflowInfo) bool {
		hasFailures := false
		workflowFailed := strings.ToLower(workflowInfo.WorkflowStatus) == "failed" || strings.ToLower(workflowInfo.WorkflowStatus) == "cancelled"
		workflowSucceeded := strings.ToLower(workflowInfo.WorkflowStatus) == "success"
		for _, resourceData := range resourcesData {
			var resourceStatus model.WorkflowStatus
			if resourceData.WorkflowStatus != nil {
				resourceStatus = mapResourceStatus(*resourceData.WorkflowStatus)
			} else {
				// Fallback: parse string status from events
				resourceStatus = model.ParseWorkflowStatus(string(getResourceStatusFromEvents(resourceData.EventsByWorkflowStep)))
			}
			// Track if any resource failed
			if resourceStatus == model.WorkflowStatusFailed {
				hasFailures = true
			}
			// Set spinner state based on resource and workflow status
			switch resourceStatus {
			case model.WorkflowStatusCompleted:
				resourceSpinners[resourceData.ResourceID].Spinner.Complete()
			case model.WorkflowStatusFailed:
				resourceSpinners[resourceData.ResourceID].Spinner.Error()
			default:
				// For any non-completed/non-failed resource, force final state based on workflow
				if workflowSucceeded {
					resourceSpinners[resourceData.ResourceID].Spinner.Complete()
				} else if workflowFailed {
					resourceSpinners[resourceData.ResourceID].Spinner.Error()
					hasFailures = true
				}
			}
		}
		sm.Stop()
		return hasFailures
	}

	// Function to fetch and display current workflow status for all resources
	displayCurrentStatus := func() (bool, error) {
		// Get workflow events for all resources in the instance with enhanced status
		resourcesData, workflowInfo, err := dataaccess.GetDebugEventsForAllResources(
			ctx, token,
			instance.ServiceId,
			instance.ServiceEnvironmentId,
			instanceID,
			true,
			actionType,
		)
		if err != nil {
			return false, err
		}

		if workflowInfo == nil {
			return true, nil // Stop polling if no workflow data
		}

		// Check if workflow is complete
		isWorkflowComplete := strings.ToLower(workflowInfo.WorkflowStatus) == "success" ||
			strings.ToLower(workflowInfo.WorkflowStatus) == "failed" ||
			strings.ToLower(workflowInfo.WorkflowStatus) == "cancelled"

		if len(resourcesData) == 0 {
			return isWorkflowComplete, nil
		}

		// Create or update spinners for each resource
		createOrUpdateSpinners(resourcesData)

		// Check for resource-level failures even if workflow is still running
		for _, resourceData := range resourcesData {
			var resourceStatus model.WorkflowStatus
			if resourceData.WorkflowStatus != nil {
				resourceStatus = mapResourceStatus(*resourceData.WorkflowStatus)
			} else {
				// Use ResourceStatus as WorkflowStatus for fallback (string types)
				resourceStatus = model.WorkflowStatus(getResourceStatusFromEvents(resourceData.EventsByWorkflowStep))
			}
			// Use getResourceStatusFromEvents for failure detection (legacy string fallback)
			resourceStatusFromEvents := getResourceStatusFromEvents(resourceData.EventsByWorkflowStep)
			if resourceStatus == model.WorkflowStatusFailed || resourceStatusFromEvents == model.ResourceStatusFailed {
				sm.Stop()
				return false, fmt.Errorf("for resource %s", resourceData.ResourceName)
			}
		}

		// If workflow is complete, complete all spinners and stop
		if isWorkflowComplete {
			hasFailures := completeSpinners(resourcesData, workflowInfo)
			if hasFailures || strings.ToLower(workflowInfo.WorkflowStatus) == "failed" {
				return false, fmt.Errorf("with status: %s", workflowInfo.WorkflowStatus)
			}
			return true, nil
		}

		return false, nil
	}

	// Start polling every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		isComplete, err := displayCurrentStatus()
		if err != nil {
			// If there's an error from displayCurrentStatus, return it
			return err
		} else if isComplete {
			break
		}
		// Wait for the next tick
		<-ticker.C
	}
	return nil
}

// getHighestPriorityEventType checks all events in a workflowStep and returns the highest priority event type
func getHighestPriorityEventType(events []dataaccess.DebugEvent) string {
	// Case 1: No events at all - treat as pending
	if len(events) == 0 {
		return string(model.WorkflowStepPending)
	}

	// Check in priority order for known event types
	// 1. First check for failed events (highest priority)
	for _, event := range events {
		if event.EventType == string(model.WorkflowStepFailed) {
			return event.EventType
		}
	}

	// 2. Then check for completed events
	for _, event := range events {
		if event.EventType == string(model.WorkflowStepCompleted) {
			return event.EventType
		}
	}

	// 3. Then check for debug events
	for _, event := range events {
		if event.EventType == string(model.WorkflowStepDebug) {
			return event.EventType
		}
	}

	// 4. Then check for started events
	for _, event := range events {
		if event.EventType == string(model.WorkflowStepStarted) {
			return event.EventType
		}
	}

	// 5. If none of the above known types, return the unknown event type
	return string(model.WorkflowStepUnknown)
}

// getEventStatusIconFromType returns an appropriate icon based on event type string
func getEventStatusIconFromType(eventType string) string {
	switch eventType {
	case string(model.WorkflowStepFailed):
		return "❌"
	case string(model.WorkflowStepCompleted):
		return "✅"
	case string(model.WorkflowStepDebug), string(model.WorkflowStepStarted):
		return "🔄"
	case string(model.WorkflowStepPending):
		return "🟡"
	default:
		return "🟡"
	}
}

// WorkflowStepStatus represents the status of a workflow step
type WorkflowStepStatus struct {
	Name      string
	EventType string
	Icon      string
	Order     int
}

// getDynamicWorkflowStepStatuses extracts all available workflow steps and their statuses
func getDynamicWorkflowStepStatuses(eventsByWorkflowStep *dataaccess.DebugEventsByWorkflowSteps) []WorkflowStepStatus {
	// Define the preferred order for workflow steps
	workflowStepOrder := map[model.WorkflowStep]int{
		model.WorkflowStepBootstrap:                   1,
		model.WorkflowStepStorage:                     2,
		model.WorkflowStepNetwork:                     3,
		model.WorkflowStepCompute:                     4,
		model.WorkflowStepDeployment:                  5,
		model.WorkflowStepMonitoring:                  6,
		model.WorkflowStep(model.WorkflowStepUnknown): 7,
	}

	var statuses []WorkflowStepStatus

	// Use reflection-like approach to get all workflowSteps dynamically
	if eventsByWorkflowStep != nil {
		workflowSteps := []struct {
			name   model.WorkflowStep
			events []dataaccess.DebugEvent
		}{
			{model.WorkflowStepBootstrap, eventsByWorkflowStep.Bootstrap},
			{model.WorkflowStepStorage, eventsByWorkflowStep.Storage},
			{model.WorkflowStepNetwork, eventsByWorkflowStep.Network},
			{model.WorkflowStepCompute, eventsByWorkflowStep.Compute},
			{model.WorkflowStepDeployment, eventsByWorkflowStep.Deployment},
			{model.WorkflowStepMonitoring, eventsByWorkflowStep.Monitoring},
			{model.WorkflowStep(model.WorkflowStepUnknown), eventsByWorkflowStep.Unknown},
		}
		// Track which workflow steps have actual events
		for _, step := range workflowSteps {
			// Check if this step has events
			if len(step.events) > 0 {
				eventType := getHighestPriorityEventType(step.events)
				icon := getEventStatusIconFromType(eventType)
				order := workflowStepOrder[step.name]
				if order == 0 {
					order = 999 // Put unknown categories at the end
				}

				statuses = append(statuses, WorkflowStepStatus{
					Name:      string(step.name),
					EventType: eventType,
					Icon:      icon,
					Order:     order,
				})
			}
		}

	}

	// Sort by order
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Order < statuses[j].Order
	})

	return statuses
}

// getResourceStatusFromEvents determines the overall status of a resource based on its events across all categories
func getResourceStatusFromEvents(eventsByWorkflowStep *dataaccess.DebugEventsByWorkflowSteps) model.ResourceStatus {
	if eventsByWorkflowStep == nil {
		return model.ResourceStatusPending
	}

	// Check all workflowSteps for their highest priority event types
	workflowSteps := []struct {
		name   model.WorkflowStep
		events []dataaccess.DebugEvent
	}{
		{model.WorkflowStepBootstrap, eventsByWorkflowStep.Bootstrap},
		{model.WorkflowStepStorage, eventsByWorkflowStep.Storage},
		{model.WorkflowStepNetwork, eventsByWorkflowStep.Network},
		{model.WorkflowStepCompute, eventsByWorkflowStep.Compute},
		{model.WorkflowStepDeployment, eventsByWorkflowStep.Deployment},
		{model.WorkflowStepMonitoring, eventsByWorkflowStep.Monitoring},
	}

	hasCompleted := false
	hasFailed := false
	hasEvents := false
	workflowStepsWithEvents := 0
	completedWorkflowSteps := 0

	for _, step := range workflowSteps {
		if len(step.events) > 0 {
			hasEvents = true
			workflowStepsWithEvents++
			eventType := getHighestPriorityEventType(step.events)

			switch model.WorkflowStepEventType(eventType) {
			case model.WorkflowStepStarted:
				hasEvents = true
			case model.WorkflowStepCompleted:
				hasCompleted = true
				completedWorkflowSteps++
			case model.WorkflowStepFailed:
				hasFailed = true
			}
		}
	}

	// Determine overall status with improved logic for partial workflowStep availability
	if hasFailed {
		return model.ResourceStatusFailed
	}

	// If all workflowSteps that have events are completed
	if hasCompleted && completedWorkflowSteps == workflowStepsWithEvents {
		return model.ResourceStatusCompleted
	}

	if hasEvents {
		return model.ResourceStatusRunning
	}

	return model.ResourceStatusPending
}

// mapResourceStatus maps API workflow status values to WorkflowStatus enum
func mapResourceStatus(apiStatus string) model.WorkflowStatus {
	return model.ParseWorkflowStatus(apiStatus)
}
