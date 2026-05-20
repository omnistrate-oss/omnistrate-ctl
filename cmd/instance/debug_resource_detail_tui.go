package instance

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newResourceDetailSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return s
}

func resourceDetailBodyHeight(height int) int {
	h := height - 8
	if h < 1 {
		h = 1
	}
	return h
}

func resourceDetailVisibleRows(height int) int {
	rows := resourceDetailBodyHeight(height) - 4
	if rows < 1 {
		rows = 1
	}
	return rows
}

func resourceDetailContentWidth(width int) int {
	w := width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func renderResourceDetailHeader(width int, node PlanDAGNode) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := titleStyle.Render(node.Name)
	if node.Name == "" {
		title = titleStyle.Render(node.Key)
	}

	typeTag := dimStyle.Render(fmt.Sprintf("[%s]", node.Type))

	return lipgloss.Place(width, 1, lipgloss.Left, lipgloss.Top,
		lipgloss.NewStyle().Padding(0, 1).Render(fmt.Sprintf("%s  %s", title, typeTag)))
}

func renderResourceDetailTabsWithBody(width, bodyHeight int, tabNames []string, activeTab int, content string) string {
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
	for i, name := range tabNames {
		style := inactiveTabStyle
		isActive := i == activeTab
		if isActive {
			style = activeTabStyle
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
	gapWidth := width - rowWidth - 2
	if gapWidth > 0 {
		gap := lipgloss.NewStyle().Border(lipgloss.Border{
			Bottom:      "─",
			BottomLeft:  "┴",
			BottomRight: "┐",
		}, false, false, true).BorderForeground(lipgloss.Color("240")).Width(gapWidth).Render("")
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	}

	window := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "│", Right: "│", Bottom: "─", BottomLeft: "└", BottomRight: "┘"}, false, true, true).
		BorderForeground(lipgloss.Color("240")).
		Width(width - 2).
		Height(bodyHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, row, window)
}

func renderResourceDetailParamTreeTab(title string, tree []outputNode, cursor, scroll, visibleRows, contentWidth int, loading bool, spinnerView string, loadErr, fetchErr error, includeExpandHelp bool) string {
	if loading {
		return fmt.Sprintf("\n  %s Fetching %s...", spinnerView, strings.ToLower(title))
	}
	if fetchErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error fetching %s: %v", strings.ToLower(title), fetchErr)))
	}
	if loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", loadErr)))
	}
	if len(tree) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render(fmt.Sprintf("No %s available for this resource.", strings.ToLower(title))))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	fmt.Fprintf(&b, "  %s\n\n", headerStyle.Render(title))

	visibleNodes := flattenOutputTree(tree)
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
	maxLineWidth := contentWidth - 4
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
		hint := "↑↓: navigate"
		if includeExpandHelp {
			hint += "  ←→/enter: expand/collapse"
		}
		fmt.Fprintf(&b, "\n  %s\n", dimStyle.Render(fmt.Sprintf("%s  [%d/%d %s]", hint, cursorIndex+1, totalEntries, pos)))
	}

	return b.String()
}

func renderResourceWorkflowEventsTab(debugData DebugData, node PlanDAGNode, wfErrors *workflowErrorsState, bodyHeight, contentWidth int, spinnerView string) string {
	loading := debugData.PlanDAG != nil && debugData.PlanDAG.ProgressLoading
	steps := getResourceWorkflowEvents(debugData, node)
	enrichBootstrapSteps(steps, node.Key, debugData.PlanDAG)
	isLive := isWorkflowInProgress(steps)
	return renderWorkflowEventsTab(steps, wfErrors, bodyHeight, contentWidth, loading, spinnerView, isLive)
}

func renderResourceDetailFooter(width int, clipboardMsg, text string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	if clipboardMsg != "" {
		clipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		text = clipStyle.Render(clipboardMsg) + "  " + text
	}
	return lipgloss.Place(width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func getResourceWorkflowEvents(debugData DebugData, node PlanDAGNode) *ResourceWorkflowSteps {
	if debugData.PlanDAG != nil && debugData.PlanDAG.WorkflowStepsByKey != nil {
		return debugData.PlanDAG.WorkflowStepsByKey[node.Key]
	}
	return nil
}

func handleResourceWorkflowRefreshTick(debugData DebugData, node PlanDAGNode, wfErrors *workflowErrorsState) tea.Cmd {
	steps := getResourceWorkflowEvents(debugData, node)
	if isWorkflowInProgress(steps) && !wfErrors.refreshing {
		wfErrors.refreshing = true
		return fetchWfEventsForResource(debugData, node.Key)
	}
	return nil
}

func handleResourceWorkflowRefresh(debugData DebugData, node PlanDAGNode, wfErrors *workflowErrorsState, msg wfEventsRefreshMsg) tea.Cmd {
	wfErrors.refreshing = false
	wfErrors.lastRefresh = time.Now()
	if msg.err == nil && msg.steps != nil && debugData.PlanDAG != nil {
		if debugData.PlanDAG.WorkflowStepsByKey == nil {
			debugData.PlanDAG.WorkflowStepsByKey = make(map[string]*ResourceWorkflowSteps)
		}
		debugData.PlanDAG.WorkflowStepsByKey[node.Key] = msg.steps
	}
	if isWorkflowInProgress(getResourceWorkflowEvents(debugData, node)) {
		return tea.Batch(scheduleWfEventsRefresh(), scheduleWfCountdownTick())
	}
	return nil
}

func handleResourceWorkflowCountdown(debugData DebugData, node PlanDAGNode) tea.Cmd {
	if isWorkflowInProgress(getResourceWorkflowEvents(debugData, node)) {
		return scheduleWfCountdownTick()
	}
	return nil
}

func scheduleResourceWorkflowRefreshIfNeeded(debugData DebugData, node PlanDAGNode) tea.Cmd {
	if isWorkflowInProgress(getResourceWorkflowEvents(debugData, node)) {
		return tea.Batch(scheduleWfEventsRefresh(), scheduleWfCountdownTick())
	}
	return nil
}

func moveResourceDetailTreeUp(tree []outputNode, cursor, scroll, visibleRows int) (int, int) {
	if len(tree) == 0 {
		return cursor, scroll
	}
	visibleNodes := flattenOutputTree(tree)
	if cursor > 0 {
		cursor--
	}
	return normalizeViewport(cursor, scroll, len(visibleNodes), visibleRows)
}

func moveResourceDetailTreeDown(tree []outputNode, cursor, scroll, visibleRows int) (int, int) {
	if len(tree) == 0 {
		return cursor, scroll
	}
	visibleNodes := flattenOutputTree(tree)
	if cursor < len(visibleNodes)-1 {
		cursor++
	}
	return normalizeViewport(cursor, scroll, len(visibleNodes), visibleRows)
}

func toggleResourceDetailTreeNode(tree []outputNode, cursor int) {
	visibleNodes := flattenOutputTree(tree)
	if cursor >= 0 && cursor < len(visibleNodes) && visibleNodes[cursor].expandable {
		visibleNodes[cursor].expanded = !visibleNodes[cursor].expanded
	}
}

func expandResourceDetailTreeNode(tree []outputNode, cursor int) {
	visibleNodes := flattenOutputTree(tree)
	if cursor >= 0 && cursor < len(visibleNodes) && visibleNodes[cursor].expandable && !visibleNodes[cursor].expanded {
		visibleNodes[cursor].expanded = true
	}
}

func collapseResourceDetailTreeNode(tree []outputNode, cursor int) {
	visibleNodes := flattenOutputTree(tree)
	if cursor >= 0 && cursor < len(visibleNodes) && visibleNodes[cursor].expandable && visibleNodes[cursor].expanded {
		visibleNodes[cursor].expanded = false
	}
}
