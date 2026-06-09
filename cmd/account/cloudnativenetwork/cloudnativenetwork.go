package cloudnativenetwork

import (
	"github.com/spf13/cobra"
)

// Cmd is the top-level "account cloud-native-network" subcommand.
var Cmd = newCmd(removeCmd, deploymentCellCmd)

// NewCmd returns a fresh cloud-native-network command tree.
func NewCmd() *cobra.Command {
	return newCmd(newRemoveCmd(), newDeploymentCellCmd())
}

func newCmd(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cloud-native-network [operation] [flags]",
		Short:        "Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account",
		Long:         `This command helps you manage cloud-native networks (VPCs) associated with your BYOA cloud provider accounts.`,
		Run:          run,
		SilenceUsage: true,
	}

	cmd.AddCommand(subCommands...)

	return cmd
}

func run(cmd *cobra.Command, args []string) {
	_ = cmd.Help()
}
