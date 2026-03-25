package environment

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "environment [operation] [flags]",
	Short:        "Manage Service Environments for your service",
	Long:         `This command helps you manage the environments for your service.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(promoteCmd)

}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
