package docs

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "docs [operation] [flags]",
	Short: "Search and access documentation",
	Long: `This command helps you search and access documentation.
You can search through documentation content.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(composeSpecCmd)
	Cmd.AddCommand(systemParametersCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
