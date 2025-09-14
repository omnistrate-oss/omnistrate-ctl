package alarms

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/alarms/notificationchannel"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "alarms [operation] [flags]",
	Short:        "Manage alarms and notification channels",
	Long:         `This command helps you manage alarms and notification channels for your services.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(notificationchannel.Cmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
