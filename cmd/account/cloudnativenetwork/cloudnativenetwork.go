package cloudnativenetwork

import (
	"github.com/spf13/cobra"
)

// Cmd is the top-level "account cloud-native-network" subcommand.
var Cmd = &cobra.Command{
	Use:          "cloud-native-network [operation] [flags]",
	Short:        "Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account",
	Long:         `This command helps you discover, import, and manage cloud-native networks (VPCs) associated with your BYOA cloud provider accounts.`,
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
