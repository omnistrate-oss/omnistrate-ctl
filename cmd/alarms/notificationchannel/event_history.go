package notificationchannel

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
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

// --- Styles ---

var (
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")) // yellow
	headingStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	focusedBorder      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("2")) // green
	unfocusedBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")) // grey
	helpStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	statusFailStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	statusPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	statusDefaultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
)

func eventStatusColor(status string) lipgloss.Style {
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "success") || strings.Contains(s, "delivered"):
		return statusSuccessStyle
	case strings.Contains(s, "failed") || strings.Contains(s, "error"):
		return statusFailStyle
	case strings.Contains(s, "pending") || strings.Contains(s, "retry"):
		return statusPendingStyle
	default:
		return statusDefaultStyle
	}
}

// --- List items ---

// eventItem represents a top-level event in the list.
type eventItem struct {
	event openapiclientfleet.Event
	index int
}

func (i eventItem) Title() string {
	idShort := i.event.GetId()
	if len(idShort) > 8 {
		idShort = idShort[:8] + "..."
	}
	return fmt.Sprintf("Event %d (%s)", i.index+1, idShort)
}

func (i eventItem) Description() string {
	return eventStatusColor(i.event.GetPublicationStatus()).Render(i.event.GetPublicationStatus())
}

func (i eventItem) FilterValue() string { return i.event.GetId() }

// subMenuItem represents a child option (Overview, Event Body, Channel Response).
type subMenuItem struct {
	label string
	event openapiclientfleet.Event
}

func (i subMenuItem) Title() string       { return i.label }
func (i subMenuItem) Description() string { return "" }
func (i subMenuItem) FilterValue() string { return i.label }

// --- Delegate ---

type eventDelegate struct{}

func (d eventDelegate) Height() int                             { return 2 }
func (d eventDelegate) Spacing() int                            { return 0 }
func (d eventDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d eventDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title, desc string
	switch it := item.(type) {
	case eventItem:
		title = it.Title()
		desc = it.Description()
	case subMenuItem:
		title = "  " + it.Title()
	}

	isSelected := index == m.Index()
	if isSelected {
		title = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true).Render("> " + title)
	} else {
		title = "  " + title
	}

	fmt.Fprintf(w, "%s\n", title)
	if desc != "" {
		fmt.Fprintf(w, "  %s\n", desc)
	}
}

// --- Focus enum ---

type focusPane int

const (
	focusList focusPane = iota
	focusViewport
)

// --- View mode ---

type viewMode int

const (
	modeEventList viewMode = iota
	modeSubMenu
)

// --- Bubbletea model ---

type eventHistoryModel struct {
	events    []openapiclientfleet.Event
	channelID string

	list     list.Model
	viewport viewport.Model

	focus     focusPane
	mode      viewMode
	parentIdx int // index of selected event when in sub-menu mode

	width  int
	height int
	ready  bool
}

func newEventHistoryModel(events []openapiclientfleet.Event, channelID string) eventHistoryModel {
	items := make([]list.Item, len(events))
	for i, ev := range events {
		items[i] = eventItem{event: ev, index: i}
	}

	delegate := eventDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("Channel: %s", channelID)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	vp := viewport.New(0, 0)
	vp.SetContent("Select an event to view content")

	return eventHistoryModel{
		events:    events,
		channelID: channelID,
		list:      l,
		viewport:  vp,
		focus:     focusList,
		mode:      modeEventList,
	}
}

func (m eventHistoryModel) Init() tea.Cmd {
	return nil
}

func (m eventHistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate to focused component
	return m.delegateUpdate(msg)
}

func (m eventHistoryModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "Q"))):
		return m, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		return m, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		if m.focus == focusList {
			m.focus = focusViewport
		} else {
			m.focus = focusList
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		return m.handleEnter()

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		return m.handleEsc()
	}

	return m.delegateUpdate(msg)
}

func (m eventHistoryModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.focus == focusList {
		selected := m.list.SelectedItem()
		if selected == nil {
			return m, nil
		}
		switch it := selected.(type) {
		case eventItem:
			// Drill into sub-menu for this event
			m.parentIdx = m.list.Index()
			m.mode = modeSubMenu
			subItems := m.buildSubMenu(it.event)
			cmd := m.list.SetItems(subItems)
			m.list.Title = it.Title()
			m.list.ResetSelected()
			// Show overview in viewport
			m.viewport.SetContent(formatEventOverview(it.event))
			return m, cmd
		case subMenuItem:
			// Show content in viewport and switch focus
			m.updateViewportContent(it)
			m.focus = focusViewport
			return m, nil
		}
	}
	return m, nil
}

func (m eventHistoryModel) handleEsc() (tea.Model, tea.Cmd) {
	if m.focus == focusViewport {
		m.focus = focusList
		return m, nil
	}
	if m.mode == modeSubMenu {
		// Go back to event list
		m.mode = modeEventList
		items := make([]list.Item, len(m.events))
		for i, ev := range m.events {
			items[i] = eventItem{event: ev, index: i}
		}
		cmd := m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("Channel: %s", m.channelID)
		m.list.Select(m.parentIdx)
		m.viewport.SetContent("Select an event to view content")
		return m, cmd
	}
	return m, tea.Quit
}

func (m *eventHistoryModel) updateViewportContent(item subMenuItem) {
	switch item.label {
	case "Overview":
		m.viewport.SetContent(formatEventOverview(item.event))
	case "Event Body":
		m.viewport.SetContent(formatEventBody(item.event.GetBody()))
	case "Channel Response":
		m.viewport.SetContent(formatChannelResponse(item.event.GetChannelResponse()))
	}
}

func (m eventHistoryModel) buildSubMenu(event openapiclientfleet.Event) []list.Item {
	items := []list.Item{
		subMenuItem{label: "Overview", event: event},
	}
	if event.HasBody() {
		items = append(items, subMenuItem{label: "Event Body", event: event})
	}
	if event.GetChannelResponse() != nil {
		items = append(items, subMenuItem{label: "Channel Response", event: event})
	}
	return items
}

func (m eventHistoryModel) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.focus == focusList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		// Update viewport preview when navigating events
		if m.mode == modeEventList {
			if sel := m.list.SelectedItem(); sel != nil {
				if it, ok := sel.(eventItem); ok {
					m.viewport.SetContent(formatEventOverview(it.event))
				}
			}
		}
		return m, cmd
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *eventHistoryModel) updateLayout() {
	helpH := 1
	listW := m.width / 3
	contentW := m.width - listW
	innerH := m.height - helpH

	// Account for borders (2 top/bottom)
	m.list.SetSize(listW-2, innerH-2)
	m.viewport.Width = contentW - 2
	m.viewport.Height = innerH - 2
}

func (m eventHistoryModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	helpH := 1
	listW := m.width / 3
	contentW := m.width - listW
	innerH := m.height - helpH

	// Style panels based on focus
	var leftStyle, rightStyle lipgloss.Style
	if m.focus == focusList {
		leftStyle = focusedBorder.Width(listW - 2).Height(innerH - 2)
		rightStyle = unfocusedBorder.Width(contentW - 2).Height(innerH - 2)
	} else {
		leftStyle = unfocusedBorder.Width(listW - 2).Height(innerH - 2)
		rightStyle = focusedBorder.Width(contentW - 2).Height(innerH - 2)
	}

	leftPanel := leftStyle.Render(m.list.View())
	rightPanel := rightStyle.Render(m.viewport.View())

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	help := helpStyle.Render("↑/↓: navigate • enter: select • tab: switch panel • esc: back • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, panels, help)
}

func showEventHistoryTUI(events []openapiclientfleet.Event, channelID string) error {
	m := newEventHistoryModel(events, channelID)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}

// --- Format helpers ---

func formatEventOverview(event openapiclientfleet.Event) string {
	heading := headingStyle.Render("Event Overview")
	return fmt.Sprintf(`%s

ID: %s
Publication Status: %s
Timestamp: %s

Has Body: %t
Has Channel Response: %t

Select "Event Body" or "Channel Response" from the menu to view detailed content.`,
		heading,
		event.GetId(),
		eventStatusColor(event.GetPublicationStatus()).Render(event.GetPublicationStatus()),
		event.GetTimestamp().Format(time.RFC3339),
		event.HasBody(),
		event.GetChannelResponse() != nil)
}

func formatEventBody(body interface{}) string {
	heading := headingStyle.Render("Event Body")
	if body == nil {
		return heading + "\n\nNo event body available"
	}

	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return heading + fmt.Sprintf("\n\nError formatting body: %v", err)
	}
	return heading + "\n\n" + string(jsonBytes)
}

func formatChannelResponse(response interface{}) string {
	heading := headingStyle.Render("Channel Response")
	if response == nil {
		return heading + "\n\nNo channel response available"
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return heading + fmt.Sprintf("\n\nError formatting response: %v", err)
	}
	return heading + "\n\n" + string(jsonBytes)
}
