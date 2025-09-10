package workflow

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workflow [operation] [flags]",
	Short: "Manage service workflows",
	Long: `This command helps you manage workflows for your services.
You can list, describe, get events, and terminate workflows.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(summaryCmd)
	Cmd.AddCommand(eventsCmd)
	Cmd.AddCommand(terminateCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
