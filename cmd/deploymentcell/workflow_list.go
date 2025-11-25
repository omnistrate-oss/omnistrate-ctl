package deploymentcell

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var workflowListCmd = &cobra.Command{
	Use:   "list [deployment-cell-id]",
	Short: "List workflows for a deployment cell",
	Long:  "List workflows for a specific deployment cell. By default, shows the 10 most recent workflows.",
	Args:  cobra.ExactArgs(1),
	RunE:  listDeploymentCellWorkflows,
}

type DeploymentCellWorkflowListItem struct {
	ID           string `json:"id" table:"ID"`
	Status       string `json:"status" table:"Status"`
	WorkflowType string `json:"workflowType" table:"Workflow Type"`
	StartTime    string `json:"startTime" table:"Start Time"`
	EndTime      string `json:"endTime" table:"End Time"`
	OrgName      string `json:"orgName" table:"Organization"`
}

func init() {
	workflowListCmd.Flags().Int("limit", 10, "Maximum number of workflows to return (default: 10, use 0 for no limit)")
	workflowListCmd.Flags().String("start-date", "", "Filter workflows created after this date (RFC3339 format)")
	workflowListCmd.Flags().String("end-date", "", "Filter workflows created before this date (RFC3339 format)")
	workflowListCmd.Flags().String("next-page-token", "", "Token for next page of results")
}

func listDeploymentCellWorkflows(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	deploymentCellID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Create options
	opts := &dataaccess.ListDeploymentCellWorkflowsOptions{
		NextPageToken: nextPageToken,
	}

	// Use limit as page size
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

	result, err := dataaccess.ListDeploymentCellWorkflows(ctx, token, deploymentCellID, opts)
	if err != nil {
		return fmt.Errorf("failed to list deployment cell workflows: %w", err)
	}

	if result == nil || result.Workflows == nil {
		fmt.Println("No workflows found")
		return nil
	}

	var workflows []DeploymentCellWorkflowListItem
	for _, workflow := range result.Workflows {
		item := DeploymentCellWorkflowListItem{
			ID:           workflow.WorkflowID,
			Status:       workflow.Status,
			WorkflowType: workflow.WorkflowType,
			StartTime:    workflow.StartTime,
			OrgName:      workflow.OrgName,
		}
		if workflow.EndTime != nil {
			item.EndTime = *workflow.EndTime
		}
		workflows = append(workflows, item)
	}

	// Sort workflows by start time (most recent first)
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
