package service

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "service [operation] [flags]",
	Short: "Manage Services for your account",
	Long: `This command helps you manage the services for your account.
You can delete, describe, and get services.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(listCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
