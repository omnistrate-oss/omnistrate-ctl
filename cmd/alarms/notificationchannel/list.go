package notificationchannel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notification channels",
	Long:  `Display a table of all configured notification channels showing their ID, name, type, and subscription details.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	result, err := dataaccess.ListNotificationChannels(cmd.Context(), token)
	if err != nil {
		return fmt.Errorf("failed to list notification channels: %v", err)
	}

	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat == "json" {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}

	channels := result.GetChannels()
	if len(channels) == 0 {
		fmt.Println("No notification channels found.")
		return nil
	}

	return displayChannelsTable(channels)
}

func displayChannelsTable(channels []openapiclientfleet.Channel) error {
	// Create table with appropriate columns
	table := utils.NewTable([]any{"ID", "Name", "Type", "Event Categories", "Event Priorities", "Alert Types"})

	for _, channel := range channels {
		subscription := channel.GetSubscription()
		
		// Format arrays as comma-separated strings, truncating if too long
		eventCategories := formatStringArray(subscription.GetEventCategories(), 30)
		eventPriorities := formatStringArray(subscription.GetEventPriorities(), 20)
		alertTypes := formatStringArray(subscription.GetAlertTypes(), 20)

		table.AddRow([]any{
			channel.GetId(),
			channel.GetName(),
			channel.GetChannelType(),
			eventCategories,
			eventPriorities,
			alertTypes,
		})
	}

	table.Print()
	return nil
}

// formatStringArray formats a string array for table display, truncating if necessary
func formatStringArray(arr []string, maxLength int) string {
	if len(arr) == 0 {
		return "-"
	}
	
	joined := strings.Join(arr, ", ")
	if len(joined) <= maxLength {
		return joined
	}
	
	// Truncate and add ellipsis
	return joined[:maxLength-3] + "..."
}