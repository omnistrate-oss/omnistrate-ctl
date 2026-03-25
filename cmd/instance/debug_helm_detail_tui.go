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
	helmTabLogs     = 0
	helmTabValues   = 1
	helmTabWfErrors = 2
	helmNumTabs     = 3
)

var helmTabNames = []string{"Helm Logs", "Chart Values", "Workflow Events"}

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

	// Logs tab (streaming)
	logLines     []string
	logChan      chan logLineMsg
	logCancel    context.CancelFunc
	logScroll    int
	logFollow    bool
	logStreaming bool
	logDone      bool
	logErr       error

	// Values tab (tree explorer)
	valuesTree   []outputNode
	valuesCursor int

	// Workflow Errors tab
	wfErrors *workflowErrorsState

	// Clipboard flash message
	clipboardMsg string
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
		logChan:   make(chan logLineMsg, 50),
		logFollow: true,
		wfErrors:  &workflowErrorsState{},
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

// watchHelmLogs polls the DebugResourceInstance API every logPollInterval,
// extracts InstallLog for the given resource key, diffs against previous
// content, and sends new lines via the channel — mirroring watchApplyDestroyLogs.
func watchHelmLogs(ctx context.Context, dd DebugData, nodeKey string, ch chan logLineMsg) tea.Cmd {
	return func() tea.Msg {
		var prevLines []string

		for {
			debugResult, err := dataaccess.DebugResourceInstance(
				ctx, dd.Token, dd.ServiceID, dd.EnvironmentID, dd.InstanceID,
			)
			if err != nil {
				if ctx.Err() != nil {
					close(ch)
					return logStreamDoneMsg{}
				}
				close(ch)
				return logStreamDoneMsg{err: err}
			}

			if debugResult.ResourcesDebug != nil {
				for resourceKey, resourceDebugInfo := range *debugResult.ResourcesDebug {
					if resourceKey != nodeKey {
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
					if helmData.InstallLog == "" {
						continue
					}
					lines := strings.Split(helmData.InstallLog, "\n")
					// Trim trailing empty lines
					for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
						lines = lines[:len(lines)-1]
					}

					if len(prevLines) == 0 {
						// First fetch with content — send full
						select {
						case ch <- logLineMsg{lines: lines, replace: true}:
						case <-ctx.Done():
							close(ch)
							return logStreamDoneMsg{}
						}
						prevLines = lines
					} else if len(lines) > len(prevLines) {
						// More content — send only new lines
						select {
						case ch <- logLineMsg{lines: lines[len(prevLines):]}:
						case <-ctx.Done():
							close(ch)
							return logStreamDoneMsg{}
						}
						prevLines = lines
					} else if !slicesEqual(lines, prevLines) {
						// Content changed — replace all
						select {
						case ch <- logLineMsg{lines: lines, replace: true}:
						case <-ctx.Done():
							close(ch)
							return logStreamDoneMsg{}
						}
						prevLines = lines
					}
				}
			}

			select {
			case <-ctx.Done():
				close(ch)
				return logStreamDoneMsg{}
			case <-time.After(logPollInterval):
			}
		}
	}
}

func (m helmDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.logStreaming || m.wfErrors.refreshing || isWorkflowInProgress(m.getWfEvents()) {
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
		var cmds []tea.Cmd
		if m.helmData != nil {
			// Parse initial log lines
			if m.helmData.InstallLog != "" {
				m.logLines = strings.Split(m.helmData.InstallLog, "\n")
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
			// Start log polling
			ctx, cancel := context.WithCancel(context.Background())
			m.logCancel = cancel
			m.logStreaming = true
			cmds = append(cmds,
				watchHelmLogs(ctx, m.debugData, m.node.Key, m.logChan),
				waitForLogLines(m.logChan),
			)
		}
		if isWorkflowInProgress(m.getWfEvents()) {
			cmds = append(cmds, scheduleWfEventsRefresh(), scheduleWfCountdownTick())
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case logLineMsg:
		if msg.replace {
			m.logLines = msg.lines
		} else {
			m.logLines = append(m.logLines, msg.lines...)
		}
		if m.logFollow {
			bodyH := m.helmBodyHeight() - 4
			if bodyH < 1 {
				bodyH = 1
			}
			maxSc := len(m.logLines) - bodyH
			if maxSc < 0 {
				maxSc = 0
			}
			m.logScroll = maxSc
		}
		return m, waitForLogLines(m.logChan)

	case logStreamDoneMsg:
		m.logStreaming = false
		if msg.err != nil {
			if m.logCancel != nil {
				m.logCancel()
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.logCancel = cancel
			m.logChan = make(chan logLineMsg, 50)
			m.logStreaming = true
			m.logErr = nil
			return m, tea.Batch(
				tea.Tick(logPollInterval, func(time.Time) tea.Msg { return nil }),
				watchHelmLogs(ctx, m.debugData, m.node.Key, m.logChan),
				waitForLogLines(m.logChan),
			)
		}
		m.logDone = true

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
			if m.logCancel != nil {
				m.logCancel()
			}
			return m, tea.Quit
		case "esc":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalText = ""
				m.wfErrors.modalTitle = ""
				m.wfErrors.modalScroll = 0
				return m, nil
			}
			if m.logCancel != nil {
				m.logCancel()
			}
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab + 1) % helmNumTabs
			return m, nil
		case "shift+tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab - 1 + helmNumTabs) % helmNumTabs
			return m, nil
		case "up", "k":
			if m.wfErrors.modalText != "" {
				if m.wfErrors.modalScroll > 0 {
					m.wfErrors.modalScroll--
				}
				return m, nil
			}
			if m.activeTab == helmTabLogs {
				m.logFollow = false
				if m.logScroll > 0 {
					m.logScroll--
				}
			} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor > 0 {
					m.valuesCursor--
				}
				_ = visibleNodes
			} else if m.activeTab == helmTabWfErrors {
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
			if m.activeTab == helmTabLogs {
				m.logFollow = false
				m.logScroll++
				if m.logScroll > m.helmLogMaxScroll() {
					m.logScroll = m.helmLogMaxScroll()
				}
			} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
				visibleNodes := flattenOutputTree(m.valuesTree)
				if m.valuesCursor < len(visibleNodes)-1 {
					m.valuesCursor++
				}
			} else if m.activeTab == helmTabWfErrors {
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items)-1 {
					m.wfErrors.cursor++
				}
			}
		case "pgup":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalScroll -= m.helmBodyHeight()
				if m.wfErrors.modalScroll < 0 {
					m.wfErrors.modalScroll = 0
				}
				return m, nil
			}
			switch m.activeTab {
			case helmTabLogs:
				m.logFollow = false
				m.logScroll -= m.helmBodyHeight()
				if m.logScroll < 0 {
					m.logScroll = 0
				}
			case helmTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				pageItems := m.helmBodyHeight() / 2
				if pageItems < 1 {
					pageItems = 1
				}
				m.wfErrors.cursor -= pageItems
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
				_ = items
			}
		case "pgdown":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalScroll += m.helmBodyHeight()
				maxScroll := wfEventModalMaxScroll(m.wfErrors, m.width, m.height)
				if m.wfErrors.modalScroll > maxScroll {
					m.wfErrors.modalScroll = maxScroll
				}
				return m, nil
			}
			switch m.activeTab {
			case helmTabLogs:
				m.logFollow = false
				m.logScroll += m.helmBodyHeight()
				if m.logScroll > m.helmLogMaxScroll() {
					m.logScroll = m.helmLogMaxScroll()
				}
			case helmTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				pageItems := m.helmBodyHeight() / 2
				if pageItems < 1 {
					pageItems = 1
				}
				m.wfErrors.cursor += pageItems
				if m.wfErrors.cursor >= len(items) {
					m.wfErrors.cursor = len(items) - 1
				}
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "f":
			if m.activeTab == helmTabLogs {
				m.logFollow = !m.logFollow
				if m.logFollow {
					bodyH := m.helmBodyHeight() - 4
					if bodyH < 1 {
						bodyH = 1
					}
					maxSc := len(m.logLines) - bodyH
					if maxSc < 0 {
						maxSc = 0
					}
					m.logScroll = maxSc
				}
			}
		case "enter", "right", "l":
			if m.activeTab == helmTabWfErrors {
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor >= 0 && m.wfErrors.cursor < len(items) {
					item := items[m.wfErrors.cursor]
					if item.event != nil {
						m.wfErrors.modalText = formatEventDetail(item.event)
						m.wfErrors.modalTitle = extractEventAction(item.event.Message)
						m.wfErrors.modalScroll = 0
					}
				}
			} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
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
		case "y":
			text := m.helmCopyableContent()
			if text != "" {
				m.clipboardMsg = "Copying..."
				return m, copyToClipboardCmd(text)
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

func (m helmDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.wfErrors.modalText != "" {
		return renderWfEventModal(m.wfErrors, m.width, m.height)
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
		Width(m.width-2).
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
	case helmTabWfErrors:
		return m.renderHelmWfErrorsTab()
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
	statusText := ""
	if m.logStreaming {
		statusText = " ● LIVE"
	} else if m.logDone {
		statusText = " ○ ended"
	}
	followText := ""
	if m.logFollow {
		followText = " [following]"
	}
	b.WriteString(fmt.Sprintf("  %s\n\n",
		headerStyle.Render(fmt.Sprintf("Helm Install Log (%d lines%s%s)", len(m.logLines), statusText, followText)),
	))

	bodyH := m.helmBodyHeight() - 4
	if bodyH < 1 {
		bodyH = 1
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	maxCodeWidth := m.helmContentWidth() - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	// Expand all source lines into visual lines with wrapping
	vlines := expandLinesToVisual(m.logLines, maxCodeWidth)
	totalLines := len(vlines)
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

	for i := scroll; i < end; i++ {
		vl := vlines[i]
		styled := highlightHelmLogLine(vl.text)
		if vl.sourceNum > 0 {
			lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", vl.sourceNum))
			b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, styled))
		} else {
			b.WriteString(fmt.Sprintf("  %s   %s\n", "    ", styled))
		}
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
		followHint := "f: follow"
		if m.logFollow {
			followHint = "f: unfollow"
		}
		text = fmt.Sprintf("↑↓/pgup/pgdn: scroll  %s  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit", followHint)
	} else if m.activeTab == helmTabValues && len(m.valuesTree) > 0 {
		text = "↑↓: navigate  ←→/enter: expand/collapse  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else if m.activeTab == helmTabWfErrors {
		text = "↑↓/pgup/pgdn: scroll  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else {
		text = "tab/shift+tab: switch tabs  esc: back  q: quit"
	}
	if m.clipboardMsg != "" {
		clipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		text = clipStyle.Render(m.clipboardMsg) + "  " + text
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

// helmCopyableContent returns the plain text content appropriate for the current tab.
func (m helmDetailModel) helmCopyableContent() string {
	switch m.activeTab {
	case helmTabLogs:
		if len(m.logLines) > 0 {
			return strings.Join(m.logLines, "\n")
		}
	case helmTabValues:
		if m.helmData != nil && len(m.helmData.ChartValues) > 0 {
			raw, err := json.Marshal(m.helmData.ChartValues)
			if err == nil {
				return string(raw)
			}
		}
	case helmTabWfErrors:
		return workflowEventsCopyText(m.getWfEvents())
	}
	return ""
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
	maxCodeWidth := m.helmContentWidth() - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}
	vlines := expandLinesToVisual(m.logLines, maxCodeWidth)
	maxScroll := len(vlines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m helmDetailModel) getWfEvents() *ResourceWorkflowSteps {
	if m.debugData.PlanDAG != nil && m.debugData.PlanDAG.WorkflowStepsByKey != nil {
		return m.debugData.PlanDAG.WorkflowStepsByKey[m.node.Key]
	}
	return nil
}

func (m helmDetailModel) renderHelmWfErrorsTab() string {
	loading := m.debugData.PlanDAG != nil && m.debugData.PlanDAG.ProgressLoading
	steps := m.getWfEvents()
	enrichBootstrapSteps(steps, m.node.Key, m.debugData.PlanDAG)
	isLive := isWorkflowInProgress(steps)
	return renderWorkflowEventsTab(steps, m.wfErrors, m.helmBodyHeight(), m.helmContentWidth(), loading, m.spinner.View(), isLive)
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
