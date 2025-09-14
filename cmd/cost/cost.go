package cost

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cost [operation] [flags]",
	Short: "Manage cost analytics for your services",
	Long: `This command helps you analyze costs for your services across different dimensions.
You can get cost breakdowns by cloud provider, deployment cell, region, or user.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
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
