package agent

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "agent [operation] [flags]",
	Short:        "Manage AI agent configurations and skills",
	Long:         `This command helps you manage AI agent configurations and Claude Code skills for Omnistrate.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(initCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
