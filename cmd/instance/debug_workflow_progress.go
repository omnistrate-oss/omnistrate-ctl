package instance

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

type ResourceProgress struct {
	Percent        int    `json:"percent"`
	Status         string `json:"status,omitempty"`
	CompletedSteps int    `json:"completedSteps,omitempty"`
	TotalSteps     int    `json:"totalSteps,omitempty"`
}

func attachWorkflowProgress(ctx context.Context, token, serviceID, environmentID, instanceID string, plan *PlanDAG) {
	if plan == nil {
		return
	}

	resourcesData, workflowInfo, err := dataaccess.GetDebugEventsForAllResources(ctx, token, serviceID, environmentID, instanceID, true)
	if err != nil {
		plan.Errors = append(plan.Errors, fmt.Sprintf("workflow progress incomplete: %v", err))
	}

	if len(resourcesData) == 0 {
		return
	}

	if workflowInfo != nil && workflowInfo.WorkflowID != "" {
		plan.WorkflowID = workflowInfo.WorkflowID
	}

	plan.ProgressByID = map[string]ResourceProgress{}
	plan.ProgressByKey = map[string]ResourceProgress{}
	plan.ProgressByName = map[string]ResourceProgress{}

	for _, resource := range resourcesData {
		progress := computeResourceProgress(resource.EventsByWorkflowStep, resource.WorkflowStatus, workflowInfo)
		if resource.ResourceID != "" {
			plan.ProgressByID[resource.ResourceID] = progress
		}
		if resource.ResourceKey != "" {
			plan.ProgressByKey[resource.ResourceKey] = progress
		}
		if resource.ResourceName != "" {
			plan.ProgressByName[resource.ResourceName] = progress
		}
	}
}

func computeResourceProgress(events *dataaccess.DebugEventsByWorkflowSteps, workflowStatus *string, workflowInfo *dataaccess.WorkflowInfo) ResourceProgress {
	progress := ResourceProgress{}
	status := strings.ToLower(safeStatus(workflowStatus, workflowInfo))

	total := 0
	completed := 0
	hasFailed := false
	hasRunning := false

	if events != nil {
		steps := []struct {
			events []dataaccess.DebugEvent
		}{
			{events: events.Bootstrap},
			{events: events.Storage},
			{events: events.Network},
			{events: events.Compute},
			{events: events.Deployment},
			{events: events.Monitoring},
			{events: events.Unknown},
		}
		for _, step := range steps {
			if len(step.events) == 0 {
				continue
			}
			total++
			eventType := getHighestPriorityEventType(step.events)
			switch model.WorkflowStepEventType(eventType) {
			case model.WorkflowStepFailed:
				hasFailed = true
			case model.WorkflowStepCompleted:
				completed++
			case model.WorkflowStepDebug, model.WorkflowStepStarted:
				hasRunning = true
			}
		}
	}

	progress.CompletedSteps = completed
	progress.TotalSteps = total

	if total == 0 {
		switch status {
		case "success", "completed":
			progress.Status = "completed"
			progress.Percent = 100
		case "failed", "cancelled", "canceled", "error":
			progress.Status = "failed"
			progress.Percent = 100
		case "running", "in_progress", "in-progress", "started":
			progress.Status = "running"
			progress.Percent = 0
		default:
			progress.Status = "pending"
			progress.Percent = 0
		}
		return progress
	}

	if hasFailed {
		progress.Status = "failed"
	} else if completed == total {
		progress.Status = "completed"
	} else if hasRunning {
		progress.Status = "running"
	} else {
		progress.Status = "pending"
	}

	percent := int(math.Round((float64(completed) / float64(total)) * 100))
	if progress.Status == "completed" {
		percent = 100
	}
	progress.Percent = clampInt(percent, 0, 100)
	return progress
}

func safeStatus(resourceStatus *string, workflowInfo *dataaccess.WorkflowInfo) string {
	if resourceStatus != nil && *resourceStatus != "" {
		return *resourceStatus
	}
	if workflowInfo != nil {
		return workflowInfo.WorkflowStatus
	}
	return ""
}
