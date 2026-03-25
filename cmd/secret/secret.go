package secret

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "secret [operation] [flags]",
	Short:        "Manage secrets",
	Long:         `This command helps you manage secrets for your services.`,
	Run:          runSecret,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(secretSetCmd)
	Cmd.AddCommand(secretListCmd)
	Cmd.AddCommand(secretGetCmd)
	Cmd.AddCommand(secretDeleteCmd)
}

func runSecret(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
