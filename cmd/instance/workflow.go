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
func displayWorkflowResourceDataWithSpinners(ctx context.Context, token, instanceID string) error {
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
	var sm ysmrr.SpinnerManager
	sm = ysmrr.NewSpinnerManager()
	
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
				bootstrapEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Bootstrap)
				bootstrapIcon := getEventStatusIconFromType(bootstrapEventType)
				
				storageEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Storage)
				storageIcon := getEventStatusIconFromType(storageEventType)
				
				networkEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Network)
				networkIcon := getEventStatusIconFromType(networkEventType)
				
				computeEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Compute)
				computeIcon := getEventStatusIconFromType(computeEventType)

				deploymentEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Deployment)
				deploymentIcon := getEventStatusIconFromType(deploymentEventType)

				monitoringEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Monitoring)
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

				// Check if any category has failed
				if bootstrapEventType == "WorkflowStepFailed" || bootstrapEventType == "WorkflowFailed" ||
				   storageEventType == "WorkflowStepFailed" || storageEventType == "WorkflowFailed" ||
				   networkEventType == "WorkflowStepFailed" || networkEventType == "WorkflowFailed" ||
				   computeEventType == "WorkflowStepFailed" || computeEventType == "WorkflowFailed" ||
				   deploymentEventType == "WorkflowStepFailed" || deploymentEventType == "WorkflowFailed" ||
				   monitoringEventType == "WorkflowStepFailed" || monitoringEventType == "WorkflowFailed" {
					// Error spinner if any category failed
					resourceSpinners[i].Spinner.Error()
				} else if bootstrapEventType == "WorkflowStepCompleted" && 
						  storageEventType == "WorkflowStepCompleted" && 
						  networkEventType == "WorkflowStepCompleted" && 
						  computeEventType == "WorkflowStepCompleted" && 
						  deploymentEventType == "WorkflowStepCompleted" && 
						  monitoringEventType == "WorkflowStepCompleted" {
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

	// Initial display
	isComplete, err := displayCurrentStatus()
	if err != nil {
		return err
	}

	// If workflow is already complete, don't start polling
	if isComplete {
		return nil
	}

	// Start polling every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		isComplete, err := displayCurrentStatus()
		if err != nil {
			// Handle error but continue polling
			continue
		}

		// Stop polling when workflow is complete
		if isComplete {
			break
		}
	}

	return nil
}


// getHighestPriorityEventType checks all events in a category and returns the highest priority event type
func getHighestPriorityEventType(events []dataaccess.CustomWorkflowEvent) string {
	if len(events) == 0 {
		return ""
	}

	// Check in priority order
	// 1. First check for failed events (highest priority)
	for _, event := range events {
		if event.EventType == "WorkflowStepFailed" || event.EventType == "WorkflowFailed" {
			return event.EventType
		}
	}

	// 2. Then check for completed events
	for _, event := range events {
		if event.EventType == "WorkflowStepCompleted" {
			return event.EventType
		}
	}

	// 3. Then check for debug or started events
	for _, event := range events {
		if event.EventType == "WorkflowStepDebug"  {
			return event.EventType
		}
	}

	// 4. Then check for started events
	for _, event := range events {
		if  event.EventType == "WorkflowStepStarted" {
			return event.EventType
		}
	}
	// 5. If none of the above, return the last event type
	if len(events) > 0 {
		return events[len(events)-1].EventType
	}

	return ""
}

// getEventStatusIconFromType returns an appropriate icon based on event type string
func getEventStatusIconFromType(eventType string) string {
	switch eventType {
	case "WorkflowStepFailed", "WorkflowFailed":
		return "❌"
	case "WorkflowStepCompleted":
		return "✅"
	case "WorkflowStepDebug", "WorkflowStepStarted":
		return "🔄"
	default:
		return "🟡"
	}
}
