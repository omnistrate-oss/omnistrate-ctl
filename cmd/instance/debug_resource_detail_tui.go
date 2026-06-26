package instance

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
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
	row := renderResourceDetailTabRow(width, tabNames, activeTab, true)

	window := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "│", Right: "│", Bottom: "─", BottomLeft: "└", BottomRight: "┘"}, false, true, true).
		BorderForeground(lipgloss.Color("240")).
		Width(width - 2).
		Height(bodyHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, row, window)
}

func renderResourceDetailTabRow(width int, tabNames []string, activeTab int, framed bool) string {
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
	gapWidth := width - rowWidth
	if framed {
		gapWidth -= 2
	}
	if gapWidth > 0 {
		bottomRight := "┐"
		if !framed {
			bottomRight = "─"
		}
		gap := lipgloss.NewStyle().Border(lipgloss.Border{
			Bottom:      "─",
			BottomLeft:  "┴",
			BottomRight: bottomRight,
		}, false, false, true).BorderForeground(lipgloss.Color("240")).Width(gapWidth).Render("")
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	}

	return row
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

func renderResourceMetricsTab(debugData DebugData, bodyHeight, contentWidth, scroll int) string {
	content := renderDashboardDetailContent(resourceMetricsCopyText(debugData), contentWidth)
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}

	visibleRows := bodyHeight - 1
	if visibleRows < 1 {
		visibleRows = 1
	}

	maxScroll := len(lines) - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + visibleRows
	if end > len(lines) {
		end = len(lines)
	}

	visible := append([]string(nil), lines[scroll:end]...)
	for len(visible) < visibleRows {
		visible = append(visible, "")
	}

	if maxScroll > 0 {
		pos := "top"
		if scroll == maxScroll {
			pos = "end"
		} else if scroll > 0 {
			pos = fmt.Sprintf("%d%%", (scroll*100)/maxScroll)
		}
		visible = append(visible, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
			fmt.Sprintf("[%d/%d %s]", end, len(lines), pos),
		))
	}

	for index, line := range visible {
		visible[index] = padRightANSI(line, contentWidth)
	}

	return strings.Join(visible, "\n")
}

func resourceMetricsCopyText(debugData DebugData) string {
	catalog := debugData.DashboardCatalog
	if catalog == nil || len(catalog.Features) == 0 {
		return "Metrics Details\n\nNo Grafana dashboard metadata is available for this instance."
	}

	var b strings.Builder
	b.WriteString("Metrics Details")
	if catalog.InstanceID != "" {
		fmt.Fprintf(&b, "\nInstance: %s", catalog.InstanceID)
	}
	if catalog.PreferredFeatureKey != "" {
		fmt.Fprintf(&b, "\nPreferred view: %s", catalog.PreferredFeatureKey)
	}

	if len(catalog.Features) > 0 {
		b.WriteString("\n\nAvailable views:")
		for _, feature := range catalog.Features {
			fmt.Fprintf(&b, "\n- %s", dashboardFeatureDisplayName(feature))
		}
	}

	accessGroups := buildMetricsAccessGroups(catalog.Features)
	if len(accessGroups) > 0 {
		b.WriteString("\n\nGrafana Access")
		for _, group := range accessGroups {
			if len(accessGroups) > 1 {
				fmt.Fprintf(&b, "\n\nViews: %s", strings.Join(group.Views, ", "))
			}
			if group.GrafanaEndpoint != "" {
				fmt.Fprintf(&b, "\nGrafana Endpoint: %s", group.GrafanaEndpoint)
			}
			if group.GrafanaUIUsername != "" || group.GrafanaUIPassword != "" {
				b.WriteString("\nGrafana UI")
				fmt.Fprintf(&b, "\nUsername: %s", dashboardDisplayValue(group.GrafanaUIUsername))
				fmt.Fprintf(&b, "\nPassword: %s", dashboardDisplayValue(group.GrafanaUIPassword))
				if group.GrafanaUILoginScope != "" {
					fmt.Fprintf(&b, "\nScope: %s", dashboardDisplayValue(group.GrafanaUILoginScope))
				}
			}
			if group.ServiceAccountName != "" || group.ServiceAccountToken != "" {
				b.WriteString("\nGrafana API")
				fmt.Fprintf(&b, "\nService account: %s", dashboardDisplayValue(group.ServiceAccountName))
				fmt.Fprintf(&b, "\nService account token: %s", dashboardDisplayValue(group.ServiceAccountToken))
			}
		}
	}

	dashboards := uniqueMetricDashboards(catalog.Features)
	if len(dashboards) > 0 {
		b.WriteString("\n\nPublished dashboards:")
		for _, dashboard := range dashboards {
			if dashboard.Description != "" {
				fmt.Fprintf(&b, "\n- %s: %s", dashboard.Name, dashboard.Description)
			} else {
				fmt.Fprintf(&b, "\n- %s", dashboard.Name)
			}
			if dashboard.URL != "" {
				fmt.Fprintf(&b, "\n  %s", dashboard.URL)
			}
		}
	}

	definitions := uniqueMetricDashboardDefinitions(catalog.Features)
	if len(definitions) > 0 {
		b.WriteString("\n\nDashboard templates:")
		for _, dashboard := range definitions {
			if dashboard.Title != "" {
				fmt.Fprintf(&b, "\n- %s/%s: %s", dashboard.Source, dashboard.Name, dashboard.Title)
				continue
			}
			fmt.Fprintf(&b, "\n- %s/%s", dashboard.Source, dashboard.Name)
		}
	}

	return b.String()
}

type metricAccessGroup struct {
	Views               []string
	GrafanaEndpoint     string
	GrafanaUIUsername   string
	GrafanaUIPassword   string
	GrafanaUILoginScope string
	ServiceAccountName  string
	ServiceAccountToken string
}

func buildMetricsAccessGroups(features []dataaccess.DashboardFeatureInfo) []metricAccessGroup {
	groups := make([]metricAccessGroup, 0)
	indexByKey := make(map[string]int)
	for _, feature := range features {
		key := strings.Join([]string{
			feature.GrafanaEndpoint,
			feature.GrafanaUIUsername,
			feature.GrafanaUIPassword,
			feature.GrafanaUILoginScope,
			feature.ServiceAccountName,
			feature.ServiceAccountToken,
		}, "\x00")

		index, ok := indexByKey[key]
		if !ok {
			index = len(groups)
			indexByKey[key] = index
			groups = append(groups, metricAccessGroup{
				GrafanaEndpoint:     feature.GrafanaEndpoint,
				GrafanaUIUsername:   feature.GrafanaUIUsername,
				GrafanaUIPassword:   feature.GrafanaUIPassword,
				GrafanaUILoginScope: feature.GrafanaUILoginScope,
				ServiceAccountName:  feature.ServiceAccountName,
				ServiceAccountToken: feature.ServiceAccountToken,
			})
		}
		groups[index].Views = append(groups[index].Views, dashboardFeatureDisplayName(feature))
	}

	return groups
}

func uniqueMetricDashboards(features []dataaccess.DashboardFeatureInfo) []dataaccess.DashboardRef {
	seen := make(map[string]bool)
	var dashboards []dataaccess.DashboardRef

	for _, feature := range features {
		for _, dashboard := range feature.Dashboards {
			key := strings.Join([]string{dashboard.Name, dashboard.Description, dashboard.URL}, "\x00")
			if seen[key] {
				continue
			}
			seen[key] = true
			dashboards = append(dashboards, dashboard)
		}
	}

	return dashboards
}

func uniqueMetricDashboardDefinitions(features []dataaccess.DashboardFeatureInfo) []dataaccess.DashboardDefinition {
	seen := make(map[string]bool)
	var definitions []dataaccess.DashboardDefinition

	for _, feature := range features {
		for _, dashboard := range feature.DashboardDefinitions {
			key := strings.Join([]string{dashboard.Source, dashboard.Name, dashboard.Title}, "\x00")
			if seen[key] {
				continue
			}
			seen[key] = true
			definitions = append(definitions, dashboard)
		}
	}

	return definitions
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
