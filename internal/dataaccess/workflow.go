package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)



func ListWorkflows(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.ListServiceWorkflowsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiListServiceWorkflows(
		ctxWithToken,
		serviceID,
		environmentID,
	).InstanceId(instanceID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}






func GetWorkflowEvents(ctx context.Context, token string, serviceID, environmentID, workflowID string) (res *openapiclientfleet.GetWorkflowEventsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiGetWorkflowEvents(
		ctxWithToken,
		serviceID,
		environmentID,
		workflowID,
	)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

// CustomWorkflowEvent represents a workflow event with known field names
type CustomWorkflowEvent struct {
	EventTime string `json:"eventTime"`
	EventType string `json:"eventType"`
	Message   string `json:"message"`
}

// WorkflowInfo represents workflow metadata information
type WorkflowInfo struct {
	WorkflowID     string `json:"workflowId,omitempty"`
	WorkflowStatus string `json:"workflowStatus,omitempty"`
	StartTime      string `json:"startTime,omitempty"`
	EndTime        string `json:"endTime,omitempty"`
}

// WorkflowEventsByCategory represents workflow events organized by category
type WorkflowEventsByCategory struct {
	Bootstrap  []CustomWorkflowEvent `json:"bootstrap"`
	Storage    []CustomWorkflowEvent `json:"storage"`
	Network    []CustomWorkflowEvent `json:"network"`
	Compute    []CustomWorkflowEvent `json:"compute"`
	Deployment []CustomWorkflowEvent `json:"deployment"`
	Monitoring []CustomWorkflowEvent `json:"monitoring"`
	Other      []CustomWorkflowEvent `json:"other"`
}

// GetDebugEventsForResource gets workflow events for a specific resource organized by categories
func GetDebugEventsForResource(ctx context.Context, token string, serviceID, environmentID, instanceID, resourceKey string) (*WorkflowEventsByCategory, *WorkflowInfo, error) {
	// First, list all workflows for the instance
	workflows, err := ListWorkflows(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return nil, nil, err
	}

	if workflows == nil || workflows.Workflows == nil {
		return &WorkflowEventsByCategory{}, &WorkflowInfo{}, nil
	}

	eventsByCategory := &WorkflowEventsByCategory{
		Bootstrap:  []CustomWorkflowEvent{},
		Storage:    []CustomWorkflowEvent{},
		Network:    []CustomWorkflowEvent{},
		Compute:    []CustomWorkflowEvent{},
		Deployment: []CustomWorkflowEvent{},
		Monitoring: []CustomWorkflowEvent{},
		Other:      []CustomWorkflowEvent{},
	}
	workflowInfo := &WorkflowInfo{}
	// Find the latest workflow (assuming they are ordered by creation time or use the last one)
	var latestWorkflowID string
	for _, workflow := range workflows.Workflows {
		if workflow.Id == "" {
			continue
		}
		latestWorkflowID = workflow.Id

		// Populate workflow metadata
		workflowInfo.WorkflowID = workflow.Id
		
		// Extract status, start time, and end time if available
		workflowInfo.WorkflowStatus = workflow.Status
		
		if workflow.StartTime != "" {
			workflowInfo.StartTime = workflow.StartTime
		}
		if workflow.EndTime != nil {
			workflowInfo.EndTime = *workflow.EndTime
		}
		
		// Use the last workflow found (assuming they are ordered)
		break
	}

	// Process only the latest workflow
	if latestWorkflowID != "" {
		workflowEvents, err := GetWorkflowEvents(ctx, token, serviceID, environmentID, latestWorkflowID)
		if err != nil {
			return eventsByCategory, workflowInfo, err
		}

		if workflowEvents != nil {
			// Convert the result to JSON and parse it with our known structure
			workflowEventsJSON, err := json.Marshal(workflowEvents)
			if err != nil {
				return eventsByCategory, workflowInfo, fmt.Errorf("failed to marshal workflow events: %w", err)
			}

			// Parse the JSON into our expected structure
			var parsedResponse struct {
				Resources []struct {
					ResourceID    string `json:"resourceId"`
					ResourceKey   string `json:"resourceKey"`
					ResourceName  string `json:"resourceName"`
					WorkflowSteps []struct {
						StepName string `json:"stepName"`
						Events   []struct {
							EventTime string `json:"eventTime"`
							EventType string `json:"eventType"`
							Message   string `json:"message"`
						} `json:"events"`
					} `json:"workflowSteps"`
				} `json:"resources"`
			}

			err = json.Unmarshal(workflowEventsJSON, &parsedResponse)
			if err != nil {
				return eventsByCategory, workflowInfo, fmt.Errorf("failed to unmarshal workflow events: %w", err)
			}

			// Process each resource's workflow steps
			for _, resource := range parsedResponse.Resources {
				// Filter by resourceKey if provided (optional)
				if resourceKey != "" && resource.ResourceKey != resourceKey {
					continue
				}

				// Categorize events by workflow step
				for _, step := range resource.WorkflowSteps {
					stepCategory := categorizeStepName(step.StepName)
					
					for _, event := range step.Events {
						// Create a CustomWorkflowEvent with proper data
						workflowEvent := CustomWorkflowEvent{
							EventTime: event.EventTime,
							EventType: event.EventType,
							Message:   event.Message,
						}

						// Add to appropriate category
						switch stepCategory {
						case "bootstrap":
							eventsByCategory.Bootstrap = append(eventsByCategory.Bootstrap, workflowEvent)
						case "storage":
							eventsByCategory.Storage = append(eventsByCategory.Storage, workflowEvent)
						case "network":
							eventsByCategory.Network = append(eventsByCategory.Network, workflowEvent)
						case "compute":
							eventsByCategory.Compute = append(eventsByCategory.Compute, workflowEvent)
						case "deployment":
							eventsByCategory.Deployment = append(eventsByCategory.Deployment, workflowEvent)
						case "monitoring":
							eventsByCategory.Monitoring = append(eventsByCategory.Monitoring, workflowEvent)
						default:
							eventsByCategory.Other = append(eventsByCategory.Other, workflowEvent)
						}
					}
				}
			}
		}
	}

	return eventsByCategory, workflowInfo, nil
}



// categorizeStepName determines the category for a workflow step name
func categorizeStepName(stepName string) string {
	stepLower := strings.ToLower(stepName)
	switch stepLower {
	case "bootstrap":
		return "bootstrap"
	case "storage":
		return "storage"
	case "network":
		return "network"
	case "deployment":
		return "deployment"
	case "compute":
		return "compute"
	case "monitoring":
		return "monitoring"
	default:
		return "other"
	}
}

