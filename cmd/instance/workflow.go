package instance

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

// DisplayWorkflowResourceDataWithSpinners renders deployment workflow progress for each resource.
func DisplayWorkflowResourceDataWithSpinners(ctx context.Context, token, instanceID, actionType string) error {
	return displayWorkflowResourceDataWithProgress(ctx, token, instanceID, actionType)
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
