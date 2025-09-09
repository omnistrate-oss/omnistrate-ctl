package cost

import (
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Get cost breakdown by user",
	Long:  "Get the total cost of operating your fleet for different users.",
	RunE:  getUserCost,
}

func init() {
	userCmd.Flags().String("start-date", "", "Start date for cost analysis (RFC3339 format) (required)")
	userCmd.Flags().String("end-date", "", "End date for cost analysis (RFC3339 format) (required)")
	userCmd.Flags().StringP("environment-type", "e", "", "Environment type (required)")
	userCmd.Flags().String("include-users", "", "User IDs to include (comma-separated)")
	userCmd.Flags().String("exclude-users", "", "User IDs to exclude (comma-separated)")
	userCmd.Flags().Int64("top-n-users", 0, "Limit results to top N users by cost")
	userCmd.Flags().Int64("top-n-instances", 0, "Limit results to top N instances by cost")

	_ = userCmd.MarkFlagRequired("start-date")
	_ = userCmd.MarkFlagRequired("end-date")
	_ = userCmd.MarkFlagRequired("environment-type")
}

func getUserCost(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	includeUsers, _ := cmd.Flags().GetString("include-users")
	excludeUsers, _ := cmd.Flags().GetString("exclude-users")
	topNUsers, _ := cmd.Flags().GetInt64("top-n-users")
	topNInstances, _ := cmd.Flags().GetInt64("top-n-instances")

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start-date format, expected RFC3339: %w", err)
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		return fmt.Errorf("invalid end-date format, expected RFC3339: %w", err)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := dataaccess.CostOptions{
		StartDate:       startDate,
		EndDate:         endDate,
		EnvironmentType: environmentType,
	}

	if includeUsers != "" {
		opts.IncludeUserIDs = &includeUsers
	}
	if excludeUsers != "" {
		opts.ExcludeUserIDs = &excludeUsers
	}
	if topNUsers > 0 {
		opts.TopNUsers = &topNUsers
	}
	if topNInstances > 0 {
		opts.TopNInstances = &topNInstances
	}

	result, err := dataaccess.DescribeUserCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get user cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}