package serviceplan

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "service-plan [operation] [flags]",
	Short:        "Manage Service Plans for your service",
	Long:         `This command helps you manage the service plans for your service.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(releaseCmd)
	Cmd.AddCommand(setDefaultCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(describeVersionCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(listVersionsCmd)
	Cmd.AddCommand(enableCmd)
	Cmd.AddCommand(disableCmd)
	Cmd.AddCommand(updateCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
