package workflow

import (
	"fmt"
	"sort"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows for a service environment",
	Long:  "List workflows for a specific service and environment. By default, shows the 10 most recent workflows.",
	RunE:  listWorkflows,
}

type WorkflowListItem struct {
	ID              string `json:"id" table:"ID"`
	Status          string `json:"status" table:"Status"`
	WorkflowType    string `json:"workflowType" table:"Workflow Type"`
	CloudProvider   string `json:"cloudProvider" table:"Cloud Provider"`
	StartTime       string `json:"startTime" table:"Start Time"`
	EndTime         string `json:"endTime" table:"End Time"`
	OrgName         string `json:"orgName" table:"Organization"`
	ServicePlanName string `json:"servicePlanName" table:"Service Plan"`
}

func init() {
	listCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	listCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")
	listCmd.Flags().StringP("instance-id", "i", "", "Filter by instance ID (optional)")
	listCmd.Flags().Int("limit", 10, "Maximum number of workflows to return (default: 10, use 0 for no limit)")
	listCmd.Flags().String("start-date", "", "Filter workflows created after this date (RFC3339 format)")
	listCmd.Flags().String("end-date", "", "Filter workflows created before this date (RFC3339 format)")
	listCmd.Flags().String("next-page-token", "", "Token for next page of results")

	_ = listCmd.MarkFlagRequired("service-id")
	_ = listCmd.MarkFlagRequired("environment-id")
}

func listWorkflows(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	instanceID, _ := cmd.Flags().GetString("instance-id")
	limit, _ := cmd.Flags().GetInt("limit")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Always create options to handle limit
	opts := &dataaccess.ListWorkflowsOptions{
		InstanceID:    instanceID,
		NextPageToken: nextPageToken,
	}

	// Use limit as page size to get workflows
	if limit > 0 {
		limitInt64 := int64(limit)
		opts.PageSize = &limitInt64
	}

	if startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return fmt.Errorf("invalid start-date format, expected RFC3339: %w", err)
		}
		opts.StartDate = &startDate
	}

	if endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return fmt.Errorf("invalid end-date format, expected RFC3339: %w", err)
		}
		opts.EndDate = &endDate
	}

	result, err := dataaccess.ListWorkflows(ctx, token, serviceID, environmentID, opts)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	if result == nil || result.Workflows == nil {
		fmt.Println("No workflows found")
		return nil
	}

	var workflows []WorkflowListItem
	for _, workflow := range result.Workflows {
		item := WorkflowListItem{
			ID:            workflow.Id,
			Status:        workflow.Status,
			WorkflowType:  workflow.WorkflowType,
			CloudProvider: workflow.CloudProvider,
			StartTime:     workflow.StartTime,
			OrgName:       workflow.OrgName,
		}
		if workflow.EndTime != nil {
			item.EndTime = *workflow.EndTime
		}
		if workflow.ServicePlanName != nil {
			item.ServicePlanName = *workflow.ServicePlanName
		}
		workflows = append(workflows, item)
	}

	// Sort workflows by start time (most recent first) to ensure we show the most recent
	sort.Slice(workflows, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, workflows[i].StartTime)
		timeJ, errJ := time.Parse(time.RFC3339, workflows[j].StartTime)

		// If we can't parse times, keep original order
		if errI != nil || errJ != nil {
			return false
		}

		// Sort descending (most recent first)
		return timeI.After(timeJ)
	})

	// Apply client-side limit if API returned more than requested
	if limit > 0 && len(workflows) > limit {
		workflows = workflows[:limit]
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, workflows)
}
