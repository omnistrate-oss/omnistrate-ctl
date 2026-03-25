package helm

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "helm [operation] [flags]",
	Short: "Manage Helm Charts for your service",
	Long: `This command helps you manage the templates for your helm charts. 
Omnistrate automatically installs this charts and maintains the deployment of the release in every cloud / region / account your service is active in.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(saveCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(listInstallationsCmd)

}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
