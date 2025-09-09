package cost

import (
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var cloudProviderCmd = &cobra.Command{
	Use:   "cloud-provider",
	Short: "Get cost breakdown by cloud provider",
	Long:  "Get the total cost of operating your fleet across different cloud providers.",
	RunE:  getCloudProviderCost,
}

func init() {
	cloudProviderCmd.Flags().String("start-date", "", "Start date for cost analysis (RFC3339 format) (required)")
	cloudProviderCmd.Flags().String("end-date", "", "End date for cost analysis (RFC3339 format) (required)")
	cloudProviderCmd.Flags().StringP("environment-type", "e", "", "Environment type (required)")
	cloudProviderCmd.Flags().StringP("frequency", "f", "daily", "Frequency of cost data (daily, weekly, monthly)")
	cloudProviderCmd.Flags().String("include-providers", "", "Cloud provider IDs to include (comma-separated)")
	cloudProviderCmd.Flags().String("exclude-providers", "", "Cloud provider IDs to exclude (comma-separated)")

	_ = cloudProviderCmd.MarkFlagRequired("start-date")
	_ = cloudProviderCmd.MarkFlagRequired("end-date")
	_ = cloudProviderCmd.MarkFlagRequired("environment-type")
}

func getCloudProviderCost(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	frequency, _ := cmd.Flags().GetString("frequency")
	includeProviders, _ := cmd.Flags().GetString("include-providers")
	excludeProviders, _ := cmd.Flags().GetString("exclude-providers")

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

	result, err := dataaccess.DescribeCloudProviderCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get cloud provider cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}