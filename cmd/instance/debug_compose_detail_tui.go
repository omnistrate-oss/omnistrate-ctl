package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	composeTabInputVars  = 0
	composeTabOutputVars = 1
	composeTabWfErrors   = 2
	composeNumTabs       = 3
)

var composeTabNames = []string{"Deployment API parameters", "Deployment Output Parameters", "Workflow Events"}

func init() {
	if len(composeTabNames) != composeNumTabs {
		panic(fmt.Sprintf("composeTabNames length %d does not match composeNumTabs %d", len(composeTabNames), composeNumTabs))
	}
}

// composeDataMsg is sent when compose debug data has been fetched.
type composeDataMsg struct {
	composeData *ComposeData
	inputErr    error
	outputErr   error
}

type composeDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int

	loading   bool
	loadErr   error
	inputErr  error
	outputErr error
	spinner   spinner.Model

	composeData *ComposeData

	// Input variables tree
	inputTree   []outputNode
	inputCursor int
	inputScroll int

	// Output variables tree
	outputTree   []outputNode
	outputCursor int
	outputScroll int

	// Workflow Events tab
	wfErrors *workflowErrorsState

	clipboardMsg string
}

func newComposeDetailModel(node PlanDAGNode, data DebugData) composeDetailModel {
	return composeDetailModel{
		node:      node,
		debugData: data,
		activeTab: composeTabInputVars,
		loading:   true,
		spinner:   newResourceDetailSpinner(),
		wfErrors:  &workflowErrorsState{},
	}
}

func (m composeDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchComposeData())
}

func (m composeDetailModel) fetchComposeData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		cData := &ComposeData{}

		// Fetch all input parameters
		inputParams, inputErr := fetchInputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.InputParams,
		)
		if inputErr == nil {
			cData.InputParams = inputParams
		}

		// Fetch exported output parameters
		outputParams, outputErr := fetchOutputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.ResultParams,
		)
		if outputErr == nil {
			cData.OutputParams = outputParams
		}

		return composeDataMsg{
			composeData: cData,
			inputErr:    inputErr,
			outputErr:   outputErr,
		}
	}
}

func (m composeDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.wfErrors.refreshing || isWorkflowInProgress(m.getWfEvents()) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case composeDataMsg:
		m.loading = false
		m.inputErr = msg.inputErr
		m.outputErr = msg.outputErr
		m.composeData = msg.composeData
		if m.composeData != nil {
			m.inputTree = buildOperatorParamTree(m.composeData.InputParams)
			m.outputTree = buildOperatorOutputParamTree(m.composeData.OutputParams)
		}
		return m, scheduleResourceWorkflowRefreshIfNeeded(m.debugData, m.node)

	case wfEventsRefreshTickMsg:
		return m, handleResourceWorkflowRefreshTick(m.debugData, m.node, m.wfErrors)
	case wfEventsRefreshMsg:
		return m, handleResourceWorkflowRefresh(m.debugData, m.node, m.wfErrors, msg)
	case wfCountdownTickMsg:
		return m, handleResourceWorkflowCountdown(m.debugData, m.node)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalText = ""
				m.wfErrors.modalTitle = ""
				m.wfErrors.modalScroll = 0
				return m, nil
			}
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab + 1) % composeNumTabs
			return m, nil
		case "shift+tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab - 1 + composeNumTabs) % composeNumTabs
			return m, nil
		case "up", "k":
			if m.wfErrors.modalText != "" {
				if m.wfErrors.modalScroll > 0 {
					m.wfErrors.modalScroll--
				}
				return m, nil
			}
			switch m.activeTab {
			case composeTabInputVars:
				m.inputCursor, m.inputScroll = moveResourceDetailTreeUp(m.inputTree, m.inputCursor, m.inputScroll, m.composeVisibleRows())
			case composeTabOutputVars:
				m.outputCursor, m.outputScroll = moveResourceDetailTreeUp(m.outputTree, m.outputCursor, m.outputScroll, m.composeVisibleRows())
			case composeTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor > 0 {
					m.wfErrors.cursor--
				}
				_ = items
			}
		case "down", "j":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalScroll++
				maxScroll := wfEventModalMaxScroll(m.wfErrors, m.width, m.height)
				if m.wfErrors.modalScroll > maxScroll {
					m.wfErrors.modalScroll = maxScroll
				}
				return m, nil
			}
			switch m.activeTab {
			case composeTabInputVars:
				m.inputCursor, m.inputScroll = moveResourceDetailTreeDown(m.inputTree, m.inputCursor, m.inputScroll, m.composeVisibleRows())
			case composeTabOutputVars:
				m.outputCursor, m.outputScroll = moveResourceDetailTreeDown(m.outputTree, m.outputCursor, m.outputScroll, m.composeVisibleRows())
			case composeTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items)-1 {
					m.wfErrors.cursor++
				}
				_ = items
			}
		case "enter":
			switch m.activeTab {
			case composeTabInputVars:
				toggleResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case composeTabOutputVars:
				toggleResourceDetailTreeNode(m.outputTree, m.outputCursor)
			case composeTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items) {
					item := items[m.wfErrors.cursor]
					if item.event != nil {
						m.wfErrors.modalText = formatEventDetail(item.event)
						m.wfErrors.modalTitle = extractEventAction(item.event.Message)
						m.wfErrors.modalScroll = 0
					}
				}
			}
		case "right", "l":
			switch m.activeTab {
			case composeTabInputVars:
				expandResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case composeTabOutputVars:
				expandResourceDetailTreeNode(m.outputTree, m.outputCursor)
			}
		case "left", "h":
			switch m.activeTab {
			case composeTabInputVars:
				collapseResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case composeTabOutputVars:
				collapseResourceDetailTreeNode(m.outputTree, m.outputCursor)
			}
		case "pgup":
			switch m.activeTab {
			case composeTabWfErrors:
				m.wfErrors.cursor -= m.composeVisibleRows()
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "pgdown":
			switch m.activeTab {
			case composeTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				m.wfErrors.cursor += m.composeVisibleRows()
				if m.wfErrors.cursor >= len(items) {
					m.wfErrors.cursor = len(items) - 1
				}
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "y":
			content := m.composeCopyableContent()
			if content != "" {
				return m, copyToClipboardCmd(content)
			}
		}

	case clipboardResultMsg:
		if msg.err != nil {
			m.clipboardMsg = fmt.Sprintf("✗ %v", msg.err)
		} else {
			m.clipboardMsg = "✓ Copied to clipboard"
		}
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearClipboardMsg{} })
	case clearClipboardMsg:
		m.clipboardMsg = ""
	}
	return m, nil
}

func (m composeDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.wfErrors.modalText != "" {
		return renderWfEventModal(m.wfErrors, m.width, m.height)
	}

	header := m.renderComposeHeader()
	tabs := m.renderComposeTabsWithBody()
	footer := m.renderComposeFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, footer)
}

func (m composeDetailModel) renderComposeHeader() string {
	return renderResourceDetailHeader(m.width, m.node)
}

func (m composeDetailModel) renderComposeTabsWithBody() string {
	return renderResourceDetailTabsWithBody(m.width, m.composeBodyHeight(), composeTabNames, m.activeTab, m.getComposeTabContent())
}

func (m composeDetailModel) getComposeTabContent() string {
	switch m.activeTab {
	case composeTabInputVars:
		return m.renderComposeInputVarsTab()
	case composeTabOutputVars:
		return m.renderComposeOutputVarsTab()
	case composeTabWfErrors:
		return m.renderComposeWfErrorsTab()
	}
	return ""
}

func (m composeDetailModel) renderComposeInputVarsTab() string {
	return m.renderComposeParamTreeTab("Deployment API parameters", m.inputTree, m.inputCursor, m.inputScroll, m.inputErr)
}

func (m composeDetailModel) renderComposeOutputVarsTab() string {
	return m.renderComposeParamTreeTab("Deployment Output Parameters", m.outputTree, m.outputCursor, m.outputScroll, m.outputErr)
}

func (m composeDetailModel) renderComposeParamTreeTab(title string, tree []outputNode, cursor, scroll int, fetchErr error) string {
	return renderResourceDetailParamTreeTab(title, tree, cursor, scroll, m.composeVisibleRows(), m.composeContentWidth(), m.loading, m.spinner.View(), m.loadErr, fetchErr, false)
}

func (m composeDetailModel) renderComposeWfErrorsTab() string {
	return renderResourceWorkflowEventsTab(m.debugData, m.node, m.wfErrors, m.composeBodyHeight(), m.composeContentWidth(), m.spinner.View())
}

func (m composeDetailModel) renderComposeFooter() string {
	var text string
	switch m.activeTab {
	case composeTabInputVars, composeTabOutputVars:
		if (m.activeTab == composeTabInputVars && len(m.inputTree) > 0) ||
			(m.activeTab == composeTabOutputVars && len(m.outputTree) > 0) {
			text = "↑↓: navigate  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
		} else {
			text = "tab/shift+tab: switch tabs  esc: back  q: quit"
		}
	case composeTabWfErrors:
		text = "↑↓/pgup/pgdn: scroll  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	default:
		text = "tab/shift+tab: switch tabs  esc: back  q: quit"
	}
	return renderResourceDetailFooter(m.width, m.clipboardMsg, text)
}

func (m composeDetailModel) composeCopyableContent() string {
	switch m.activeTab {
	case composeTabInputVars:
		if m.composeData != nil && len(m.composeData.InputParams) > 0 {
			raw, err := json.Marshal(m.composeData.InputParams)
			if err == nil {
				return string(raw)
			}
		}
	case composeTabOutputVars:
		if m.composeData != nil && len(m.composeData.OutputParams) > 0 {
			raw, err := json.Marshal(m.composeData.OutputParams)
			if err == nil {
				return string(raw)
			}
		}
	case composeTabWfErrors:
		return workflowEventsCopyText(m.getWfEvents())
	}
	return ""
}

func (m composeDetailModel) composeBodyHeight() int {
	return resourceDetailBodyHeight(m.height)
}

func (m composeDetailModel) composeVisibleRows() int {
	return resourceDetailVisibleRows(m.height)
}

func (m composeDetailModel) composeContentWidth() int {
	return resourceDetailContentWidth(m.width)
}

func (m composeDetailModel) getWfEvents() *ResourceWorkflowSteps {
	return getResourceWorkflowEvents(m.debugData, m.node)
}
