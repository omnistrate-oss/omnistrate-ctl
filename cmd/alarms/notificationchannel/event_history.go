package notificationchannel

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

var eventHistoryCmd = &cobra.Command{
	Use:   "event-history [channel-id]",
	Short: "Show event history for a notification channel with interactive TUI",
	Long:  `Display event history for a notification channel in an interactive table interface that allows expanding rows to see event details.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEventHistory,
}

var (
	startTimeFlag string
	endTimeFlag   string
)

func init() {
	eventHistoryCmd.Flags().StringVarP(&startTimeFlag, "start-time", "s", "", "Start time for event history (RFC3339 format)")
	eventHistoryCmd.Flags().StringVarP(&endTimeFlag, "end-time", "e", "", "End time for event history (RFC3339 format)")
}

func runEventHistory(cmd *cobra.Command, args []string) error {
	channelID := args[0]
	
	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var startTime, endTime *time.Time
	if startTimeFlag != "" {
		t, err := time.Parse(time.RFC3339, startTimeFlag)
		if err != nil {
			return fmt.Errorf("invalid start time format: %v", err)
		}
		startTime = &t
	}
	
	if endTimeFlag != "" {
		t, err := time.Parse(time.RFC3339, endTimeFlag)
		if err != nil {
			return fmt.Errorf("invalid end time format: %v", err)
		}
		endTime = &t
	}

	result, err := dataaccess.GetNotificationChannelEventHistory(cmd.Context(), token, channelID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get event history: %v", err)
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

	return showEventHistoryTUI(result.GetEvents(), channelID)
}

func showEventHistoryTUI(events []openapiclientfleet.Event, channelID string) error {
	app := tview.NewApplication()

	// Create main layout
	flex := tview.NewFlex()

	// Left panel - Events (tree view)
	leftPanel := tview.NewTreeView()
	leftPanel.SetBorder(true).SetTitle("Events")

	// Create root node
	root := tview.NewTreeNode(fmt.Sprintf("Channel: %s", channelID))
	root.SetColor(tcell.ColorYellow)
	leftPanel.SetRoot(root)

	// Add events as tree nodes
	for i, event := range events {
		eventLabel := fmt.Sprintf("Event %d", i+1)
		if len(event.GetId()) > 0 {
			// Show first 8 chars of ID for brevity
			idShort := event.GetId()
			if len(idShort) > 8 {
				idShort = idShort[:8] + "..."
			}
			eventLabel = fmt.Sprintf("Event %d (%s)", i+1, idShort)
		}
		
		eventNode := tview.NewTreeNode(eventLabel)
		eventNode.SetReference(event)
		eventNode.SetColor(getEventColor(event))

		// Add sub-options for the event
		if event.HasBody() {
			bodyNode := tview.NewTreeNode("Event Body")
			bodyNode.SetReference(map[string]interface{}{
				"type":  "event-body",
				"event": event,
			})
			bodyNode.SetColor(tcell.ColorGreen)
			eventNode.AddChild(bodyNode)
		}

		if event.GetChannelResponse() != nil {
			responseNode := tview.NewTreeNode("Channel Response")
			responseNode.SetReference(map[string]interface{}{
				"type":  "channel-response",
				"event": event,
			})
			responseNode.SetColor(tcell.ColorGreen)
			eventNode.AddChild(responseNode)
		}

		root.AddChild(eventNode)
	}

	root.SetExpanded(true)

	// Right panel - Content
	rightPanel := tview.NewTextView()
	rightPanel.SetBorder(true).SetTitle("Content")
	rightPanel.SetDynamicColors(true)
	rightPanel.SetWrap(true)
	rightPanel.SetScrollable(true)
	rightPanel.SetText("Select an event or event detail to view content")

	// Add focus handlers to show which panel is active
	leftPanel.SetFocusFunc(func() {
		leftPanel.SetBorderColor(tcell.ColorGreen)
		rightPanel.SetBorderColor(tcell.ColorDefault)
	})
	rightPanel.SetFocusFunc(func() {
		rightPanel.SetBorderColor(tcell.ColorGreen)
		leftPanel.SetBorderColor(tcell.ColorDefault)
	})

	// Handle tree selection
	leftPanel.SetChangedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			rightPanel.SetTitle("Content")
			rightPanel.SetText("Select an event or event detail to view content")
			return
		}

		switch ref := reference.(type) {
		case openapiclientfleet.Event:
			// Show event overview
			content := formatEventOverview(ref)
			rightPanel.SetTitle(fmt.Sprintf("Event: %s", ref.GetId()))
			rightPanel.SetText(content)
		case map[string]interface{}:
			handleEventOptionSelection(ref, rightPanel)
		}
	})

	// Also handle direct selection (Enter key)
	leftPanel.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference != nil {
			switch ref := reference.(type) {
			case openapiclientfleet.Event:
				content := formatEventOverview(ref)
				rightPanel.SetTitle(fmt.Sprintf("Event: %s", ref.GetId()))
				rightPanel.SetText(content)
			case map[string]interface{}:
				handleEventOptionSelection(ref, rightPanel)
				return // Don't toggle expansion for options
			}
		}
		// Toggle expansion for event nodes
		node.SetExpanded(!node.IsExpanded())
	})

	// Set up layout
	flex.AddItem(leftPanel, 0, 1, true)
	flex.AddItem(rightPanel, 0, 2, false)

	// Create main layout with help text
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainFlex.AddItem(flex, 0, 1, true)
	mainFlex.AddItem(createEventHelpText(), 1, 0, false)

	// Create main input handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyEnter:
			// Switch from left panel to right panel to view content
			if app.GetFocus() == leftPanel {
				app.SetFocus(rightPanel)
				return nil
			}
			// If on right panel, let default behavior handle scrolling
		case tcell.KeyEscape:
			if app.GetFocus() == rightPanel {
				// Go back to left panel from right panel
				app.SetFocus(leftPanel)
				return nil
			} else {
				// Exit the application
				app.Stop()
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q':
				app.Stop()
				return nil
			}
		}
		return event
	})

	// Set initial focus and selection
	app.SetFocus(leftPanel)

	// Set initial selection to first event if available
	if len(events) > 0 {
		// Find the first event node
		if len(root.GetChildren()) > 0 {
			firstEvent := root.GetChildren()[0]
			leftPanel.SetCurrentNode(firstEvent)
			// Expand the first event to show its options
			firstEvent.SetExpanded(true)
		}
	}

	// Start the application (disable mouse to allow terminal text selection)
	if err := app.SetRoot(mainFlex, true).EnableMouse(false).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func getEventColor(event openapiclientfleet.Event) tcell.Color {
	status := strings.ToLower(event.GetPublicationStatus())
	switch {
	case strings.Contains(status, "success") || strings.Contains(status, "delivered"):
		return tcell.ColorGreen
	case strings.Contains(status, "failed") || strings.Contains(status, "error"):
		return tcell.ColorRed
	case strings.Contains(status, "pending") || strings.Contains(status, "retry"):
		return tcell.ColorYellow
	default:
		return tcell.ColorBlue
	}
}

func createEventHelpText() *tview.TextView {
	helpText := tview.NewTextView()
	helpText.SetText("Navigate: ↑/↓ to move | Enter: view content/expand | Esc: go back/exit | q: quit")
	helpText.SetTextAlign(tview.AlignCenter)
	helpText.SetDynamicColors(true)
	return helpText
}

func handleEventOptionSelection(ref map[string]interface{}, rightPanel *tview.TextView) {
	optionType, _ := ref["type"].(string)
	event, _ := ref["event"].(openapiclientfleet.Event)

	switch optionType {
	case "event-body":
		if event.HasBody() {
			content := formatEventBody(event.GetBody())
			rightPanel.SetTitle("Event Body")
			rightPanel.SetText(content)
		}
	case "channel-response":
		if event.GetChannelResponse() != nil {
			content := formatChannelResponse(event.GetChannelResponse())
			rightPanel.SetTitle("Channel Response")
			rightPanel.SetText(content)
		}
	}
}

func formatEventOverview(event openapiclientfleet.Event) string {
	return fmt.Sprintf(`[yellow]Event Overview[white]

ID: %s
Publication Status: %s
Timestamp: %s

Has Body: %t
Has Channel Response: %t

Select "Event Body" or "Channel Response" from the tree to view detailed content.`,
		event.GetId(),
		event.GetPublicationStatus(),
		event.GetTimestamp().Format(time.RFC3339),
		event.HasBody(),
		event.GetChannelResponse() != nil)
}

func formatEventBody(body interface{}) string {
	if body == nil {
		return "[yellow]Event Body[white]\n\nNo event body available"
	}

	content := "[yellow]Event Body[white]\n\n"
	
	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err == nil {
		// Apply JSON syntax highlighting
		highlightedContent := addJSONSyntaxHighlighting(string(jsonBytes))
		content += highlightedContent
	} else {
		content += fmt.Sprintf("Error formatting body: %v", err)
	}

	return content
}

func formatChannelResponse(response interface{}) string {
	if response == nil {
		return "[yellow]Channel Response[white]\n\nNo channel response available"
	}

	content := "[yellow]Channel Response[white]\n\n"
	
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err == nil {
		// Apply JSON syntax highlighting
		highlightedContent := addJSONSyntaxHighlighting(string(jsonBytes))
		content += highlightedContent
	} else {
		content += fmt.Sprintf("Error formatting response: %v", err)
	}

	return content
}

// addJSONSyntaxHighlighting adds basic syntax highlighting for JSON content
func addJSONSyntaxHighlighting(content string) string {
	// Simple approach - just return the content without color tags
	// The tview TextView will handle proper display of plain JSON
	return content
}