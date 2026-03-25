package account

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "account [operation] [flags]",
	Short:        "Manage your Cloud Provider Accounts",
	Long:         `This command helps you manage your Cloud Provider Accounts.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(deleteCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
