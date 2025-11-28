package cost

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byCellCmd = &cobra.Command{
	Use:   "by-cell",
	Short: "Get cost breakdown by deployment cell",
	Long:  "Analyze costs aggregated by deployment cell across your fleet.",
}

var byCellListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all deployment cells",
	Long:  "Get cost breakdown for all deployment cells in your account.",
	RunE:  runByCellList,
}

var byCellShowCmd = &cobra.Command{
	Use:   "show <cell-id>",
	Short: "Show costs for a specific deployment cell",
	Long:  "Get detailed cost breakdown for a specific deployment cell.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByCellShow,
}

var byCellCompareCmd = &cobra.Command{
	Use:   "compare <cell-id-1> <cell-id-2> [cell-id-3...]",
	Short: "Compare costs across multiple deployment cells",
	Long:  "Compare cost breakdown across two or more deployment cells.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runByCellCompare,
}

var byCellTopInstancesCmd = &cobra.Command{
	Use:   "top-instances <N>",
	Short: "Get top N most expensive instances across all cells",
	Long:  "List the top N most expensive instances across all deployment cells.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByCellTopInstances,
}

var byCellInProviderCmd = &cobra.Command{
	Use:   "in-provider <provider-id>",
	Short: "List cells in a specific cloud provider",
	Long:  "Get cost breakdown for all deployment cells in a specific cloud provider.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByCellInProvider,
}

var byCellInRegionCmd = &cobra.Command{
	Use:   "in-region <region-id>",
	Short: "List cells in a specific region",
	Long:  "Get cost breakdown for all deployment cells in a specific region.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByCellInRegion,
}

func init() {
	// Add subcommands
	byCellCmd.AddCommand(byCellListCmd)
	byCellCmd.AddCommand(byCellShowCmd)
	byCellCmd.AddCommand(byCellCompareCmd)
	byCellCmd.AddCommand(byCellTopInstancesCmd)
	byCellCmd.AddCommand(byCellInProviderCmd)
	byCellCmd.AddCommand(byCellInRegionCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{
		byCellListCmd, byCellShowCmd, byCellCompareCmd,
		byCellTopInstancesCmd, byCellInProviderCmd, byCellInRegionCmd,
	} {
		addCommonTimeFlags(cmd)
	}
}

func runByCellList(cmd *cobra.Command, args []string) error {
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

func runByCellShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cellID := args[0]

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeDeploymentCellIDs = &cellID

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for deployment cell: %s\n", cellID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByCellCompare(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cellIDs := args

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Join cell IDs with comma
	cellIDsStr := ""
	for i, id := range cellIDs {
		if i > 0 {
			cellIDsStr += ","
		}
		cellIDsStr += id
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeDeploymentCellIDs = &cellIDsStr

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found for specified deployment cells")
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByCellTopInstances(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var topN int64
	_, err := fmt.Sscanf(args[0], "%d", &topN)
	if err != nil {
		return fmt.Errorf("invalid number: %s", args[0])
	}

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.TopNInstances = &topN

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

func runByCellInProvider(cmd *cobra.Command, args []string) error {
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

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for deployment cells in provider: %s\n", providerID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}

func runByCellInRegion(cmd *cobra.Command, args []string) error {
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

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for deployment cells in region: %s\n", regionID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{result})
}
