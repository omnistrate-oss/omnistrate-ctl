package cost

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byUserCmd = &cobra.Command{
	Use:   "by-user",
	Short: "Get cost breakdown by user",
	Long:  "Analyze costs aggregated by user across your fleet.",
}

var byUserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all users",
	Long:  "Get cost breakdown for all users in your account.",
	RunE:  runByUserList,
}

var byUserShowCmd = &cobra.Command{
	Use:   "show <user-id>",
	Short: "Show costs for a specific user",
	Long:  "Get detailed cost breakdown for a specific user.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByUserShow,
}

var byUserTopCmd = &cobra.Command{
	Use:   "top <N>",
	Short: "Get top N users by cost",
	Long:  "List the top N users with the highest costs.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByUserTop,
}

var byUserTopInstancesCmd = &cobra.Command{
	Use:   "top-instances <N>",
	Short: "Get top N instances by cost for each user",
	Long:  "List the top N most expensive instances for each user.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByUserTopInstances,
}

var byUserCompareCmd = &cobra.Command{
	Use:   "compare <user-id-1> <user-id-2> [user-id-3...]",
	Short: "Compare costs across multiple users",
	Long:  "Compare cost breakdown across two or more users.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runByUserCompare,
}

func init() {
	// Add subcommands
	byUserCmd.AddCommand(byUserListCmd)
	byUserCmd.AddCommand(byUserShowCmd)
	byUserCmd.AddCommand(byUserTopCmd)
	byUserCmd.AddCommand(byUserTopInstancesCmd)
	byUserCmd.AddCommand(byUserCompareCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{
		byUserListCmd, byUserShowCmd, byUserTopCmd,
		byUserTopInstancesCmd, byUserCompareCmd,
	} {
		addCommonTimeFlags(cmd)
	}
}

func runByUserList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDate, endDate, environmentType, _, err := parseTimeFlags(cmd)
	if err != nil {
		return err
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

func runByUserShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	userID := args[0]

	startDate, endDate, environmentType, _, err := parseTimeFlags(cmd)
	if err != nil {
		return err
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
	opts.IncludeUserIDs = &userID

	result, err := dataaccess.DescribeUserCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get user cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for user: %s\n", userID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByUserTop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var topN int64
	_, err := fmt.Sscanf(args[0], "%d", &topN)
	if err != nil {
		return fmt.Errorf("invalid number: %s", args[0])
	}

	startDate, endDate, environmentType, _, err := parseTimeFlags(cmd)
	if err != nil {
		return err
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
	opts.TopNUsers = &topN

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

func runByUserTopInstances(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var topN int64
	_, err := fmt.Sscanf(args[0], "%d", &topN)
	if err != nil {
		return fmt.Errorf("invalid number: %s", args[0])
	}

	startDate, endDate, environmentType, _, err := parseTimeFlags(cmd)
	if err != nil {
		return err
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
	opts.TopNInstances = &topN

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

func runByUserCompare(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	userIDs := args

	startDate, endDate, environmentType, _, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Join user IDs with comma
	userIDsStr := ""
	for i, id := range userIDs {
		if i > 0 {
			userIDsStr += ","
		}
		userIDsStr += id
	}

	opts := dataaccess.CostOptions{
		StartDate:       startDate,
		EndDate:         endDate,
		EnvironmentType: environmentType,
	}
	opts.IncludeUserIDs = &userIDsStr

	result, err := dataaccess.DescribeUserCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get user cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found for specified users")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
