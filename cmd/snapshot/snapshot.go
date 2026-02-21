package snapshot

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "snapshot [operation] [flags]",
	Short:        "Manage instance snapshots and backups",
	Long:         `This command helps you manage snapshots for your service instances, including creating, copying, listing, describing, deleting, and restoring snapshots.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(restoreCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
