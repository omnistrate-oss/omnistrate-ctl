package vpc

import (
	"github.com/spf13/cobra"
)

// Cmd is the top-level "account vpc" subcommand.
var Cmd = &cobra.Command{
	Use:          "vpc [operation] [flags]",
	Short:        "Manage VPCs for a Cloud Provider Account",
	Long:         `This command helps you discover, import, and manage VPCs associated with your BYOA cloud provider accounts.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(importCmd)
	Cmd.AddCommand(unimportCmd)
	Cmd.AddCommand(bulkImportCmd)
}

func run(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}
