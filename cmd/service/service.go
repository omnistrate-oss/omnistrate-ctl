package service

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/serviceplan"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "service [operation] [flags]",
	Short: "Manage Services for your account",
	Long: `This command helps you manage the services for your account.
You can delete, describe, and get services.`,
	RunE:               run,
	DisableFlagParsing: true,
	SilenceUsage:       true,
}

func init() {
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(newResourceCmd())
	Cmd.AddCommand(serviceplan.NewNestedCommand())
}

func run(cmd *cobra.Command, args []string) error {
	if handled, err := runServiceResourceDeletePath(cmd, args); handled || err != nil {
		return err
	}
	return cmd.Help()
}
