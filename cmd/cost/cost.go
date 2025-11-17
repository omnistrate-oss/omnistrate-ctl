package cost

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cost [operation] [flags]",
	Short: "Manage cost analytics for your services",
	Long: `This command helps you analyze costs for your services across different dimensions.
You can get cost breakdowns by cloud provider, deployment cell, region, user, instance type, or individual instance.

Available aggregations:
  by-provider       Cost breakdown by cloud provider
  by-cell           Cost breakdown by deployment cell
  by-region         Cost breakdown by region
  by-user           Cost breakdown by user
  by-instance-type  Cost breakdown by instance type (e.g., m5.large)
  by-instance       Cost breakdown by individual instance

Legacy commands (deprecated):
  cloud-provider    Use 'by-provider' instead
  deployment-cell   Use 'by-cell' instead
  region            Use 'by-region' instead
  user              Use 'by-user' instead`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	// New explicit commands
	Cmd.AddCommand(byProviderCmd)
	Cmd.AddCommand(byCellCmd)
	Cmd.AddCommand(byRegionCmd)
	Cmd.AddCommand(byUserCmd)
	Cmd.AddCommand(byInstanceTypeCmd)
	Cmd.AddCommand(byInstanceCmd)

	// Legacy commands (kept for backward compatibility)
	Cmd.AddCommand(cloudProviderCmd)
	Cmd.AddCommand(deploymentCellCmd)
	Cmd.AddCommand(regionCmd)
	Cmd.AddCommand(userCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
