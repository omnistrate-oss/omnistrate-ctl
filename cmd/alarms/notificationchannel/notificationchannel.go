package notificationchannel

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "notification-channel [operation] [flags]",
	Short:        "Manage notification channels",
	Long:         `This command helps you manage notification channels for alarms and events.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(eventHistoryCmd)
	Cmd.AddCommand(replayEventCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}