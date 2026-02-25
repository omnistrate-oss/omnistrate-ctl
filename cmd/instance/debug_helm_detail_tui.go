package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

const (
	helmTabLogs   = 0
	helmTabValues = 1
	helmNumTabs   = 2
)

var helmTabNames = []string{"Helm Logs", "Chart Values"}

// helmDataMsg is sent when helm debug data has been fetched
type helmDataMsg struct {
	helmData *HelmData
	err      error
}

type helmDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int

	// Loading state
	loading bool
	spinner spinner.Model
	loadErr error

	// Helm data
	helmData *HelmData

	// Logs tab
	logLines  []string
	logScroll int

	// Values tab (tree explorer)
	valuesTree   []outputNode
	valuesCursor int
}

func newHelmDetailModel(node PlanDAGNode, data DebugData) helmDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return helmDetailModel{
		node:      node,
		debugData: data,
		activeTab: helmTabLogs,
		loading:   true,
		spinner:   s,
	}
}

func (m helmDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchHelmData())
}

func (m helmDetailModel) fetchHelmData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		debugResult, err := dataaccess.DebugResourceInstance(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.debugData.EnvironmentID, m.debugData.InstanceID,
		)
		if err != nil {
			return helmDataMsg{err: fmt.Errorf("failed to get debug info: %w", err)}
		}

		if debugResult.ResourcesDebug == nil {
			return helmDataMsg{err: fmt.Errorf("no debug data available")}
		}

		// Find the helm resource by matching node key
		for resourceKey, resourceDebugInfo := range *debugResult.ResourcesDebug {
			if resourceKey != m.node.Key {
				continue
			}

			debugDataInterface, ok := resourceDebugInfo.GetDebugDataOk()
			if !ok || debugDataInterface == nil {
				continue
			}

			actualDebugData, ok := (*debugDataInterface).(map[string]interface{})
			if !ok {
				continue
			}

			helmData := parseHelmData(actualDebugData)
			return helmDataMsg{helmData: helmData}
		}

		return helmDataMsg{err: fmt.Errorf("no helm data found for resource %s", m.node.Key)}
	}
}

func (m helmDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case helmDataMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err
			return m, nil
		}
		m.helmData = msg.helmData
		if m.helmData != nil {
			// Parse log lines
			if m.helmData.InstallLog != "" {
				m.logLines = strings.Split(m.helmData.InstallLog, "\n")
				// Trim trailing empty lines
				for len(m.logLines) > 0 && strings.TrimSpace(m.logLines[len(m.logLines)-1]) == "" {
					m.logLines = m.logLines[:len(m.logLines)-1]
				}
			}
			// Build values tree
			if len(m.helmData.ChartValues) > 0 {
				raw, err := json.Marshal(m.helmData.ChartValues)
				if err == nil {
					m.valuesTree = buildHelmValuesTree(m.helmData.ChartValues, string(raw))
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			// Signal to parent to close detail view
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab":
			m.activeTab = (m.activeTab + 1) % helmNumTabs
			return m, nil
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + helmNumTabs) % helmNumTabs
			return m, nil
		case "up", "k":
			if m.activeTab == helmTabLogs {
				if m.logScroll > 0 {
					m.logScroll--
				}
			} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor > 0 {
					m.valuesCursor--
				}
				_ = visibleNodes
			}
		case "down", "j":
			if m.activeTab == helmTabLogs {
				m.logScroll++
				if m.logScroll > m.helmLogMaxScroll() {
					m.logScroll = m.helmLogMaxScroll()
				}
			} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor < len(visibleNodes)-1 {
					m.valuesCursor++
				}
			}
		case "pgup":
			if m.activeTab == helmTabLogs {
				m.logScroll -= m.helmBodyHeight()
				if m.logScroll < 0 {
					m.logScroll = 0
				}
			}
		case "pgdown":
			if m.activeTab == helmTabLogs {
				m.logScroll += m.helmBodyHeight()
				if m.logScroll > m.helmLogMaxScroll() {
					m.logScroll = m.helmLogMaxScroll()
				}
			}
		case "enter", "right", "l":
			if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor >= 0 && m.valuesCursor < len(visibleNodes) {
					node := visibleNodes[m.valuesCursor]
					if node.expandable && !node.expanded {
						node.expanded = true
					}
				}
			}
		case "left", "h":
			if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor >= 0 && m.valuesCursor < len(visibleNodes) {
					node := visibleNodes[m.valuesCursor]
					if node.expandable && node.expanded {
						node.expanded = false
					}
				}
			}
		}
	}
	return m, nil
}

func (m helmDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	header := m.renderHelmHeader()
	tabs := m.renderHelmTabsWithBody()
	footer := m.renderHelmFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, footer)
}

func (m helmDetailModel) renderHelmHeader() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := titleStyle.Render(m.node.Name)
	if m.node.Name == "" {
		title = titleStyle.Render(m.node.Key)
	}

	info := ""
	if m.helmData != nil {
		parts := []string{}
		if m.helmData.ReleaseName != "" {
			parts = append(parts, fmt.Sprintf("release: %s", m.helmData.ReleaseName))
		}
		if m.helmData.Namespace != "" {
			parts = append(parts, fmt.Sprintf("ns: %s", m.helmData.Namespace))
		}
		if m.helmData.ChartVersion != "" {
			parts = append(parts, fmt.Sprintf("v%s", m.helmData.ChartVersion))
		}
		if len(parts) > 0 {
			info = dimStyle.Render("  " + strings.Join(parts, "  │  "))
		}
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(
		fmt.Sprintf("%s %s%s", lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("⎈"), title, info),
	)
}

func (m helmDetailModel) renderHelmTabsWithBody() string {
	highlightColor := lipgloss.Color("62")

	inactiveTabBorder := tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder := tabBorderWithBottom("┘", " ", "└")

	inactiveTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(highlightColor).
		Padding(0, 1).
		Foreground(lipgloss.Color("245"))
	activeTabStyle := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(highlightColor).
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("230"))

	var renderedTabs []string
	for i, name := range helmTabNames {
		isFirst := i == 0
		isActive := i == m.activeTab

		var style lipgloss.Style
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}

		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(name))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	rowWidth := lipgloss.Width(row)
	gapWidth := m.width - rowWidth - 2
	if gapWidth > 0 {
		gapBorder := lipgloss.Border{
			Bottom:      "─",
			BottomLeft:  "┴",
			BottomRight: "┐",
		}
		gapStyle := lipgloss.NewStyle().
			Border(gapBorder, false, false, true, false).
			BorderForeground(highlightColor)
		gap := gapStyle.Render(strings.Repeat(" ", gapWidth))
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	}

	content := m.getHelmTabContent()

	bodyH := m.helmBodyHeight()
	windowStyle := lipgloss.NewStyle().
		BorderForeground(highlightColor).
		Border(lipgloss.NormalBorder()).
		UnsetBorderTop().
		Width(m.width - 2).
		Height(bodyH).
		Padding(0, 1)

	window := windowStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, row, window)
}

func (m helmDetailModel) getHelmTabContent() string {
	switch m.activeTab {
	case helmTabLogs:
		return m.renderHelmLogsTab()
	case helmTabValues:
		return m.renderHelmValuesTab()
	}
	return ""
}

func (m helmDetailModel) renderHelmLogsTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching helm logs...", m.spinner.View())
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}
	if len(m.logLines) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No helm installation logs available for this resource."))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	b.WriteString(fmt.Sprintf("  %s\n\n",
		headerStyle.Render(fmt.Sprintf("Helm Install Log (%d lines)", len(m.logLines))),
	))

	bodyH := m.helmBodyHeight() - 4
	if bodyH < 1 {
		bodyH = 1
	}

	totalLines := len(m.logLines)
	scroll := m.logScroll

	maxScroll := totalLines - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + bodyH
	if end > totalLines {
		end = totalLines
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	maxCodeWidth := m.helmContentWidth() - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	for i := scroll; i < end; i++ {
		line := m.logLines[i]
		runes := []rune(line)
		if len(runes) > maxCodeWidth {
			line = string(runes[:maxCodeWidth-1]) + "…"
		}
		lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", i+1))
		styled := highlightHelmLogLine(line)
		b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, styled))
	}

	// Pad remaining lines
	for i := end - scroll; i < bodyH; i++ {
		b.WriteString("\n")
	}

	// Scroll indicator
	if totalLines > bodyH {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pos := ""
		if scroll == 0 {
			pos = "top"
		} else if end >= totalLines {
			pos = "end"
		} else {
			pct := (scroll * 100) / maxScroll
			pos = fmt.Sprintf("%d%%", pct)
		}
		b.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(
			fmt.Sprintf("[%d/%d %s]", scroll+bodyH, totalLines, pos))))
	}

	return b.String()
}

func (m helmDetailModel) renderHelmValuesTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching chart values...", m.spinner.View())
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}
	if len(m.valuesTree) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No chart values available for this resource."))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))

	chartInfo := ""
	if m.helmData != nil && m.helmData.ChartRepoName != "" {
		chartInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
			fmt.Sprintf("  (%s)", m.helmData.ChartRepoName))
	}
	b.WriteString(fmt.Sprintf("  %s%s\n\n", headerStyle.Render("Chart Values"), chartInfo))

	visibleNodes := flattenOutputTree(m.valuesTree)

	visibleRows := m.helmBodyHeight() - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	totalEntries := len(visibleNodes)

	scrollOffset := 0
	if m.valuesCursor >= visibleRows {
		scrollOffset = m.valuesCursor - visibleRows + 1
	}
	if scrollOffset > totalEntries-visibleRows {
		scrollOffset = totalEntries - visibleRows
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

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

	maxValWidth := m.helmContentWidth() - 20
	if maxValWidth < 20 {
		maxValWidth = 20
	}

	for idx := scrollOffset; idx < end; idx++ {
		node := visibleNodes[idx]
		indent := strings.Repeat("  ", node.depth)

		cursor := "  "
		if idx == m.valuesCursor {
			cursor = "▶ "
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
				val := node.value
				runes := []rune(val)
				if len(runes) > maxValWidth {
					val = string(runes[:maxValWidth-1]) + "…"
				}
				styledVal = strStyle.Render(fmt.Sprintf("%q", val))
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

		if idx == m.valuesCursor {
			line = selectedBg.Render(line)
		}

		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, line))
	}

	// Scroll indicator
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
		b.WriteString(fmt.Sprintf("\n  %s\n", dimStyle.Render(fmt.Sprintf("↑↓: navigate  enter: expand/collapse  [%d/%d %s]", m.valuesCursor+1, totalEntries, pos))))
	} else {
		b.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑↓: navigate  enter: expand/collapse")))
	}

	return b.String()
}

func (m helmDetailModel) renderHelmFooter() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	var text string
	if m.activeTab == helmTabLogs {
		text = "↑↓/pgup/pgdn: scroll  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
		text = "↑↓: navigate  ←→/enter: expand/collapse  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else {
		text = "tab/shift+tab: switch tabs  esc: back  q: quit"
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func (m helmDetailModel) helmBodyHeight() int {
	// header(1) + tab row(3) + window bottom border(1) + window padding(2) + footer(1) = 8
	h := m.height - 8
	if h < 1 {
		h = 1
	}
	return h
}

func (m helmDetailModel) helmContentWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m helmDetailModel) helmLogMaxScroll() int {
	bodyH := m.helmBodyHeight() - 4
	if bodyH < 1 {
		bodyH = 1
	}
	maxScroll := len(m.logLines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

// buildHelmValuesTree builds a tree of outputNodes from helm chart values (a plain map).
// Unlike terraform output which has sensitive/type wrappers, helm values are raw JSON.
func buildHelmValuesTree(values map[string]interface{}, _ string) []outputNode {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var roots []outputNode
	for _, key := range keys {
		node := buildJSONNode(key, values[key], 0)
		roots = append(roots, *node)
	}
	return roots
}

// highlightHelmLogLine applies syntax highlighting to helm installation log lines
func highlightHelmLogLine(line string) string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	keywordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))

	lower := strings.ToLower(line)
	trimmed := strings.TrimSpace(line)

	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "fatal"):
		return errStyle.Render(line)
	case strings.Contains(lower, "warning") || strings.Contains(lower, "warn"):
		return warnStyle.Render(line)
	case strings.Contains(lower, "successfully") || strings.Contains(lower, "complete") ||
		strings.Contains(lower, "deployed") || strings.Contains(lower, "installed"):
		return successStyle.Render(line)
	case strings.HasPrefix(trimmed, "NAME:") || strings.HasPrefix(trimmed, "NAMESPACE:") ||
		strings.HasPrefix(trimmed, "STATUS:") || strings.HasPrefix(trimmed, "REVISION:") ||
		strings.HasPrefix(trimmed, "CHART:") || strings.HasPrefix(trimmed, "APP VERSION:") ||
		strings.HasPrefix(trimmed, "LAST DEPLOYED:") || strings.HasPrefix(trimmed, "TEST SUITE:"):
		// Key: value lines in helm status output
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			return keywordStyle.Render(parts[0]+":") + lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(parts[1])
		}
		return keywordStyle.Render(line)
	case strings.HasPrefix(trimmed, "NOTES:") || strings.HasPrefix(trimmed, "HOOKS:") ||
		strings.HasPrefix(trimmed, "MANIFEST:") || strings.HasPrefix(trimmed, "RESOURCES:"):
		return headerStyle.Render(line)
	case strings.HasPrefix(trimmed, "---"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(line)
	case strings.HasPrefix(trimmed, "apiVersion:") || strings.HasPrefix(trimmed, "kind:") ||
		strings.HasPrefix(trimmed, "metadata:") || strings.HasPrefix(trimmed, "spec:") ||
		strings.HasPrefix(trimmed, "data:") || strings.HasPrefix(trimmed, "type:"):
		// YAML-like keys in manifest output
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			return keywordStyle.Render(parts[0]+":") + lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(parts[1])
		}
		return keywordStyle.Render(line)
	case strings.HasPrefix(trimmed, "#"):
		return dimStyle.Render(line)
	case strings.TrimSpace(line) == "":
		return line
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(line)
	}
}
