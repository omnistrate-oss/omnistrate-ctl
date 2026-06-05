package cloudnativenetwork

import (
	"github.com/spf13/cobra"
)

const defaultCommandPath = "account cloud-native-network"

// Cmd is the top-level "account cloud-native-network" subcommand.
var Cmd = NewCommand(defaultCommandPath)

// NewCommand creates a fresh cloud-native-network command tree.
func NewCommand(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cloud-native-network [operation] [flags]",
		Short:        "Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account",
		Long:         `This command helps you manage cloud-native networks (VPCs) associated with your BYOA cloud provider accounts.`,
		Run:          run,
		SilenceUsage: true,
	}
	cmd.AddCommand(newListCmd(commandPath))
	cmd.AddCommand(newSyncCmd(commandPath))
	cmd.AddCommand(newImportCmd(commandPath))
	cmd.AddCommand(newRemoveCmd(commandPath))
	return cmd
}

func run(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}
