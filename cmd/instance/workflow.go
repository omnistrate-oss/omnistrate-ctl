package instance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

// ResourceSpinner holds spinner information for a resource
type ResourceSpinner struct {
	ResourceName string
	Spinner      *ysmrr.Spinner
}

// displayWorkflowResourceDataWithSpinners creates individual spinners for each resource and updates them dynamically
func displayWorkflowResourceDataWithSpinners(ctx context.Context, token, instanceID, actionType string) error {
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
	var resourceSpinners []ResourceSpinner
	
	// Function to create or update spinners for each resource
	createOrUpdateSpinners := func(resourcesData []dataaccess.ResourceWorkflowData) {
		// If this is the first time, create spinners for each resource
		if len(resourceSpinners) == 0 {
			for _, resourceData := range resourcesData {
				spinner := sm.AddSpinner(fmt.Sprintf("%s: Initializing...", resourceData.ResourceName))
				resourceSpinners = append(resourceSpinners, ResourceSpinner{
					ResourceName: resourceData.ResourceName,
					Spinner:      spinner,
				})
			}
			sm.Start()
		}
		
		// Update each resource spinner with current status
		for i, resourceData := range resourcesData {
			if i < len(resourceSpinners) {
				// Get status icons for each category
				bootstrapEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Bootstrap, "bootstrap", &resourceData)
				bootstrapIcon := getEventStatusIconFromType(bootstrapEventType)
				
				storageEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Storage, "storage", &resourceData)
				storageIcon := getEventStatusIconFromType(storageEventType)
				
				networkEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Network, "network", &resourceData)
				networkIcon := getEventStatusIconFromType(networkEventType)
				
				computeEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Compute, "compute", &resourceData)
				computeIcon := getEventStatusIconFromType(computeEventType)

				deploymentEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Deployment, "deployment", &resourceData)
				deploymentIcon := getEventStatusIconFromType(deploymentEventType)

				monitoringEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Monitoring, "monitoring", &resourceData)
				monitoringIcon := getEventStatusIconFromType(monitoringEventType)

				// Create dynamic message for this resource
				message := fmt.Sprintf("%s - Bootstrap: %s | Storage: %s | Network: %s | Compute: %s | Deployment: %s | Monitoring: %s",
					resourceData.ResourceName,
					bootstrapIcon,
					storageIcon,
					networkIcon,
					computeIcon,
					deploymentIcon,
					monitoringIcon)

				// Update spinner message
				resourceSpinners[i].Spinner.UpdateMessage(message)

				// Check resource status and update spinner accordingly
				if hasFailedEvent(bootstrapEventType, storageEventType, networkEventType, computeEventType, deploymentEventType, monitoringEventType) {
					// Error spinner if any category failed
					resourceSpinners[i].Spinner.Error()
				} else if allEventsCompleted(bootstrapEventType, storageEventType, networkEventType, computeEventType, deploymentEventType, monitoringEventType) {
					// Complete spinner if all categories are completed
					resourceSpinners[i].Spinner.Complete()
				}
			}
		}
	}
	
	// Function to complete spinners when deployment is done
	completeSpinners := func(resourcesData []dataaccess.ResourceWorkflowData, workflowInfo *dataaccess.WorkflowInfo) {
		for i := range resourcesData {
			if i < len(resourceSpinners) {
				if strings.ToLower(workflowInfo.WorkflowStatus) == "success" {
					resourceSpinners[i].Spinner.Complete()
				} else {
					resourceSpinners[i].Spinner.Error()
				}
			}
		}
		sm.Stop()
	}
	
	// Function to fetch and display current workflow status for all resources
	displayCurrentStatus := func() (bool, error) {
		// Get workflow events for all resources in the instance
		resourcesData, workflowInfo, err := dataaccess.GetDebugEventsForAllResources(
			ctx, token,
			instance.ServiceId,
			instance.ServiceEnvironmentId,
			instanceID,
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
		
		// If workflow is complete, complete all spinners and stop
		if isWorkflowComplete {
			completeSpinners(resourcesData, workflowInfo)
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
			// Handle error but continue polling
		} else if isComplete {
			break
		}

		// Wait for the next tick
		<-ticker.C
	}

	return nil
}


// getHighestPriorityEventType checks all events in a category and returns the highest priority event type
func getHighestPriorityEventType(events []dataaccess.CustomWorkflowEvent, categoryName string, resourceData *dataaccess.ResourceWorkflowData) string {
	// Case 1: No events at all - need to determine if step is not started or not applicable
	if len(events) == 0 {
		// For now, we'll assume common categories are applicable for most resources
		// This could be enhanced later with more detailed workflow step information
		commonCategories := map[string]bool{
			"bootstrap":  true,
			"deployment": true,
			"compute":    true,
		}
		
		if commonCategories[categoryName] {
			return "not_started" // Likely workflow step exists but not started yet
		} else {
			return "not_applicable" // Less common step, might not apply to this resource
		}
	}

	// Check in priority order for known event types
	// 1. First check for failed events (highest priority)
	for _, event := range events {
		if event.EventType == "WorkflowStepFailed" {
			return event.EventType
		}
	}

	// 2. Then check for completed events
	for _, event := range events {
		if event.EventType == "WorkflowStepCompleted" {
			return event.EventType
		}
	}

	// 3. Then check for debug events
	for _, event := range events {
		if event.EventType == "WorkflowStepDebug" {
			return event.EventType
		}
	}

	// 4. Then check for started events
	for _, event := range events {
		if event.EventType == "WorkflowStepStarted" {
			return event.EventType
		}
	}
	
	// 5. If none of the above known types, return the last event type as fallback
	return "unknown"
	
}

// getEventStatusIconFromType returns an appropriate icon based on event type string
func getEventStatusIconFromType(eventType string) string {
	switch eventType {
	case "WorkflowStepFailed":
		return "❌"
	case "WorkflowStepCompleted":
		return "✅"
	case "WorkflowStepDebug", "WorkflowStepStarted":
		return "🔄"
	case "not_started":
		return "⚪" // White circle for not started
	case "not_applicable":
		return "\033[90m➖\033[0m" // Gray colored dash for not applicable
	default:
		// Handle unknown event types (starts with "unknown:")
		if strings.HasPrefix(eventType, "unknown") {
			return "❓" // Question mark for unknown event types
		}
		return "🟡" // Yellow circle for other cases
	}
}

// hasFailedEvent checks if any of the event types indicates a failure
func hasFailedEvent(eventTypes ...string) bool {
	for _, eventType := range eventTypes {
		if eventType == "WorkflowStepFailed" {
			return true
		}
		// Also treat unknown event types as potential failures to be safe
		if strings.HasPrefix(eventType, "unknown") && strings.Contains(strings.ToLower(eventType), "fail") {
			return true
		}
	}
	return false
}

// allEventsCompleted checks if all non-empty event types are completed
func allEventsCompleted(eventTypes ...string) bool {
	hasAtLeastOneEvent := false
	for _, eventType := range eventTypes {
		// Skip not_started and not_applicable categories (they don't affect completion status)
		if eventType == "not_started" || eventType == "not_applicable" || eventType == "" {
			continue
		}
		
		hasAtLeastOneEvent = true
		if eventType != "WorkflowStepCompleted" {
			return false
		}
	}
	return hasAtLeastOneEvent
}
