package dataaccess

import (
	"context"
	"net/http"
	"strings"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type ListWorkflowsOptions struct {
	InstanceID    string
	StartDate     *time.Time
	EndDate       *time.Time
	PageSize      *int64
	NextPageToken string
}

func ListWorkflows(ctx context.Context, token string, serviceID, environmentID string, opts *ListWorkflowsOptions) (res *openapiclientfleet.ListServiceWorkflowsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiListServiceWorkflows(
		ctxWithToken,
		serviceID,
		environmentID,
	)

	if opts != nil {
		if opts.InstanceID != "" {
			req = req.InstanceId(opts.InstanceID)
		}
		if opts.StartDate != nil {
			req = req.StartDate(*opts.StartDate)
		}
		if opts.EndDate != nil {
			req = req.EndDate(*opts.EndDate)
		}
		if opts.PageSize != nil {
			req = req.PageSize(*opts.PageSize)
		}
		if opts.NextPageToken != "" {
			req = req.NextPageToken(opts.NextPageToken)
		}
	}

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

func DescribeWorkflow(ctx context.Context, token string, serviceID, environmentID, workflowID string) (res *openapiclientfleet.DescribeServiceWorkflowResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiDescribeServiceWorkflow(
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

func DescribeWorkflowSummary(ctx context.Context, token string, serviceID, environmentID string) (res *openapiclientfleet.DescribeServiceWorkflowSummaryResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiDescribeServiceWorkflowSummary(
		ctxWithToken,
		serviceID,
		environmentID,
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

func TerminateWorkflow(ctx context.Context, token string, serviceID, environmentID, workflowID string) (res *openapiclientfleet.ServiceWorkflow, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetWorkflowsApiAPI.FleetWorkflowsApiTerminateServiceWorkflow(
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

// GetDebugEventsForAllResources gets workflow events for all resources in an instance, organized by resource and category
func GetDebugEventsForAllResources(ctx context.Context, token string, serviceID, environmentID, instanceID string, expectedAction ...string) ([]ResourceWorkflowData, *WorkflowInfo, error) {
	// First, list all workflows for the instance
	workflows, err := ListWorkflows(ctx, token, serviceID, environmentID, &ListWorkflowsOptions{
		InstanceID: instanceID,
	})
	if err != nil {
		return nil, nil, err
	}

	if workflows == nil || workflows.Workflows == nil {
		return []ResourceWorkflowData{}, &WorkflowInfo{}, nil
	}

	workflowInfo := &WorkflowInfo{}
	var resourcesData []ResourceWorkflowData

	// Find the latest workflow that matches the expected action (if specified)
	var latestWorkflowID string
	for _, workflow := range workflows.Workflows {
		// Skip workflows that are pending or have specific prefixes license and backup
		if workflow.Id == "" || workflow.Status == "pending" || strings.HasPrefix(workflow.Id, "submit-rotate-license") || strings.HasPrefix(workflow.Id, "submit-backup") {
			continue
		}

		// If expected action is specified, validate workflow matches the action
		if len(expectedAction) > 0 && expectedAction[0] != "" {
			actionType := expectedAction[0]
			workflowMatches := false

			// Check if workflow ID contains the expected action type
			workflowLower := strings.ToLower(workflow.Id)
			actionLower := strings.ToLower(actionType)

			switch actionLower {
			case "create":
				workflowMatches = strings.HasPrefix(workflowLower, "submit-create")
			case "modify":
				workflowMatches = strings.HasPrefix(workflowLower, "submit-update") || strings.HasPrefix(workflowLower, "submit-modify")
			case "upgrade":
				workflowMatches = strings.HasPrefix(workflowLower, "submit-update") || strings.HasPrefix(workflowLower, "submit-modify") || strings.Contains(workflowLower, "upgrade")
			case "delete":
				workflowMatches = strings.HasPrefix(workflowLower, "submit-delete")
			default:
				// If action type is unknown, fall back to original behavior
				workflowMatches = true
			}

			// Skip workflows that don't match the expected action
			if !workflowMatches {
				continue
			}
		}

		latestWorkflowID = workflow.Id

		// Populate workflow metadata
		workflowInfo.WorkflowID = workflow.Id
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
			return resourcesData, workflowInfo, err
		}

		if workflowEvents != nil && workflowEvents.Resources != nil {
			// Work directly with the struct - no need for marshaling/unmarshaling
			for _, resource := range workflowEvents.Resources {
				eventsByCategory := &WorkflowEventsByCategory{
					Bootstrap:  []CustomWorkflowEvent{},
					Storage:    []CustomWorkflowEvent{},
					Network:    []CustomWorkflowEvent{},
					Compute:    []CustomWorkflowEvent{},
					Deployment: []CustomWorkflowEvent{},
					Monitoring: []CustomWorkflowEvent{},
					Other:      []CustomWorkflowEvent{},
				}

				// Categorize events by workflow step for this resource
				if resource.WorkflowSteps != nil {
					for _, step := range resource.WorkflowSteps {
						stepCategory := categorizeStepName(step.StepName)

						if step.Events != nil {
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

				// Add this resource's data to the result
				resourcesData = append(resourcesData, ResourceWorkflowData{
					ResourceID:       resource.ResourceId,
					ResourceKey:      resource.ResourceKey,
					ResourceName:     resource.ResourceName,
					EventsByCategory: eventsByCategory,
				})
			}
		}
	}

	return resourcesData, workflowInfo, nil
}

// ResourceWorkflowData represents workflow events for a resource organized by category
type ResourceWorkflowData struct {
	ResourceID       string                    `json:"resourceId"`
	ResourceKey      string                    `json:"resourceKey"`
	ResourceName     string                    `json:"resourceName"`
	EventsByCategory *WorkflowEventsByCategory `json:"eventsByCategory"`
	ResourceStatus   string                    `json:"resourceStatus,omitempty"` // From DescribeWorkflow API
}

// GetResourceStatusFromDescribeWorkflow gets resource status information using DescribeWorkflow API
func GetResourceStatusFromDescribeWorkflow(ctx context.Context, token string, serviceID, environmentID, workflowID string) (map[string]string, error) {
	// Call DescribeWorkflow API
	describeResult, err := DescribeWorkflow(ctx, token, serviceID, environmentID, workflowID)
	if err != nil {
		return nil, err
	}

	resourceStatusMap := make(map[string]string)
	if describeResult != nil {
		// Get workflow status as fallback
		workflowStatus := describeResult.Workflow.Status
		
		// Check if the workflow has resource information
		// Since we can't access Resources directly from ServiceWorkflow,
		// we use the workflow status as a general indicator
		// This will be enhanced when we get more detailed resource status info
		resourceStatusMap["__workflow_general_status__"] = workflowStatus
	}
	
	return resourceStatusMap, nil
}

// GetDebugEventsForAllResourcesWithStatus gets workflow events and resource status for all resources in an instance
func GetDebugEventsForAllResourcesWithStatus(ctx context.Context, token string, serviceID, environmentID, instanceID string, expectedAction ...string) ([]ResourceWorkflowData, *WorkflowInfo, error) {
	// First get the basic workflow events data
	resourcesData, workflowInfo, err := GetDebugEventsForAllResources(ctx, token, serviceID, environmentID, instanceID, expectedAction...)
	if err != nil {
		return resourcesData, workflowInfo, err
	}

	// If we have a workflow ID, get additional resource status from DescribeWorkflow
	if workflowInfo != nil && workflowInfo.WorkflowID != "" {
		resourceStatusMap, err := GetResourceStatusFromDescribeWorkflow(ctx, token, serviceID, environmentID, workflowInfo.WorkflowID)
		if err != nil {
			// If DescribeWorkflow fails, continue with existing data
			// Don't fail the entire operation
			return resourcesData, workflowInfo, nil
		}
		
		// Enhance resource data with status from DescribeWorkflow
		for i := range resourcesData {
			if status, exists := resourceStatusMap[resourcesData[i].ResourceID]; exists {
				resourcesData[i].ResourceStatus = status
			}
		}
	}

	return resourcesData, workflowInfo, nil
}

// categorizeStepName determines the category for a workflow step name
func categorizeStepName(stepName string) string {
	stepLower := strings.ToLower(stepName)

	// More comprehensive categorization based on step name patterns
	if strings.Contains(stepLower, "bootstrap") || strings.Contains(stepLower, "init") {
		return "bootstrap"
	}
	if strings.Contains(stepLower, "storage") || strings.Contains(stepLower, "disk") || strings.Contains(stepLower, "volume") {
		return "storage"
	}
	if strings.Contains(stepLower, "network") || strings.Contains(stepLower, "vpc") || strings.Contains(stepLower, "subnet") {
		return "network"
	}
	if strings.Contains(stepLower, "compute") || strings.Contains(stepLower, "instance") || strings.Contains(stepLower, "vm") || strings.Contains(stepLower, "server") {
		return "compute"
	}
	if strings.Contains(stepLower, "deployment") || strings.Contains(stepLower, "deploy") || strings.Contains(stepLower, "install") {
		return "deployment"
	}
	if strings.Contains(stepLower, "monitoring") || strings.Contains(stepLower, "observability") || strings.Contains(stepLower, "metrics") {
		return "monitoring"
	}

	return "other"
}
