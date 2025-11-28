package servicesorchestration

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "services-orchestration [operation] [flags]",
	Short:        "Manage Services Orchestration Deployments across services",
	Long:         `This command helps you manage orchestration of deployment across multiple services.`,
	Deprecated:   "This command is deprecated and will be removed in a future release.",
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(modifyCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
