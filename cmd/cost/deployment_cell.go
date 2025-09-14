package cost

import (
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var deploymentCellCmd = &cobra.Command{
	Use:   "deployment-cell",
	Short: "Get cost breakdown by deployment cell",
	Long:  "Get the total cost of operating your fleet across different deployment cells.",
	RunE:  getDeploymentCellCost,
}

func init() {
	deploymentCellCmd.Flags().String("start-date", "", "Start date for cost analysis (RFC3339 format) (required)")
	deploymentCellCmd.Flags().String("end-date", "", "End date for cost analysis (RFC3339 format) (required)")
	deploymentCellCmd.Flags().StringP("environment-type", "e", "", "Environment type (required)")
	deploymentCellCmd.Flags().StringP("frequency", "f", "daily", "Frequency of cost data (daily, weekly, monthly)")
	deploymentCellCmd.Flags().String("include-providers", "", "Cloud provider IDs to include (comma-separated)")
	deploymentCellCmd.Flags().String("exclude-providers", "", "Cloud provider IDs to exclude (comma-separated)")
	deploymentCellCmd.Flags().String("include-cells", "", "Deployment cell IDs to include (comma-separated)")
	deploymentCellCmd.Flags().String("exclude-cells", "", "Deployment cell IDs to exclude (comma-separated)")
	deploymentCellCmd.Flags().String("include-instances", "", "Instance IDs to include (comma-separated)")
	deploymentCellCmd.Flags().String("exclude-instances", "", "Instance IDs to exclude (comma-separated)")
	deploymentCellCmd.Flags().Int64("top-n-instances", 0, "Limit results to top N instances by cost")

	_ = deploymentCellCmd.MarkFlagRequired("start-date")
	_ = deploymentCellCmd.MarkFlagRequired("end-date")
	_ = deploymentCellCmd.MarkFlagRequired("environment-type")
}

func getDeploymentCellCost(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	frequency, _ := cmd.Flags().GetString("frequency")
	includeProviders, _ := cmd.Flags().GetString("include-providers")
	excludeProviders, _ := cmd.Flags().GetString("exclude-providers")
	includeCells, _ := cmd.Flags().GetString("include-cells")
	excludeCells, _ := cmd.Flags().GetString("exclude-cells")
	includeInstances, _ := cmd.Flags().GetString("include-instances")
	excludeInstances, _ := cmd.Flags().GetString("exclude-instances")
	topNInstances, _ := cmd.Flags().GetInt64("top-n-instances")

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start-date format, expected RFC3339 (e.g. '2006-01-02T15:04:05Z07:00'): %w", err)
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		return fmt.Errorf("invalid end-date format, expected RFC3339 (e.g. '2006-01-02T15:04:05Z07:00'): %w", err)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := dataaccess.CostOptions{
		StartDate:       startDate,
		EndDate:         endDate,
		EnvironmentType: environmentType,
		Frequency:       frequency,
	}

	if includeProviders != "" {
		opts.IncludeCloudProviderIDs = &includeProviders
	}
	if excludeProviders != "" {
		opts.ExcludeCloudProviderIDs = &excludeProviders
	}
	if includeCells != "" {
		opts.IncludeDeploymentCellIDs = &includeCells
	}
	if excludeCells != "" {
		opts.ExcludeDeploymentCellIDs = &excludeCells
	}
	if includeInstances != "" {
		opts.IncludeInstanceIDs = &includeInstances
	}
	if excludeInstances != "" {
		opts.ExcludeInstanceIDs = &excludeInstances
	}
	if topNInstances > 0 {
		opts.TopNInstances = &topNInstances
	}

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
