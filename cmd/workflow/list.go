package workflow

import (
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows for a service environment",
	Long:  "List all workflows for a specific service and environment.",
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
	listCmd.Flags().String("start-date", "", "Filter workflows created after this date (RFC3339 format)")
	listCmd.Flags().String("end-date", "", "Filter workflows created before this date (RFC3339 format)")
	listCmd.Flags().Int64("page-size", 0, "Number of results per page")
	listCmd.Flags().String("next-page-token", "", "Token for next page of results")

	_ = listCmd.MarkFlagRequired("service-id")
	_ = listCmd.MarkFlagRequired("environment-id")
}

func listWorkflows(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	instanceID, _ := cmd.Flags().GetString("instance-id")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	var opts *dataaccess.ListWorkflowsOptions
	if instanceID != "" || startDateStr != "" || endDateStr != "" || pageSize != 0 || nextPageToken != "" {
		opts = &dataaccess.ListWorkflowsOptions{
			InstanceID:    instanceID,
			NextPageToken: nextPageToken,
		}

		if pageSize != 0 {
			opts.PageSize = &pageSize
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

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, workflows)
}
