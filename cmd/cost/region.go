package cost

import (
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var regionCmd = &cobra.Command{
	Use:   "region",
	Short: "Get cost breakdown by region",
	Long:  "Get the total cost of operating your fleet across different regions.",
	RunE:  getRegionCost,
}

func init() {
	regionCmd.Flags().String("start-date", "", "Start date for cost analysis (RFC3339 format) (required)")
	regionCmd.Flags().String("end-date", "", "End date for cost analysis (RFC3339 format) (required)")
	regionCmd.Flags().StringP("environment-type", "e", "", "Environment type (required)")
	regionCmd.Flags().StringP("frequency", "f", "daily", "Frequency of cost data (daily, weekly, monthly)")
	regionCmd.Flags().String("include-providers", "", "Cloud provider IDs to include (comma-separated)")
	regionCmd.Flags().String("exclude-providers", "", "Cloud provider IDs to exclude (comma-separated)")
	regionCmd.Flags().String("include-regions", "", "Region IDs to include (comma-separated)")
	regionCmd.Flags().String("exclude-regions", "", "Region IDs to exclude (comma-separated)")

	_ = regionCmd.MarkFlagRequired("start-date")
	_ = regionCmd.MarkFlagRequired("end-date")
	_ = regionCmd.MarkFlagRequired("environment-type")
}

func getRegionCost(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	frequency, _ := cmd.Flags().GetString("frequency")
	includeProviders, _ := cmd.Flags().GetString("include-providers")
	excludeProviders, _ := cmd.Flags().GetString("exclude-providers")
	includeRegions, _ := cmd.Flags().GetString("include-regions")
	excludeRegions, _ := cmd.Flags().GetString("exclude-regions")

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
	if includeRegions != "" {
		opts.IncludeRegionIDs = &includeRegions
	}
	if excludeRegions != "" {
		opts.ExcludeRegionIDs = &excludeRegions
	}

	result, err := dataaccess.DescribeRegionCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get region cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
