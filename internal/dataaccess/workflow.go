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

// DebugEvent represents a workflow debug event with known field names
type DebugEvent struct {
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

// DebugEventsByWorkflowSteps represents workflow debug events organized by workflow step
type DebugEventsByWorkflowSteps struct {
	Bootstrap  []DebugEvent `json:"bootstrap"`
	Storage    []DebugEvent `json:"storage"`
	Network    []DebugEvent `json:"network"`
	Compute    []DebugEvent `json:"compute"`
	Deployment []DebugEvent `json:"deployment"`
	Monitoring []DebugEvent `json:"monitoring"`
	Unknown    []DebugEvent `json:"unknown"`
}

// GetDebugEventsForAllResources gets workflow events for all resources in an instance, organized by resource and workflow step.
// If fetchResourceStatus is true, also fetches and sets workflow status from DescribeWorkflow.
func GetDebugEventsForAllResources(ctx context.Context, token string, serviceID, environmentID, instanceID string, fetchResourceStatus bool, expectedAction ...string) ([]ResourceWorkflowDebugEvents, *WorkflowInfo, error) {
	// First, list all workflows for the instance
	workflows, err := ListWorkflows(ctx, token, serviceID, environmentID, &ListWorkflowsOptions{
		InstanceID: instanceID,
	})
	if err != nil {
		return nil, nil, err
	}

	if workflows == nil || workflows.Workflows == nil {
		return []ResourceWorkflowDebugEvents{}, &WorkflowInfo{}, nil
	}

	workflowInfo := &WorkflowInfo{}
	var resourcesData []ResourceWorkflowDebugEvents

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
				eventsByWorkflowStep := &DebugEventsByWorkflowSteps{
					Bootstrap:  []DebugEvent{},
					Storage:    []DebugEvent{},
					Network:    []DebugEvent{},
					Compute:    []DebugEvent{},
					Deployment: []DebugEvent{},
					Monitoring: []DebugEvent{},
					Unknown:    []DebugEvent{},
				}

				// Categorize events by workflow step for this resource
				if resource.WorkflowSteps != nil {
					for _, step := range resource.WorkflowSteps {
						workflowStep := workflowStepName(step.StepName)

						if step.Events != nil {
							for _, event := range step.Events {
								// Create a DebugEvent with proper data
								workflowEvent := DebugEvent{
									EventTime: event.EventTime,
									EventType: event.EventType,
									Message:   event.Message,
								}

								// Add to appropriate workflowStep
								switch workflowStep {
								case "bootstrap":
									eventsByWorkflowStep.Bootstrap = append(eventsByWorkflowStep.Bootstrap, workflowEvent)
								case "storage":
									eventsByWorkflowStep.Storage = append(eventsByWorkflowStep.Storage, workflowEvent)
								case "network":
									eventsByWorkflowStep.Network = append(eventsByWorkflowStep.Network, workflowEvent)
								case "compute":
									eventsByWorkflowStep.Compute = append(eventsByWorkflowStep.Compute, workflowEvent)
								case "deployment":
									eventsByWorkflowStep.Deployment = append(eventsByWorkflowStep.Deployment, workflowEvent)
								case "monitoring":
									eventsByWorkflowStep.Monitoring = append(eventsByWorkflowStep.Monitoring, workflowEvent)
								default:
									eventsByWorkflowStep.Unknown = append(eventsByWorkflowStep.Unknown, workflowEvent)
								}
							}
						}
					}
				}

				// Add this resource's data to the result
				resourcesData = append(resourcesData, ResourceWorkflowDebugEvents{
					ResourceID:           resource.ResourceId,
					ResourceKey:          resource.ResourceKey,
					ResourceName:         resource.ResourceName,
					EventsByWorkflowStep: eventsByWorkflowStep,
				})
			}
		}
	}

	// Optionally fetch and set resource status from DescribeWorkflow
	if fetchResourceStatus && workflowInfo.WorkflowID != "" {
		describeResult, err := DescribeWorkflow(ctx, token, serviceID, environmentID, workflowInfo.WorkflowID)
		if err != nil {
			// If DescribeWorkflow fails, continue with existing data
			// Don't fail the entire operation - log the error but continue
			return resourcesData, workflowInfo, err
		}
		if describeResult != nil {
			generalStatus := describeResult.Workflow.Status
			for i := range resourcesData {
				// Only set status if not already set from events analysis
				if resourcesData[i].WorkflowStatus == nil {
					resourcesData[i].WorkflowStatus = &generalStatus
				}
			}
		}
	}
	return resourcesData, workflowInfo, nil
}

// ResourceWorkflowDebugEvents represents workflow debug events for a resource organized by workflow step
type ResourceWorkflowDebugEvents struct {
	ResourceID           string                      `json:"resourceId"`
	ResourceKey          string                      `json:"resourceKey"`
	ResourceName         string                      `json:"resourceName"`
	EventsByWorkflowStep *DebugEventsByWorkflowSteps `json:"eventsByWorkflowStep"`
	WorkflowStatus       *string                     `json:"workflowStatus,omitempty"` // From DescribeWorkflow API
}

// workflowStepName determines the workflow step for a given step name
func workflowStepName(stepName string) string {
	stepLower := strings.ToLower(stepName)

	// More comprehensive categorization based on step name patterns
	if strings.Contains(stepLower, "bootstrap") {
		return "bootstrap"
	}
	if strings.Contains(stepLower, "storage") {
		return "storage"
	}
	if strings.Contains(stepLower, "network") {
		return "network"
	}
	if strings.Contains(stepLower, "compute") {
		return "compute"
	}
	if strings.Contains(stepLower, "deployment") {
		return "deployment"
	}
	if strings.Contains(stepLower, "monitoring") {
		return "monitoring"
	}

	return "unknown"
}

// ListDeploymentCellWorkflowsOptions contains options for listing deployment cell workflows
type ListDeploymentCellWorkflowsOptions struct {
	StartDate     *time.Time
	EndDate       *time.Time
	PageSize      *int64
	NextPageToken string
}

// ListDeploymentCellWorkflows lists workflows for a deployment cell
func ListDeploymentCellWorkflows(ctx context.Context, token string, hostClusterID string, opts *ListDeploymentCellWorkflowsOptions) (res *openapiclientfleet.ListDeploymentCellWorkflowsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Create the request body
	reqBody := openapiclientfleet.NewListDeploymentCellWorkflowsRequest2()

	if opts != nil {
		if opts.StartDate != nil {
			reqBody.SetStartDate(*opts.StartDate)
		}
		if opts.EndDate != nil {
			reqBody.SetEndDate(*opts.EndDate)
		}
		if opts.PageSize != nil {
			reqBody.SetPageSize(*opts.PageSize)
		}
		if opts.NextPageToken != "" {
			reqBody.SetNextPageToken(opts.NextPageToken)
		}
	}

	req := apiClient.HostclusterApiAPI.HostclusterApiListDeploymentCellWorkflows(
		ctxWithToken,
		hostClusterID,
	).ListDeploymentCellWorkflowsRequest2(*reqBody)

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

// DescribeDeploymentCellWorkflow describes a specific deployment cell workflow
func DescribeDeploymentCellWorkflow(ctx context.Context, token string, hostClusterID, workflowID string) (res *openapiclientfleet.DescribeDeploymentCellWorkflowResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiDescribeDeploymentCellWorkflow(
		ctxWithToken,
		hostClusterID,
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

// GetDeploymentCellWorkflowEvents gets events for a deployment cell workflow
func GetDeploymentCellWorkflowEvents(ctx context.Context, token string, hostClusterID, workflowID string) (res *openapiclientfleet.GetDeploymentCellWorkflowEventsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiGetDeploymentCellWorkflowEvents(
		ctxWithToken,
		hostClusterID,
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
