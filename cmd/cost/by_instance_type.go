package cost

import (
	"fmt"
	"sort"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byInstanceTypeCmd = &cobra.Command{
	Use:   "by-instance-type",
	Short: "Get cost breakdown by instance type",
	Long:  "Analyze costs aggregated by instance type (e.g., m5.large, c5.xlarge) across your fleet.",
}

var byInstanceTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all instance types",
	Long:  "Get cost breakdown for all instance types in your account.",
	RunE:  runByInstanceTypeList,
}

var byInstanceTypeShowCmd = &cobra.Command{
	Use:   "show <instance-type>",
	Short: "Show costs for a specific instance type",
	Long:  "Get detailed cost breakdown for a specific instance type (e.g., m5.large).",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTypeShow,
}

var byInstanceTypeTopCmd = &cobra.Command{
	Use:   "top <N>",
	Short: "Get top N instance types by cost",
	Long:  "List the top N instance types with the highest costs.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTypeTop,
}

var byInstanceTypeInCellCmd = &cobra.Command{
	Use:   "in-cell <cell-id>",
	Short: "List instance type costs in a specific deployment cell",
	Long:  "Get cost breakdown by instance type for a specific deployment cell.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTypeInCell,
}

var byInstanceTypeInProviderCmd = &cobra.Command{
	Use:   "in-provider <provider-id>",
	Short: "List instance type costs in a specific cloud provider",
	Long:  "Get cost breakdown by instance type for a specific cloud provider.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTypeInProvider,
}

var byInstanceTypeInRegionCmd = &cobra.Command{
	Use:   "in-region <region-id>",
	Short: "List instance type costs in a specific region",
	Long:  "Get cost breakdown by instance type for a specific region.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTypeInRegion,
}

func init() {
	// Add subcommands
	byInstanceTypeCmd.AddCommand(byInstanceTypeListCmd)
	byInstanceTypeCmd.AddCommand(byInstanceTypeShowCmd)
	byInstanceTypeCmd.AddCommand(byInstanceTypeTopCmd)
	byInstanceTypeCmd.AddCommand(byInstanceTypeInCellCmd)
	byInstanceTypeCmd.AddCommand(byInstanceTypeInProviderCmd)
	byInstanceTypeCmd.AddCommand(byInstanceTypeInRegionCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{
		byInstanceTypeListCmd, byInstanceTypeShowCmd, byInstanceTypeTopCmd,
		byInstanceTypeInCellCmd, byInstanceTypeInProviderCmd, byInstanceTypeInRegionCmd,
	} {
		addCommonTimeFlags(cmd)
	}
}

func runByInstanceTypeList(cmd *cobra.Command, args []string) error {
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
	// Fetch all instances to aggregate by type
	topN := int64(10000) // Large number to get all instances
	opts.TopNInstances = &topN

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	if len(aggregates) == 0 {
		fmt.Println("No instance type cost data found")
		return nil
	}

	// Convert to slice for output
	aggregateSlice := make([]interface{}, 0, len(aggregates))
	for _, agg := range aggregates {
		aggregateSlice = append(aggregateSlice, agg)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, aggregateSlice)
}

func runByInstanceTypeShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	instanceType := args[0]

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	// Fetch all instances to aggregate by type
	topN := int64(10000)
	opts.TopNInstances = &topN

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	agg, exists := aggregates[instanceType]
	if !exists {
		fmt.Printf("No cost data found for instance type: %s\n", instanceType)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{agg})
}

func runByInstanceTypeTop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var topN int
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
	// Fetch all instances to aggregate by type
	allInstances := int64(10000)
	opts.TopNInstances = &allInstances

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found")
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	if len(aggregates) == 0 {
		fmt.Println("No instance type cost data found")
		return nil
	}

	// Sort by cost and take top N
	sortedAggregates := make([]*InstanceTypeAggregate, 0, len(aggregates))
	for _, agg := range aggregates {
		sortedAggregates = append(sortedAggregates, agg)
	}
	sort.Slice(sortedAggregates, func(i, j int) bool {
		return sortedAggregates[i].TotalCost > sortedAggregates[j].TotalCost
	})

	// Take top N
	if topN > len(sortedAggregates) {
		topN = len(sortedAggregates)
	}
	topAggregates := sortedAggregates[:topN]

	// Convert to interface slice for output
	aggregateSlice := make([]interface{}, len(topAggregates))
	for i, agg := range topAggregates {
		aggregateSlice[i] = agg
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, aggregateSlice)
}

func runByInstanceTypeInCell(cmd *cobra.Command, args []string) error {
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
	topN := int64(10000)
	opts.TopNInstances = &topN

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for deployment cell: %s\n", cellID)
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	if len(aggregates) == 0 {
		fmt.Printf("No instance type cost data found for deployment cell: %s\n", cellID)
		return nil
	}

	// Convert to slice for output
	aggregateSlice := make([]interface{}, 0, len(aggregates))
	for _, agg := range aggregates {
		aggregateSlice = append(aggregateSlice, agg)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, aggregateSlice)
}

func runByInstanceTypeInProvider(cmd *cobra.Command, args []string) error {
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
	topN := int64(10000)
	opts.TopNInstances = &topN

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for provider: %s\n", providerID)
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	if len(aggregates) == 0 {
		fmt.Printf("No instance type cost data found for provider: %s\n", providerID)
		return nil
	}

	// Convert to slice for output
	aggregateSlice := make([]interface{}, 0, len(aggregates))
	for _, agg := range aggregates {
		aggregateSlice = append(aggregateSlice, agg)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, aggregateSlice)
}

func runByInstanceTypeInRegion(cmd *cobra.Command, args []string) error {
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

	// Handle region ID that might contain commas
	regionIDClean := strings.TrimSpace(regionID)

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeRegionIDs = &regionIDClean
	topN := int64(10000)
	opts.TopNInstances = &topN

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for region: %s\n", regionID)
		return nil
	}

	// Aggregate by instance type
	aggregates := aggregateInstanceTypeCosts(result)
	if len(aggregates) == 0 {
		fmt.Printf("No instance type cost data found for region: %s\n", regionID)
		return nil
	}

	// Convert to slice for output
	aggregateSlice := make([]interface{}, 0, len(aggregates))
	for _, agg := range aggregates {
		aggregateSlice = append(aggregateSlice, agg)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, aggregateSlice)
}
