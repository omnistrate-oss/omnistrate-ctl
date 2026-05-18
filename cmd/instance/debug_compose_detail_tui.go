package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

var composeTabNames = []string{"Input Parameters", "Output Parameters", "Workflow Events"}

func init() {
	if len(composeTabNames) != composeNumTabs {
		panic(fmt.Sprintf("composeTabNames length %d does not match composeNumTabs %d", len(composeTabNames), composeNumTabs))
	}
}

// ComposeData holds debug information specific to compose-type resources.
type ComposeData struct {
	InputParams  []OperatorInputParam  `json:"inputParams,omitempty"`
	OutputParams []OperatorOutputParam `json:"outputParams,omitempty"`
}

// composeDataMsg is sent when compose debug data has been fetched.
type composeDataMsg struct {
	composeData *ComposeData
	err         error
}

type composeDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int

	loading bool
	loadErr error
	spinner spinner.Model

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
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return composeDetailModel{
		node:      node,
		debugData: data,
		activeTab: composeTabInputVars,
		loading:   true,
		spinner:   s,
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

		if inputErr != nil && outputErr != nil {
			return composeDataMsg{err: fmt.Errorf("failed to fetch compose data: %w", inputErr)}
		}

		return composeDataMsg{composeData: cData}
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
		if msg.err != nil {
			m.loadErr = msg.err
			return m, nil
		}
		m.composeData = msg.composeData
		if m.composeData != nil {
			m.inputTree = buildOperatorParamTree(m.composeData.InputParams)
			m.outputTree = buildOperatorOutputParamTree(m.composeData.OutputParams)
		}
		var cmds []tea.Cmd
		if isWorkflowInProgress(m.getWfEvents()) {
			cmds = append(cmds, scheduleWfEventsRefresh(), scheduleWfCountdownTick())
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case wfEventsRefreshTickMsg:
		steps := m.getWfEvents()
		if isWorkflowInProgress(steps) && !m.wfErrors.refreshing {
			m.wfErrors.refreshing = true
			return m, fetchWfEventsForResource(m.debugData, m.node.Key)
		}
	case wfEventsRefreshMsg:
		m.wfErrors.refreshing = false
		m.wfErrors.lastRefresh = time.Now()
		if msg.err == nil && msg.steps != nil {
			if m.debugData.PlanDAG != nil {
				if m.debugData.PlanDAG.WorkflowStepsByKey == nil {
					m.debugData.PlanDAG.WorkflowStepsByKey = make(map[string]*ResourceWorkflowSteps)
				}
				m.debugData.PlanDAG.WorkflowStepsByKey[m.node.Key] = msg.steps
			}
		}
		if isWorkflowInProgress(m.getWfEvents()) {
			return m, tea.Batch(scheduleWfEventsRefresh(), scheduleWfCountdownTick())
		}
	case wfCountdownTickMsg:
		if isWorkflowInProgress(m.getWfEvents()) {
			return m, scheduleWfCountdownTick()
		}

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
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor > 0 {
						m.inputCursor--
					}
					m.inputCursor, m.inputScroll = normalizeViewport(
						m.inputCursor, m.inputScroll, len(visibleNodes), m.composeVisibleRows(),
					)
				}
			case composeTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor > 0 {
						m.outputCursor--
					}
					m.outputCursor, m.outputScroll = normalizeViewport(
						m.outputCursor, m.outputScroll, len(visibleNodes), m.composeVisibleRows(),
					)
				}
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
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes)-1 {
						m.inputCursor++
					}
					m.inputCursor, m.inputScroll = normalizeViewport(
						m.inputCursor, m.inputScroll, len(visibleNodes), m.composeVisibleRows(),
					)
				}
			case composeTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes)-1 {
						m.outputCursor++
					}
					m.outputCursor, m.outputScroll = normalizeViewport(
						m.outputCursor, m.outputScroll, len(visibleNodes), m.composeVisibleRows(),
					)
				}
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
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable {
						visibleNodes[m.inputCursor].expanded = !visibleNodes[m.inputCursor].expanded
					}
				}
			case composeTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable {
						visibleNodes[m.outputCursor].expanded = !visibleNodes[m.outputCursor].expanded
					}
				}
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
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable && !visibleNodes[m.inputCursor].expanded {
						visibleNodes[m.inputCursor].expanded = true
					}
				}
			case composeTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable && !visibleNodes[m.outputCursor].expanded {
						visibleNodes[m.outputCursor].expanded = true
					}
				}
			}
		case "left", "h":
			switch m.activeTab {
			case composeTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable && visibleNodes[m.inputCursor].expanded {
						visibleNodes[m.inputCursor].expanded = false
					}
				}
			case composeTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable && visibleNodes[m.outputCursor].expanded {
						visibleNodes[m.outputCursor].expanded = false
					}
				}
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
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := titleStyle.Render(m.node.Name)
	if m.node.Name == "" {
		title = titleStyle.Render(m.node.Key)
	}

	typeTag := dimStyle.Render(fmt.Sprintf("[%s]", m.node.Type))

	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top,
		lipgloss.NewStyle().Padding(0, 1).Render(fmt.Sprintf("%s  %s", title, typeTag)))
}

func (m composeDetailModel) renderComposeTabsWithBody() string {
	inactiveTabBorder := tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder := tabBorderWithBottom("┘", " ", "└")

	inactiveTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	activeTabStyle := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("230"))

	var renderedTabs []string
	for i, name := range composeTabNames {
		style := inactiveTabStyle
		isActive := i == m.activeTab

		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}

		if i == 0 {
			border := style.GetBorderStyle()
			if isActive {
				border.BottomLeft = "│"
			} else {
				border.BottomLeft = "├"
			}
			style = style.Border(border, true)
		}
		renderedTabs = append(renderedTabs, style.Render(name))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	rowWidth := lipgloss.Width(row)
	gapWidth := m.width - rowWidth - 2
	if gapWidth > 0 {
		gap := lipgloss.NewStyle().Border(lipgloss.Border{
			Bottom:      "─",
			BottomLeft:  "┴",
			BottomRight: "┐",
		}, false, false, true).BorderForeground(lipgloss.Color("240")).Width(gapWidth).Render("")
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	}

	content := m.getComposeTabContent()

	window := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "│", Right: "│", Bottom: "─", BottomLeft: "└", BottomRight: "┘"}, false, true, true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width - 2).
		Height(m.composeBodyHeight()).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, row, window)
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
	return m.renderComposeParamTreeTab("Input Parameters", m.inputTree, m.inputCursor, m.inputScroll)
}

func (m composeDetailModel) renderComposeOutputVarsTab() string {
	return m.renderComposeParamTreeTab("Output Parameters", m.outputTree, m.outputCursor, m.outputScroll)
}

func (m composeDetailModel) renderComposeParamTreeTab(title string, tree []outputNode, cursor, scroll int) string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching %s...", m.spinner.View(), strings.ToLower(title))
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}
	if len(tree) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render(fmt.Sprintf("No %s available for this resource.", strings.ToLower(title))))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	fmt.Fprintf(&b, "  %s\n\n", headerStyle.Render(title))

	visibleNodes := flattenOutputTree(tree)

	visibleRows := m.composeVisibleRows()
	totalEntries := len(visibleNodes)
	cursorIndex, scrollOffset := normalizeViewport(cursor, scroll, totalEntries, visibleRows)

	end := scrollOffset + visibleRows
	if end > totalEntries {
		end = totalEntries
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	braceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedBg := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	maxLineWidth := m.composeContentWidth() - 4
	if maxLineWidth < 1 {
		maxLineWidth = 1
	}
	rowStyle := lipgloss.NewStyle().MaxWidth(maxLineWidth)
	selectedRowStyle := selectedBg.MaxWidth(maxLineWidth)

	for idx := scrollOffset; idx < end; idx++ {
		node := visibleNodes[idx]
		indent := strings.Repeat("  ", node.depth)

		cursorMarker := "  "
		if idx == cursorIndex {
			cursorMarker = "▶ "
		}

		var line string
		if node.expandable {
			arrow := "▸"
			if node.expanded {
				arrow = "▾"
			}
			childCount := len(node.children)
			typeMark := braceStyle.Render("{}")
			if node.nodeType == "array" {
				typeMark = braceStyle.Render("[]")
			}
			countStr := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(fmt.Sprintf("  %d items", childCount))
			line = fmt.Sprintf("%s%s %s %s%s", indent, arrow, keyStyle.Render(node.key), typeMark, countStr)
		} else {
			var styledVal string
			switch node.nodeType {
			case "string":
				styledVal = strStyle.Render(fmt.Sprintf("%q", node.value))
			case "number":
				styledVal = numStyle.Render(node.value)
			case "bool":
				styledVal = boolStyle.Render(node.value)
			case "null":
				styledVal = nullStyle.Render("null")
			default:
				styledVal = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(node.value)
			}
			line = fmt.Sprintf("%s  %s: %s", indent, keyStyle.Render(node.key), styledVal)
		}

		if idx == cursorIndex {
			line = selectedRowStyle.Render(line)
		} else {
			line = rowStyle.Render(line)
		}

		fmt.Fprintf(&b, "  %s%s\n", cursorMarker, line)
	}

	if totalEntries > visibleRows {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pos := ""
		if scrollOffset == 0 {
			pos = "top"
		} else if end >= totalEntries {
			pos = "end"
		} else {
			pct := (scrollOffset * 100) / (totalEntries - visibleRows)
			pos = fmt.Sprintf("%d%%", pct)
		}
		fmt.Fprintf(&b, "\n  %s\n", dimStyle.Render(fmt.Sprintf("↑↓: navigate  ←→/enter: expand/collapse  [%d/%d %s]", cursorIndex+1, totalEntries, pos)))
	}

	return b.String()
}

func (m composeDetailModel) renderComposeWfErrorsTab() string {
	loading := m.debugData.PlanDAG != nil && m.debugData.PlanDAG.ProgressLoading
	steps := m.getWfEvents()
	enrichBootstrapSteps(steps, m.node.Key, m.debugData.PlanDAG)
	isLive := isWorkflowInProgress(steps)
	return renderWorkflowEventsTab(steps, m.wfErrors, m.composeBodyHeight(), m.composeContentWidth(), loading, m.spinner.View(), isLive)
}

func (m composeDetailModel) renderComposeFooter() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	var text string
	switch m.activeTab {
	case composeTabInputVars, composeTabOutputVars:
		if (m.activeTab == composeTabInputVars && len(m.inputTree) > 0) ||
			(m.activeTab == composeTabOutputVars && len(m.outputTree) > 0) {
			text = "↑↓: navigate  ←→/enter: expand/collapse  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
		} else {
			text = "tab/shift+tab: switch tabs  esc: back  q: quit"
		}
	case composeTabWfErrors:
		text = "↑↓/pgup/pgdn: scroll  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	default:
		text = "tab/shift+tab: switch tabs  esc: back  q: quit"
	}
	if m.clipboardMsg != "" {
		clipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		text = clipStyle.Render(m.clipboardMsg) + "  " + text
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
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
	h := m.height - 8
	if h < 1 {
		h = 1
	}
	return h
}

func (m composeDetailModel) composeVisibleRows() int {
	rows := m.composeBodyHeight() - 4
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m composeDetailModel) composeContentWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m composeDetailModel) getWfEvents() *ResourceWorkflowSteps {
	if m.debugData.PlanDAG != nil && m.debugData.PlanDAG.WorkflowStepsByKey != nil {
		return m.debugData.PlanDAG.WorkflowStepsByKey[m.node.Key]
	}
	return nil
}
