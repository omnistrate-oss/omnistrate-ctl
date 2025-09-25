package instance

import (
	"context"
	"fmt"
	"sort"
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
			// Dynamically get all available categories and their status
			categoryStatuses := getDynamicCategoryStatuses(resourceData.EventsByCategory)
			
			// Build dynamic message with available categories
			   var messageParts []string
			   for _, categoryStatus := range categoryStatuses {
				   messageParts = append(messageParts, fmt.Sprintf("%s: %s", categoryStatus.Name, categoryStatus.Icon))
			   }

			// Create dynamic message for this resource
			message := fmt.Sprintf("%s - %s", resourceData.ResourceName, strings.Join(messageParts, " | "))

			// Update spinner message
			resourceSpinners[i].Spinner.UpdateMessage(message)

				// Use getResourceStatusFromEvents for status
				resourceStatus := getResourceStatusFromEvents(resourceData.EventsByCategory)
				switch resourceStatus {
				case "ResourceStatusFailed":
					resourceSpinners[i].Spinner.Error()
				case "ResourceStatusCompleted":
					resourceSpinners[i].Spinner.Complete()
				}
		}
	}

	// Function to complete spinners when deployment is done
	completeSpinners := func(resourcesData []dataaccess.ResourceWorkflowData, workflowInfo *dataaccess.WorkflowInfo) bool {
		hasFailures := false
		
		for i, resourceData := range resourcesData {
			// First check resource status from DescribeWorkflow API if available
			resourceStatus := ""
			if resourceData.ResourceStatus != nil {
				resourceStatus = mapResourceStatus(*resourceData.ResourceStatus)
			} else {
				// Fall back to getting status from events
				resourceStatus = strings.ToLower(getResourceStatusFromEvents(resourceData.EventsByCategory))
			}
			
			switch resourceStatus {
			case "ResourceStatusCompleted":
				resourceSpinners[i].Spinner.Complete()
			case "ResourceStatusFailed":
				resourceSpinners[i].Spinner.Error()
				hasFailures = true
			case "ResourceStatusPending","ResourceStatusRunning":
				// For workflow completion, determine final status
				// If workflow overall is successful, complete pending resources
				if strings.ToLower(workflowInfo.WorkflowStatus) == "success" {
					resourceSpinners[i].Spinner.Complete()
				} else {
					resourceSpinners[i].Spinner.Error()
					hasFailures = true
				}
			default:
				// Fallback to overall workflow status
				if strings.ToLower(workflowInfo.WorkflowStatus) == "success" {
					resourceSpinners[i].Spinner.Complete()
				} else {
					resourceSpinners[i].Spinner.Error()
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
				  // (Removed unused categoryStatuses and allEventTypes)
			
			resourceStatus := ""
			// Also check the resource status from DescribeWorkflow API if available
			if resourceData.ResourceStatus != nil {
				resourceStatus = mapResourceStatus(*resourceData.ResourceStatus)
				
			}

			// Use getResourceStatusFromEvents for failure detection
			resourceStatusFromEvents := getResourceStatusFromEvents(resourceData.EventsByCategory)
			if resourceStatus == "ResourceStatusFailed" || resourceStatusFromEvents == "ResourceStatusFailed" {
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
// getHighestPriorityEventType checks all events in a category and returns the highest priority event type
func getHighestPriorityEventType(events []dataaccess.CustomWorkflowEvent, categoryName string) string {
	   // Case 1: No events at all - treat as pending
	   if len(events) == 0 {
		   return "WorkflowStepPending"
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
	return "WorkflowStepUnknown"

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
	case "WorkflowStepPending":
		return "🟡"
	default:
		return "🟡"
	}
}


// CategoryStatus represents the status of a workflow category
type CategoryStatus struct {
	Name      string
	EventType string
	Icon      string
	Order     int
}

// getDynamicCategoryStatuses extracts all available categories and their statuses
func getDynamicCategoryStatuses(eventsByCategory *dataaccess.WorkflowEventsByCategory) []CategoryStatus {
	// Define the preferred order for categories
	categoryOrder := map[string]int{
		"Bootstrap":  1,
		"Storage":    2,
		"Network":    3,
		"Compute":    4,
		"Deployment": 5,
		"Monitoring": 6,
		"Other":      7,
	}

	var statuses []CategoryStatus
	
	// Use reflection-like approach to get all categories dynamically
	if eventsByCategory != nil {
		categories := []struct {
			name   string
			events []dataaccess.CustomWorkflowEvent
		}{
			{"Bootstrap", eventsByCategory.Bootstrap},
			{"Storage", eventsByCategory.Storage},
			{"Network", eventsByCategory.Network},
			{"Compute", eventsByCategory.Compute},
			{"Deployment", eventsByCategory.Deployment},
			{"Monitoring", eventsByCategory.Monitoring},
			{"Other", eventsByCategory.Other},
		}

		// Track which categories have actual events
		for _, category := range categories {
			// Check if this category has events
			if len(category.events) > 0 {
				eventType := getHighestPriorityEventType(category.events, strings.ToLower(category.name))
				icon := getEventStatusIconFromType(eventType)
				order := categoryOrder[category.name]
				if order == 0 {
					order = 999 // Put unknown categories at the end
				}

				statuses = append(statuses, CategoryStatus{
					Name:      category.name,
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
func getResourceStatusFromEvents(eventsByCategory *dataaccess.WorkflowEventsByCategory) string {
	if eventsByCategory == nil {
		return "ResourceStatusPending"
	}

	// Check all categories for their highest priority event types
	categories := []struct {
		name   string
		events []dataaccess.CustomWorkflowEvent
	}{
		{"Bootstrap", eventsByCategory.Bootstrap},
		{"Storage", eventsByCategory.Storage},
		{"Network", eventsByCategory.Network},
		{"Compute", eventsByCategory.Compute},
		{"Deployment", eventsByCategory.Deployment},
		{"Monitoring", eventsByCategory.Monitoring},
		{"Other", eventsByCategory.Other},
	}

	hasCompleted := false
	hasFailed := false
	hasEvents := false
	categoriesWithEvents := 0
	completedCategories := 0

	for _, category := range categories {
		if len(category.events) > 0 {
			hasEvents = true
			categoriesWithEvents++
			eventType := getHighestPriorityEventType(category.events, strings.ToLower(category.name))
			
			switch eventType {
			case "WorkflowStepStarted":
				hasEvents = true
			case "WorkflowStepCompleted":
				hasCompleted = true
				completedCategories++
			case "WorkflowStepFailed":
				hasFailed = true
			}
		}
	}

	// Determine overall status with improved logic for partial category availability
	if hasFailed {
		return "ResourceStatusFailed"
	}
	
	// If all categories that have events are completed
	if hasCompleted && completedCategories == categoriesWithEvents {
		return "ResourceStatusCompleted"
	}

	if hasEvents {
		return "ResourceStatusRunning"
	}

	return "ResourceStatusPending"
}


// mapResourceStatus maps API workflow status values to internal resource status constants
func mapResourceStatus(apiStatus string) string {
   switch strings.ToLower(apiStatus) {
   case "success":
	   return "ResourceStatusCompleted"
   case "failed", "error", "cancelled":
	   return "ResourceStatusFailed"
   case "pending":
	   return "ResourceStatusPending"
   case "running":
	   return "ResourceStatusRunning"
   default:
	   return "ResourceStatusPending"
   }
}