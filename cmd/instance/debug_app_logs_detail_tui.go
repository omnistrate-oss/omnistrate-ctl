package instance

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type appLogsDetailModel struct {
	node         PlanDAGNode
	debugData    DebugData
	width        int
	height       int
	spinner      spinner.Model
	appLogs      *appLogsState
	clipboardMsg string
}

func newAppLogsDetailModel(node PlanDAGNode, data DebugData) appLogsDetailModel {
	return appLogsDetailModel{
		node:      node,
		debugData: data,
		spinner:   newResourceDetailSpinner(),
		appLogs:   newAppLogsState(appLogStreamsForResource(data, node)),
	}
}

func (m appLogsDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.appLogs.start())
}

func (m appLogsDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		if m.appLogs != nil && m.appLogs.streaming {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case appLogLineMsg:
		return m, m.appLogs.handleLine(msg, resourceDetailBodyHeight(m.height), resourceDetailContentWidth(m.width))
	case appLogStreamDoneMsg:
		m.appLogs.handleDone(msg)
	case clipboardResultMsg:
		if msg.err != nil {
			m.clipboardMsg = fmt.Sprintf("✗ %v", msg.err)
		} else {
			m.clipboardMsg = "✓ Copied to clipboard"
		}
		return m, nil
	case tea.KeyMsg:
		m.clipboardMsg = ""
		switch msg.String() {
		case "ctrl+c", "q":
			m.appLogs.stop()
			return m, tea.Quit
		case "esc":
			m.appLogs.stop()
			return m, func() tea.Msg { return backToDagMsg{} }
		case "up", "k":
			m.appLogs.scrollUp()
		case "down", "j":
			m.appLogs.scrollDown(resourceDetailBodyHeight(m.height), resourceDetailContentWidth(m.width))
		case "pgup":
			m.appLogs.pageUp(resourceDetailBodyHeight(m.height))
		case "pgdown":
			m.appLogs.pageDown(resourceDetailBodyHeight(m.height), resourceDetailContentWidth(m.width))
		case "f":
			m.appLogs.toggleFollow(resourceDetailBodyHeight(m.height), resourceDetailContentWidth(m.width))
		case "y":
			content := m.appLogs.copyText()
			if content != "" {
				m.clipboardMsg = "Copying..."
				return m, copyToClipboardCmd(content)
			}
		}
	}
	return m, nil
}

func (m appLogsDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	header := renderResourceDetailHeader(m.width, m.node)
	body := renderResourceDetailTabsWithBody(
		m.width,
		resourceDetailBodyHeight(m.height),
		[]string{"App Logs"},
		0,
		renderAppLogsTab(m.appLogs, appLogsUnavailableMessage(m.debugData, m.node), m.spinner.View(), resourceDetailBodyHeight(m.height), resourceDetailContentWidth(m.width)),
	)
	footer := renderResourceDetailFooter(m.width, m.clipboardMsg, "↑↓/pgup/pgdn: scroll  f: toggle follow  y: copy  esc: back  q: quit")
	return header + "\n" + body + "\n" + footer
}
