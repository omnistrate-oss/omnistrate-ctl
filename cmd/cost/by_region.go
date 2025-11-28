package cost

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byRegionCmd = &cobra.Command{
	Use:   "by-region",
	Short: "Get cost breakdown by region",
	Long:  "Analyze costs aggregated by region across your fleet.",
}

var byRegionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all regions",
	Long:  "Get cost breakdown for all regions in your account.",
	RunE:  runByRegionList,
}

var byRegionShowCmd = &cobra.Command{
	Use:   "show <region-id>",
	Short: "Show costs for a specific region",
	Long:  "Get detailed cost breakdown for a specific region.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByRegionShow,
}

var byRegionCompareCmd = &cobra.Command{
	Use:   "compare <region-id-1> <region-id-2> [region-id-3...]",
	Short: "Compare costs across multiple regions",
	Long:  "Compare cost breakdown across two or more regions.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runByRegionCompare,
}

var byRegionInProviderCmd = &cobra.Command{
	Use:   "in-provider <provider-id>",
	Short: "List regions in a specific cloud provider",
	Long:  "Get cost breakdown for all regions in a specific cloud provider.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByRegionInProvider,
}

func init() {
	// Add subcommands
	byRegionCmd.AddCommand(byRegionListCmd)
	byRegionCmd.AddCommand(byRegionShowCmd)
	byRegionCmd.AddCommand(byRegionCompareCmd)
	byRegionCmd.AddCommand(byRegionInProviderCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{
		byRegionListCmd, byRegionShowCmd, byRegionCompareCmd, byRegionInProviderCmd,
	} {
		addCommonTimeFlags(cmd)
	}
}

func runByRegionList(cmd *cobra.Command, args []string) error {
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

func runByRegionShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	regionID := args[0]

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeRegionIDs = &regionID

	result, err := dataaccess.DescribeRegionCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get region cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for region: %s\n", regionID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByRegionCompare(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	regionIDs := args

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Join region IDs with comma
	regionIDsStr := ""
	for i, id := range regionIDs {
		if i > 0 {
			regionIDsStr += ","
		}
		regionIDsStr += id
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeRegionIDs = &regionIDsStr

	result, err := dataaccess.DescribeRegionCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get region cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found for specified regions")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByRegionInProvider(cmd *cobra.Command, args []string) error {
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

	result, err := dataaccess.DescribeRegionCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get region cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for regions in provider: %s\n", providerID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
