package cost

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byProviderCmd = &cobra.Command{
	Use:   "by-provider",
	Short: "Get cost breakdown by cloud provider",
	Long:  "Analyze costs aggregated by cloud provider across your fleet.",
}

var byProviderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all cloud providers",
	Long:  "Get cost breakdown for all cloud providers in your account.",
	RunE:  runByProviderList,
}

var byProviderShowCmd = &cobra.Command{
	Use:   "show <provider-id>",
	Short: "Show costs for a specific cloud provider",
	Long:  "Get detailed cost breakdown for a specific cloud provider.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByProviderShow,
}

var byProviderCompareCmd = &cobra.Command{
	Use:   "compare <provider-id-1> <provider-id-2> [provider-id-3...]",
	Short: "Compare costs across multiple cloud providers",
	Long:  "Compare cost breakdown across two or more cloud providers.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runByProviderCompare,
}

func init() {
	// Add subcommands
	byProviderCmd.AddCommand(byProviderListCmd)
	byProviderCmd.AddCommand(byProviderShowCmd)
	byProviderCmd.AddCommand(byProviderCompareCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{byProviderListCmd, byProviderShowCmd, byProviderCompareCmd} {
		addCommonTimeFlags(cmd)
	}
}

func runByProviderList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)

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

func runByProviderShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	providerID := args[0]

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeCloudProviderIDs = &providerID

	result, err := dataaccess.DescribeCloudProviderCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get cloud provider cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for provider: %s\n", providerID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByProviderCompare(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	providerIDs := args

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Join provider IDs with comma
	providerIDsStr := ""
	for i, id := range providerIDs {
		if i > 0 {
			providerIDsStr += ","
		}
		providerIDsStr += id
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeCloudProviderIDs = &providerIDsStr

	result, err := dataaccess.DescribeCloudProviderCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get cloud provider cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found for specified providers")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
