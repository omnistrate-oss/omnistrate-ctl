package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

const (
	opTabInputVars     = 0
	opTabOutputVars    = 1
	opTabCRDOutputVars = 2
	opTabWfErrors      = 3
	opNumTabs          = 4
)

var opTabNames = []string{"Input Parameters", "Output Parameters", "Operator CRD Outputs", "Workflow Events"}

func init() {
	if len(opTabNames) != opNumTabs {
		panic(fmt.Sprintf("opTabNames length %d does not match opNumTabs %d", len(opTabNames), opNumTabs))
	}
}

// operatorDataMsg is sent when operator debug data has been fetched.
type operatorDataMsg struct {
	operatorData *OperatorData
	err          error
}

type operatorDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int

	loading bool
	loadErr error
	spinner spinner.Model

	operatorData *OperatorData

	// Input variables tree
	inputTree   []outputNode
	inputCursor int
	inputScroll int

	// Output variables tree
	outputTree   []outputNode
	outputCursor int
	outputScroll int

	// CRD output parameters tree
	crdOutputTree   []outputNode
	crdOutputCursor int
	crdOutputScroll int

	// Workflow Events tab
	wfErrors *workflowErrorsState

	clipboardMsg string
}

func newOperatorDetailModel(node PlanDAGNode, data DebugData) operatorDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return operatorDetailModel{
		node:      node,
		debugData: data,
		activeTab: opTabInputVars,
		loading:   true,
		spinner:   s,
		wfErrors:  &workflowErrorsState{},
	}
}

func (m operatorDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchOperatorData())
}

func (m operatorDetailModel) fetchOperatorData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		opData := &OperatorData{}

		// Fetch all input parameters
		inputParams, err := fetchInputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.InputParams,
		)
		if err == nil {
			opData.InputParams = inputParams
		}

		// Fetch exported output parameters
		outputParams, listErr := fetchOutputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.ResultParams,
		)
		if listErr == nil {
			opData.OutputParams = outputParams
		}

		// Fetch CRD output parameters from DescribeResource (operatorCRDConfiguration.outputParameters)
		resourceResult, descErr := dataaccess.DescribeResource(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			&m.debugData.ProductTierID, &m.debugData.TierVersion,
		)
		if descErr == nil && resourceResult != nil {
			crdConfig, ok := resourceResult.GetOperatorCRDConfigurationOk()
			if ok && crdConfig != nil {
				outputParams := crdConfig.GetOutputParameters()
				for k, v := range outputParams {
					opData.CRDOutputParams = append(opData.CRDOutputParams, OperatorCRDOutputParam{
						Key:   k,
						Value: v,
					})
				}
			}
		}

		if err != nil && listErr != nil && descErr != nil {
			return operatorDataMsg{err: fmt.Errorf("failed to fetch operator data: %w", err)}
		}

		return operatorDataMsg{operatorData: opData}
	}
}

func (m operatorDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case operatorDataMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err
			return m, nil
		}
		m.operatorData = msg.operatorData
		if m.operatorData != nil {
			m.inputTree = buildOperatorParamTree(m.operatorData.InputParams)
			m.outputTree = buildOperatorOutputParamTree(m.operatorData.OutputParams)
			m.crdOutputTree = buildOperatorCRDOutputParamTree(m.operatorData.CRDOutputParams)
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
			m.activeTab = (m.activeTab + 1) % opNumTabs
			return m, nil
		case "shift+tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab - 1 + opNumTabs) % opNumTabs
			return m, nil
		case "up", "k":
			if m.wfErrors.modalText != "" {
				if m.wfErrors.modalScroll > 0 {
					m.wfErrors.modalScroll--
				}
				return m, nil
			}
			switch m.activeTab {
			case opTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor > 0 {
						m.inputCursor--
					}
					m.inputCursor, m.inputScroll = normalizeViewport(
						m.inputCursor, m.inputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor > 0 {
						m.outputCursor--
					}
					m.outputCursor, m.outputScroll = normalizeViewport(
						m.outputCursor, m.outputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabCRDOutputVars:
				if len(m.crdOutputTree) > 0 {
					visibleNodes := flattenOutputTree(m.crdOutputTree)
					if m.crdOutputCursor > 0 {
						m.crdOutputCursor--
					}
					m.crdOutputCursor, m.crdOutputScroll = normalizeViewport(
						m.crdOutputCursor, m.crdOutputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabWfErrors:
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
			case opTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes)-1 {
						m.inputCursor++
					}
					m.inputCursor, m.inputScroll = normalizeViewport(
						m.inputCursor, m.inputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes)-1 {
						m.outputCursor++
					}
					m.outputCursor, m.outputScroll = normalizeViewport(
						m.outputCursor, m.outputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabCRDOutputVars:
				if len(m.crdOutputTree) > 0 {
					visibleNodes := flattenOutputTree(m.crdOutputTree)
					if m.crdOutputCursor < len(visibleNodes)-1 {
						m.crdOutputCursor++
					}
					m.crdOutputCursor, m.crdOutputScroll = normalizeViewport(
						m.crdOutputCursor, m.crdOutputScroll, len(visibleNodes), m.opVisibleRows(),
					)
				}
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items)-1 {
					m.wfErrors.cursor++
				}
				_ = items
			}
		case "enter":
			switch m.activeTab {
			case opTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable {
						visibleNodes[m.inputCursor].expanded = !visibleNodes[m.inputCursor].expanded
					}
				}
			case opTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable {
						visibleNodes[m.outputCursor].expanded = !visibleNodes[m.outputCursor].expanded
					}
				}
			case opTabCRDOutputVars:
				if len(m.crdOutputTree) > 0 {
					visibleNodes := flattenOutputTree(m.crdOutputTree)
					if m.crdOutputCursor < len(visibleNodes) && visibleNodes[m.crdOutputCursor].expandable {
						visibleNodes[m.crdOutputCursor].expanded = !visibleNodes[m.crdOutputCursor].expanded
					}
				}
			case opTabWfErrors:
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
			case opTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable && !visibleNodes[m.inputCursor].expanded {
						visibleNodes[m.inputCursor].expanded = true
					}
				}
			case opTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable && !visibleNodes[m.outputCursor].expanded {
						visibleNodes[m.outputCursor].expanded = true
					}
				}
			case opTabCRDOutputVars:
				if len(m.crdOutputTree) > 0 {
					visibleNodes := flattenOutputTree(m.crdOutputTree)
					if m.crdOutputCursor < len(visibleNodes) && visibleNodes[m.crdOutputCursor].expandable && !visibleNodes[m.crdOutputCursor].expanded {
						visibleNodes[m.crdOutputCursor].expanded = true
					}
				}
			}
		case "left", "h":
			switch m.activeTab {
			case opTabInputVars:
				if len(m.inputTree) > 0 {
					visibleNodes := flattenOutputTree(m.inputTree)
					if m.inputCursor < len(visibleNodes) && visibleNodes[m.inputCursor].expandable && visibleNodes[m.inputCursor].expanded {
						visibleNodes[m.inputCursor].expanded = false
					}
				}
			case opTabOutputVars:
				if len(m.outputTree) > 0 {
					visibleNodes := flattenOutputTree(m.outputTree)
					if m.outputCursor < len(visibleNodes) && visibleNodes[m.outputCursor].expandable && visibleNodes[m.outputCursor].expanded {
						visibleNodes[m.outputCursor].expanded = false
					}
				}
			case opTabCRDOutputVars:
				if len(m.crdOutputTree) > 0 {
					visibleNodes := flattenOutputTree(m.crdOutputTree)
					if m.crdOutputCursor < len(visibleNodes) && visibleNodes[m.crdOutputCursor].expandable && visibleNodes[m.crdOutputCursor].expanded {
						visibleNodes[m.crdOutputCursor].expanded = false
					}
				}
			}
		case "pgup":
			switch m.activeTab {
			case opTabWfErrors:
				m.wfErrors.cursor -= m.opVisibleRows()
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "pgdown":
			switch m.activeTab {
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				m.wfErrors.cursor += m.opVisibleRows()
				if m.wfErrors.cursor >= len(items) {
					m.wfErrors.cursor = len(items) - 1
				}
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "y":
			content := m.opCopyableContent()
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

func (m operatorDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.wfErrors.modalText != "" {
		return renderWfEventModal(m.wfErrors, m.width, m.height)
	}

	header := m.renderOpHeader()
	tabs := m.renderOpTabsWithBody()
	footer := m.renderOpFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, footer)
}

func (m operatorDetailModel) renderOpHeader() string {
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

func (m operatorDetailModel) renderOpTabsWithBody() string {
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
	for i, name := range opTabNames {
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

	content := m.getOpTabContent()

	window := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "│", Right: "│", Bottom: "─", BottomLeft: "└", BottomRight: "┘"}, false, true, true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width - 2).
		Height(m.opBodyHeight()).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, row, window)
}

func (m operatorDetailModel) getOpTabContent() string {
	switch m.activeTab {
	case opTabInputVars:
		return m.renderInputVarsTab()
	case opTabOutputVars:
		return m.renderOutputVarsTab()
	case opTabCRDOutputVars:
		return m.renderCRDOutputVarsTab()
	case opTabWfErrors:
		return m.renderOpWfErrorsTab()
	}
	return ""
}

func (m operatorDetailModel) renderInputVarsTab() string {
	return m.renderParamTreeTab("Input Parameters", m.inputTree, m.inputCursor, m.inputScroll)
}

func (m operatorDetailModel) renderOutputVarsTab() string {
	return m.renderParamTreeTab("Output Parameters", m.outputTree, m.outputCursor, m.outputScroll)
}

func (m operatorDetailModel) renderCRDOutputVarsTab() string {
	return m.renderParamTreeTab("Operator CRD Outputs", m.crdOutputTree, m.crdOutputCursor, m.crdOutputScroll)
}

func (m operatorDetailModel) renderParamTreeTab(title string, tree []outputNode, cursor, scroll int) string {
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

	visibleRows := m.opVisibleRows()
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
	maxLineWidth := m.opContentWidth() - 4
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

func (m operatorDetailModel) renderOpWfErrorsTab() string {
	loading := m.debugData.PlanDAG != nil && m.debugData.PlanDAG.ProgressLoading
	steps := m.getWfEvents()
	enrichBootstrapSteps(steps, m.node.Key, m.debugData.PlanDAG)
	isLive := isWorkflowInProgress(steps)
	return renderWorkflowEventsTab(steps, m.wfErrors, m.opBodyHeight(), m.opContentWidth(), loading, m.spinner.View(), isLive)
}

func (m operatorDetailModel) renderOpFooter() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	var text string
	switch m.activeTab {
	case opTabInputVars, opTabOutputVars, opTabCRDOutputVars:
		if (m.activeTab == opTabInputVars && len(m.inputTree) > 0) ||
			(m.activeTab == opTabOutputVars && len(m.outputTree) > 0) ||
			(m.activeTab == opTabCRDOutputVars && len(m.crdOutputTree) > 0) {
			text = "↑↓: navigate  ←→/enter: expand/collapse  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
		} else {
			text = "tab/shift+tab: switch tabs  esc: back  q: quit"
		}
	case opTabWfErrors:
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

func (m operatorDetailModel) opCopyableContent() string {
	switch m.activeTab {
	case opTabInputVars:
		if m.operatorData != nil && len(m.operatorData.InputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.InputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabOutputVars:
		if m.operatorData != nil && len(m.operatorData.OutputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.OutputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabCRDOutputVars:
		if m.operatorData != nil && len(m.operatorData.CRDOutputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.CRDOutputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabWfErrors:
		return workflowEventsCopyText(m.getWfEvents())
	}
	return ""
}

func (m operatorDetailModel) opBodyHeight() int {
	h := m.height - 8
	if h < 1 {
		h = 1
	}
	return h
}

func (m operatorDetailModel) opVisibleRows() int {
	rows := m.opBodyHeight() - 4
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m operatorDetailModel) opContentWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m operatorDetailModel) getWfEvents() *ResourceWorkflowSteps {
	if m.debugData.PlanDAG != nil && m.debugData.PlanDAG.WorkflowStepsByKey != nil {
		return m.debugData.PlanDAG.WorkflowStepsByKey[m.node.Key]
	}
	return nil
}

// buildOperatorParamTree converts input parameters to a navigable tree of outputNodes.
func buildOperatorParamTree(params []OperatorInputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		details := map[string]interface{}{
			"type":        p.Type,
			"description": p.Description,
			"required":    p.Required,
			"modifiable":  p.Modifiable,
		}
		if p.ResolvedValue != "" {
			details["value"] = p.ResolvedValue
		} else if p.DefaultValue != "" {
			details["defaultValue"] = p.DefaultValue
		}

		displayName := p.Key
		if p.DisplayName != "" && p.DisplayName != p.Key {
			displayName = fmt.Sprintf("%s (%s)", p.Key, p.DisplayName)
		}

		node := buildJSONNode(displayName, details, 0)
		roots = append(roots, *node)
	}
	return roots
}

// buildOperatorOutputParamTree converts output parameters to a navigable tree of outputNodes.
func buildOperatorOutputParamTree(params []OperatorOutputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		details := map[string]interface{}{
			"description": p.Description,
		}
		if p.Type != "" {
			details["type"] = p.Type
		}
		if p.ResolvedValue != "" {
			details["value"] = p.ResolvedValue
		} else if p.Value != "" {
			details["value"] = p.Value
		}
		if p.ValueRef != "" {
			details["valueRef"] = p.ValueRef
		}

		displayName := p.Key
		if p.DisplayName != "" && p.DisplayName != p.Key {
			displayName = fmt.Sprintf("%s (%s)", p.Key, p.DisplayName)
		}

		node := buildJSONNode(displayName, details, 0)
		roots = append(roots, *node)
	}
	return roots
}

// buildOperatorCRDOutputParamTree converts CRD output parameters to a navigable tree of outputNodes.
func buildOperatorCRDOutputParamTree(params []OperatorCRDOutputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		details := map[string]interface{}{
			"value": p.Value,
		}

		node := buildJSONNode(p.Key, details, 0)
		roots = append(roots, *node)
	}
	return roots
}
