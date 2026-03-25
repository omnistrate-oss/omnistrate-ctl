package cost

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var byInstanceCmd = &cobra.Command{
	Use:   "by-instance",
	Short: "Get cost breakdown by individual instance",
	Long:  "Analyze costs for individual instances across your fleet.",
}

var byInstanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List costs for all instances",
	Long:  "Get cost breakdown for all instances in your account.",
	RunE:  runByInstanceList,
}

var byInstanceShowCmd = &cobra.Command{
	Use:   "show <instance-id>",
	Short: "Show costs for a specific instance",
	Long:  "Get detailed cost breakdown for a specific instance.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceShow,
}

var byInstanceTopCmd = &cobra.Command{
	Use:   "top <N>",
	Short: "Get top N most expensive instances",
	Long:  "List the top N most expensive instances.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceTop,
}

var byInstanceInCellCmd = &cobra.Command{
	Use:   "in-cell <cell-id>",
	Short: "List instances in a specific deployment cell",
	Long:  "Get cost breakdown for all instances in a specific deployment cell.",
	Args:  cobra.ExactArgs(1),
	RunE:  runByInstanceInCell,
}

var byInstanceCompareCmd = &cobra.Command{
	Use:   "compare <instance-id-1> <instance-id-2> [instance-id-3...]",
	Short: "Compare costs across multiple instances",
	Long:  "Compare cost breakdown across two or more instances.",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runByInstanceCompare,
}

func init() {
	// Add subcommands
	byInstanceCmd.AddCommand(byInstanceListCmd)
	byInstanceCmd.AddCommand(byInstanceShowCmd)
	byInstanceCmd.AddCommand(byInstanceTopCmd)
	byInstanceCmd.AddCommand(byInstanceInCellCmd)
	byInstanceCmd.AddCommand(byInstanceCompareCmd)

	// Add flags to all subcommands
	for _, cmd := range []*cobra.Command{
		byInstanceListCmd, byInstanceShowCmd, byInstanceTopCmd,
		byInstanceInCellCmd, byInstanceCompareCmd,
	} {
		addCommonTimeFlags(cmd)
	}
}

func runByInstanceList(cmd *cobra.Command, args []string) error {
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
	// Fetch all instances
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

	// Extract all instances from deployment cells
	instances := aggregateInstanceCosts(result)
	if len(instances) == 0 {
		fmt.Println("No instance cost data found")
		return nil
	}

	// Convert to interface slice for output
	instanceSlice := make([]interface{}, len(instances))
	for i, instance := range instances {
		instanceSlice[i] = instance
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, instanceSlice)
}

func runByInstanceShow(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	instanceID := args[0]

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeInstanceIDs = &instanceID

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Printf("No cost data found for instance: %s\n", instanceID)
		return nil
	}

	// Extract instances
	instances := aggregateInstanceCosts(result)
	if len(instances) == 0 {
		fmt.Printf("No cost data found for instance: %s\n", instanceID)
		return nil
	}

	// Find the specific instance
	var targetInstance *interface{}
	for _, instance := range instances {
		if instance.InstanceID == instanceID {
			var iface interface{} = instance
			targetInstance = &iface
			break
		}
	}

	if targetInstance == nil {
		fmt.Printf("No cost data found for instance: %s\n", instanceID)
		return nil
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, []interface{}{*targetInstance})
}

func runByInstanceTop(cmd *cobra.Command, args []string) error {
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

	// Extract instances
	instances := aggregateInstanceCosts(result)
	if len(instances) == 0 {
		fmt.Println("No instance cost data found")
		return nil
	}

	// Convert to interface slice for output
	instanceSlice := make([]interface{}, len(instances))
	for i, instance := range instances {
		instanceSlice[i] = instance
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, instanceSlice)
}

func runByInstanceInCell(cmd *cobra.Command, args []string) error {
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

	// Extract instances
	instances := aggregateInstanceCosts(result)
	if len(instances) == 0 {
		fmt.Printf("No instance cost data found for deployment cell: %s\n", cellID)
		return nil
	}

	// Convert to interface slice for output
	instanceSlice := make([]interface{}, len(instances))
	for i, instance := range instances {
		instanceSlice[i] = instance
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, instanceSlice)
}

func runByInstanceCompare(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	instanceIDs := args

	startDate, endDate, environmentType, frequency, err := parseTimeFlags(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Join instance IDs with comma
	instanceIDsStr := ""
	for i, id := range instanceIDs {
		if i > 0 {
			instanceIDsStr += ","
		}
		instanceIDsStr += id
	}

	opts := buildCostOptions(startDate, endDate, environmentType, frequency)
	opts.IncludeInstanceIDs = &instanceIDsStr

	result, err := dataaccess.DescribeDeploymentCellCost(ctx, token, opts)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell cost: %w", err)
	}

	if result == nil {
		fmt.Println("No cost data found for specified instances")
		return nil
	}

	// Extract instances
	instances := aggregateInstanceCosts(result)
	if len(instances) == 0 {
		fmt.Println("No instance cost data found for specified instances")
		return nil
	}

	// Convert to interface slice for output
	instanceSlice := make([]interface{}, len(instances))
	for i, instance := range instances {
		instanceSlice[i] = instance
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(outputFormat, instanceSlice)
}
