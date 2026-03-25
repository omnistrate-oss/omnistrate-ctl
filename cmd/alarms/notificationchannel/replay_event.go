package notificationchannel

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var replayEventCmd = &cobra.Command{
	Use:   "replay-event [event-id]",
	Short: "Replay a specific event to notification channels",
	Long:  `Replay a specific event by its ID to all configured notification channels.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runReplayEvent,
}

func runReplayEvent(cmd *cobra.Command, args []string) error {
	eventID := args[0]

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	err = dataaccess.ReplayNotificationEvent(cmd.Context(), token, eventID)
	if err != nil {
		return fmt.Errorf("failed to replay event: %v", err)
	}

	fmt.Printf("Successfully replayed event: %s\n", eventID)
	return nil
}
