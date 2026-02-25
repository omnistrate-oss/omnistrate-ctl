package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabProgress  = 0
	tabTfFiles   = 1
	tabTfOutput  = 2
	tabLogs      = 3
	tabOpHistory = 4
	numTabs      = 5
)

var tabNames = []string{"Progress", "Terraform Files", "Terraform Output", "Logs", "Operation History"}

// fileContentMsg is sent when file content has been fetched from the pod
type fileContentMsg struct {
	content string
	err     error
}

// Messages
type terraformDataMsg struct {
	progress     *TerraformProgressData
	history      []TerraformHistoryEntry
	k8sConn      *k8sConnection
	fileTree     *TerraformFileTree
	tfOutputJSON string // latest terraform output JSON from configmap
	err          error
}

// progressRefreshMsg is sent when auto-refresh fetches updated progress data
type progressRefreshMsg struct {
	progress *TerraformProgressData
	history  []TerraformHistoryEntry
	err      error
}

// progressTickMsg triggers a progress data refresh
type progressTickMsg struct{}

type clearClipboardMsg struct{}

type terraformDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int
	scrollY   int

	// Loading state
	loading    bool
	refreshing bool // true during auto-refresh (non-blocking)
	spinner    spinner.Model

	// Progress tab data
	tfProgress  *TerraformProgressData
	history     []TerraformHistoryEntry
	progressBar progress.Model
	loadErr     error

	// K8s connection for file operations
	k8sConn *k8sConnection

	// Terraform Files tab data
	fileTree       *TerraformFileTree
	fileCursor     int
	viewingFile    bool
	fileContent    string
	fileContentErr error
	fileLoading    bool
	fileScroll     int

	// Terraform Output tab data
	tfOutputJSON string // raw JSON from the latest output.log
	outputTree   []outputNode
	outputCursor int

	// Logs tab data
	logLines     []string
	logChan      chan logLineMsg
	logCancel    context.CancelFunc // cancels the log polling goroutine
	logScroll    int
	logFollow    bool // auto-scroll to bottom
	logStreaming bool
	logDone      bool
	logErr       error
	logLabel     string // describes which operation's log is shown

	// Operation History tab data
	historyCursor int
	historyDates  []dateSection

	// Error modal (shown over operation history)
	errorModalText   string // raw error text; non-empty means modal is open
	errorModalScroll int
	errorModalOp     string // operation name for the modal title

	// Clipboard flash message
	clipboardMsg string
}

func newTerraformDetailModel(node PlanDAGNode, data DebugData) terraformDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return terraformDetailModel{
		node:        node,
		debugData:   data,
		activeTab:   tabProgress,
		loading:     true,
		spinner:     s,
		progressBar: p,
		logChan:     make(chan logLineMsg, 50),
		logFollow:   true,
	}
}

func (m terraformDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchData())
}

const progressRefreshInterval = 5 * time.Second

func (m terraformDetailModel) isProgressInFlight() bool {
	if m.tfProgress == nil {
		return false
	}
	total := m.tfProgress.TotalResources
	if total == 0 {
		total = len(m.tfProgress.PlannedResources)
	}
	ready := countByState(m.tfProgress.Resources, "ready")
	if total > 0 && ready >= total {
		return false
	}
	s := strings.ToLower(m.tfProgress.Status)
	return s == "in_progress" || s == "running" || s == "creating" || s == "updating"
}

func scheduleProgressRefresh() tea.Cmd {
	return tea.Tick(progressRefreshInterval, func(time.Time) tea.Msg {
		return progressTickMsg{}
	})
}

func (m terraformDetailModel) fetchProgressOnly() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		instanceData, err := fetchInstanceDataForResource(
			ctx, m.debugData.Token, m.debugData.ServiceID, m.debugData.EnvironmentID, m.debugData.InstanceID,
		)
		if err != nil {
			return progressRefreshMsg{err: err}
		}
		progress, history, _, err := fetchTerraformProgress(
			ctx, m.debugData.Token, instanceData, m.debugData.InstanceID, m.node.ID,
		)
		return progressRefreshMsg{progress: progress, history: history, err: err}
	}
}

func (m terraformDetailModel) fetchData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		instanceData, err := fetchInstanceDataForResource(
			ctx, m.debugData.Token, m.debugData.ServiceID, m.debugData.EnvironmentID, m.debugData.InstanceID,
		)
		if err != nil {
			return terraformDataMsg{err: err}
		}

		progress, history, conn, err := fetchTerraformProgress(
			ctx, m.debugData.Token, instanceData, m.debugData.InstanceID, m.node.ID,
		)
		if err != nil {
			return terraformDataMsg{err: err}
		}

		// Fetch terraform output JSON from configmap Files (tf-state)
		var tfOutputJSON string
		if conn != nil {
			index, indexErr := loadTerraformConfigMapIndex(ctx, conn.clientset, m.debugData.InstanceID)
			if indexErr == nil && index != nil {
				var tfData *TerraformData
				for _, key := range resourceConfigMapKeys(m.node.ID) {
					td := index.terraformDataForResource(key)
					if td != nil && len(td.Files) > 0 {
						tfData = td
						break
					}
				}
				if tfData != nil {
					tfOutputJSON = findLatestOutputLog(tfData.Files, history)
				}
			}
		}

		// Fetch file tree from the terraform executor pod
		var fileTree *TerraformFileTree
		if progress != nil && conn != nil && progress.TerraformName != "" {
			podName := terraformExecutorPodName(progress.TerraformName)
			// Try apply directory first (most common), then diff
			for _, op := range []string{"apply", "diff", "output"} {
				basePath := terraformFilesBasePath(progress.TerraformName, progress.InstanceID, op)
				tree, fetchErr := fetchTerraformFileTree(ctx, conn, terraformConfigMapNamespace, podName, basePath)
				if fetchErr == nil && tree != nil && len(tree.Flat) > 0 {
					fileTree = tree
					break
				}
			}
		}

		return terraformDataMsg{
			progress:     progress,
			history:      history,
			k8sConn:      conn,
			fileTree:     fileTree,
			tfOutputJSON: tfOutputJSON,
			err:          err,
		}
	}
}

func (m terraformDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progressBar.Width = m.width - 40
		if m.progressBar.Width < 20 {
			m.progressBar.Width = 20
		}
		return m, tea.ClearScreen
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.logCancel != nil {
				m.logCancel()
			}
			return m, tea.Quit
		case "esc":
			if m.errorModalText != "" {
				m.errorModalText = ""
				m.errorModalScroll = 0
				m.errorModalOp = ""
				return m, nil
			}
			if m.viewingFile {
				m.viewingFile = false
				m.fileContent = ""
				m.fileContentErr = nil
				m.fileScroll = 0
				return m, nil
			}
			// Cancel log polling goroutine before leaving
			if m.logCancel != nil {
				m.logCancel()
			}
			// Signal back to DAG view
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab", "right":
			if m.errorModalText != "" {
				return m, nil // block tab switching while modal is open
			}
			if !m.viewingFile {
				m.activeTab = (m.activeTab + 1) % numTabs
				m.scrollY = 0
			}
		case "shift+tab", "left":
			if m.errorModalText != "" {
				return m, nil
			}
			if !m.viewingFile {
				m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
				m.scrollY = 0
			}
		case "up", "k":
			if m.errorModalText != "" {
				if m.errorModalScroll > 0 {
					m.errorModalScroll--
				}
				return m, nil
			}
			if m.activeTab == tabTfFiles && !m.viewingFile && m.fileTree != nil {
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			} else if m.activeTab == tabTfOutput && len(m.outputTree) > 0 {
				if m.outputCursor > 0 {
					m.outputCursor--
				}
			} else if m.activeTab == tabOpHistory && len(m.historyDates) > 0 {
				rows := flattenTimeline(m.historyDates)
				if m.historyCursor > 0 {
					m.historyCursor--
				}
				if m.historyCursor >= len(rows) {
					m.historyCursor = len(rows) - 1
				}
			} else if m.activeTab == tabLogs {
				m.logFollow = false
				if m.logScroll > 0 {
					m.logScroll--
				}
			} else if m.viewingFile {
				if m.fileScroll > 0 {
					m.fileScroll--
				}
			} else {
				if m.scrollY > 0 {
					m.scrollY--
				}
			}
		case "down", "j":
			if m.errorModalText != "" {
				m.errorModalScroll++
				maxScroll := m.errorModalMaxScroll()
				if m.errorModalScroll > maxScroll {
					m.errorModalScroll = maxScroll
				}
				return m, nil
			}
			if m.activeTab == tabTfFiles && !m.viewingFile && m.fileTree != nil {
				if m.fileCursor < len(m.fileTree.Flat)-1 {
					m.fileCursor++
				}
			} else if m.activeTab == tabTfOutput && len(m.outputTree) > 0 {
				visibleNodes := flattenOutputTree(m.outputTree)
				if m.outputCursor < len(visibleNodes)-1 {
					m.outputCursor++
				}
			} else if m.activeTab == tabOpHistory && len(m.historyDates) > 0 {
				rows := flattenTimeline(m.historyDates)
				if m.historyCursor < len(rows)-1 {
					m.historyCursor++
				}
			} else if m.activeTab == tabLogs {
				m.logFollow = false
				m.logScroll++
				if m.logScroll > m.logMaxScroll() {
					m.logScroll = m.logMaxScroll()
				}
			} else if m.viewingFile {
				m.fileScroll++
				if m.fileScroll > m.fileScrollMax() {
					m.fileScroll = m.fileScrollMax()
				}
			} else {
				m.scrollY++
				if m.scrollY > m.progressMaxScroll() {
					m.scrollY = m.progressMaxScroll()
				}
			}
		case "enter":
			if m.activeTab == tabTfFiles && m.fileTree != nil && !m.viewingFile {
				if m.fileCursor >= 0 && m.fileCursor < len(m.fileTree.Flat) {
					entry := m.fileTree.Flat[m.fileCursor]
					if entry.IsDir {
						entry.Expanded = !entry.Expanded
						m.fileTree.rebuildFlat()
						if m.fileCursor >= len(m.fileTree.Flat) {
							m.fileCursor = len(m.fileTree.Flat) - 1
						}
					} else if m.k8sConn != nil {
						m.fileLoading = true
						m.viewingFile = true
						m.fileScroll = 0
						return m, m.fetchFileContent(entry.Path)
					}
				}
			} else if m.activeTab == tabTfOutput && len(m.outputTree) > 0 {
				visibleNodes := flattenOutputTree(m.outputTree)
				if m.outputCursor >= 0 && m.outputCursor < len(visibleNodes) {
					node := visibleNodes[m.outputCursor]
					if node.expandable {
						node.expanded = !node.expanded
					} else if node.sensitive {
						node.sensitiveShown = !node.sensitiveShown
						if node.sensitiveShown {
							node.value = node.realValue
						} else {
							node.value = "••••••••  (sensitive, press enter to reveal)"
						}
					}
				}
			} else if m.activeTab == tabOpHistory && len(m.historyDates) > 0 {
				rows := flattenTimeline(m.historyDates)
				if m.historyCursor >= 0 && m.historyCursor < len(rows) {
					row := rows[m.historyCursor]
					if row.isDateHeader {
						row.date.expanded = !row.date.expanded
					} else if row.isGroupHeader {
						row.group.expanded = !row.group.expanded
					} else if row.entry != nil && row.entry.Error != "" {
						// Open error modal
						m.errorModalText = strings.ReplaceAll(row.entry.Error, "\\n", "\n")
						m.errorModalOp = row.entry.Operation
						m.errorModalScroll = 0
					}
					newRows := flattenTimeline(m.historyDates)
					if m.historyCursor >= len(newRows) {
						m.historyCursor = len(newRows) - 1
					}
				}
			}
		case "pgup":
			if m.errorModalText != "" {
				m.errorModalScroll -= m.bodyHeight()
				if m.errorModalScroll < 0 {
					m.errorModalScroll = 0
				}
				return m, nil
			}
			if m.activeTab == tabLogs {
				m.logFollow = false
				m.logScroll -= m.bodyHeight()
				if m.logScroll < 0 {
					m.logScroll = 0
				}
			} else if m.viewingFile {
				m.fileScroll -= m.bodyHeight()
				if m.fileScroll < 0 {
					m.fileScroll = 0
				}
			} else {
				m.scrollY -= m.bodyHeight()
				if m.scrollY < 0 {
					m.scrollY = 0
				}
			}
		case "pgdown":
			if m.errorModalText != "" {
				m.errorModalScroll += m.bodyHeight()
				maxScroll := m.errorModalMaxScroll()
				if m.errorModalScroll > maxScroll {
					m.errorModalScroll = maxScroll
				}
				return m, nil
			}
			if m.activeTab == tabLogs {
				m.logFollow = false
				m.logScroll += m.bodyHeight()
				if m.logScroll > m.logMaxScroll() {
					m.logScroll = m.logMaxScroll()
				}
			} else if m.viewingFile {
				m.fileScroll += m.bodyHeight()
				if m.fileScroll > m.fileScrollMax() {
					m.fileScroll = m.fileScrollMax()
				}
			} else {
				m.scrollY += m.bodyHeight()
				if m.scrollY > m.progressMaxScroll() {
					m.scrollY = m.progressMaxScroll()
				}
			}
		case "f":
			if m.activeTab == tabLogs {
				m.logFollow = !m.logFollow
				if m.logFollow {
					// Snap to bottom
					bodyH := m.bodyHeight() - 4
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
		case "y":
			text := m.copyableContent()
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
	case terraformDataMsg:
		m.loading = false
		m.loadErr = msg.err
		m.tfProgress = msg.progress
		m.history = msg.history
		m.historyDates = buildTimelineSections(msg.history)
		m.k8sConn = msg.k8sConn
		m.fileTree = msg.fileTree
		m.tfOutputJSON = msg.tfOutputJSON
		if msg.tfOutputJSON != "" {
			m.outputTree = buildOutputTreeFromJSON(msg.tfOutputJSON)
		}
		// Start log watcher for apply/destroy logs from configmap
		var cmds []tea.Cmd
		if msg.k8sConn != nil {
			ctx, cancel := context.WithCancel(context.Background())
			m.logCancel = cancel
			m.logStreaming = true
			cmds = append(cmds,
				watchApplyDestroyLogs(ctx, msg.k8sConn, m.debugData.InstanceID, m.node.ID, m.history, m.logChan),
				waitForLogLines(m.logChan),
			)
		}
		if m.isProgressInFlight() {
			cmds = append(cmds, scheduleProgressRefresh())
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	case logLineMsg:
		if msg.replace {
			m.logLines = msg.lines
		} else {
			m.logLines = append(m.logLines, msg.lines...)
		}
		if msg.label != "" {
			m.logLabel = msg.label
		}
		// If follow mode is on, snap to bottom
		if m.logFollow {
			bodyH := m.bodyHeight() - 4
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
			// Cancel previous context if any, then restart
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
				watchApplyDestroyLogs(ctx, m.k8sConn, m.debugData.InstanceID, m.node.ID, m.history, m.logChan),
				waitForLogLines(m.logChan),
			)
		}
		m.logDone = true
	case progressTickMsg:
		if m.isProgressInFlight() && !m.refreshing {
			m.refreshing = true
			return m, m.fetchProgressOnly()
		}
	case progressRefreshMsg:
		m.refreshing = false
		if msg.err == nil {
			m.tfProgress = msg.progress
			m.history = msg.history
			// Don't rebuild timeline while user is reading error modal
			if m.errorModalText == "" {
				m.historyDates = buildTimelineSections(msg.history)
			}
		}
		if m.isProgressInFlight() {
			return m, scheduleProgressRefresh()
		}
	case fileContentMsg:
		m.fileLoading = false
		m.fileContent = msg.content
		m.fileContentErr = msg.err
	case spinner.TickMsg:
		if m.loading || m.fileLoading || m.refreshing || m.isProgressInFlight() || m.logStreaming {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m terraformDetailModel) logMaxScroll() int {
	bodyH := m.bodyHeight() - 4
	if bodyH < 1 {
		bodyH = 1
	}
	maxScroll := len(m.logLines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m terraformDetailModel) progressMaxScroll() int {
	content := m.renderProgressTab()
	lines := strings.Split(content, "\n")
	bodyH := m.bodyHeight()
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m terraformDetailModel) fileScrollMax() int {
	if m.fileContent == "" {
		return 0
	}
	lines := strings.Split(m.fileContent, "\n")
	// headerLines in renderFileContentView is 2
	bodyH := m.bodyHeight() - 2
	if bodyH < 1 {
		bodyH = 1
	}
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m terraformDetailModel) bodyHeight() int {
	// header(1) + tab row(3) + window bottom border(1) + window padding(2) + footer(1) = 8
	h := m.height - 8
	if h < 1 {
		h = 1
	}
	return h
}

// errorModalLines returns the non-empty lines of the error modal text.
func (m terraformDetailModel) errorModalLines() []string {
	var lines []string
	for _, line := range strings.Split(m.errorModalText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func (m terraformDetailModel) errorModalMaxScroll() int {
	// header(1) + border(2) + footer(1) + padding(2) = 6
	bodyH := m.height - 6
	if bodyH < 1 {
		bodyH = 1
	}
	maxScroll := len(m.errorModalLines()) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (m terraformDetailModel) renderErrorModal() string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Header
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("124")).Padding(0, 1)
	title := fmt.Sprintf("Error Detail · %s", m.errorModalOp)
	header := lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, titleStyle.Render(title))

	// Body
	bodyH := m.height - 4 // header + footer + padding
	if bodyH < 1 {
		bodyH = 1
	}
	maxCodeWidth := m.width - 10
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	lines := m.errorModalLines()
	totalLines := len(lines)

	scroll := m.errorModalScroll
	maxScroll := m.errorModalMaxScroll()
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + bodyH
	if end > totalLines {
		end = totalLines
	}

	var b strings.Builder
	for i := scroll; i < end; i++ {
		line := lines[i]
		runes := []rune(line)
		if len(runes) > maxCodeWidth {
			line = string(runes[:maxCodeWidth-1]) + "…"
		}
		lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", i+1))
		b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, errStyle.Render(line)))
	}
	// Pad remaining lines
	for i := end - scroll; i < bodyH; i++ {
		b.WriteString("\n")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	pos := ""
	if totalLines <= bodyH {
		pos = "all"
	} else if scroll == 0 {
		pos = "top"
	} else if end >= totalLines {
		pos = "end"
	} else if maxScroll > 0 {
		pos = fmt.Sprintf("%d%%", (scroll*100)/maxScroll)
	}
	footerText := fmt.Sprintf("↑↓/pgup/pgdn: scroll  y: copy  esc: close  [%d/%d %s]", scroll+bodyH, totalLines, pos)
	if m.clipboardMsg != "" {
		clipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		footerText = clipStyle.Render(m.clipboardMsg) + "  " + footerText
	}
	footer := lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, footerStyle.Render(footerText))

	return lipgloss.JoinVertical(lipgloss.Left, header, b.String(), footer)
}

// contentWidth returns the usable width inside the content window (minus borders and padding)
func (m terraformDetailModel) contentWidth() int {
	// window border(2) + window padding(2) = 4
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

// copyableContent returns the plain text content appropriate for the current tab/view.
func (m terraformDetailModel) copyableContent() string {
	if m.errorModalText != "" {
		return m.errorModalText
	}
	if m.viewingFile && m.fileContent != "" {
		return m.fileContent
	}
	switch m.activeTab {
	case tabLogs:
		if len(m.logLines) > 0 {
			return strings.Join(m.logLines, "\n")
		}
	case tabTfOutput:
		if m.tfOutputJSON != "" {
			return m.tfOutputJSON
		}
	}
	return ""
}

func (m terraformDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.errorModalText != "" {
		return m.renderErrorModal()
	}

	header := m.renderHeader()
	tabsAndBody := m.renderTabsWithBody()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabsAndBody, footer)
}

func (m terraformDetailModel) renderHeader() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1)
	text := fmt.Sprintf("Resource Detail · %s · %s", m.node.Key, m.node.Type)
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m terraformDetailModel) renderTabsWithBody() string {
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
	for i, name := range tabNames {
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

	// Fill gap between last tab and right edge with a bottom border
	rowWidth := lipgloss.Width(row)
	gapWidth := m.width - rowWidth - 2 // -2 for window left+right borders
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

	content := m.getTabContent()

	bodyH := m.bodyHeight()
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

func (m terraformDetailModel) getTabContent() string {
	var content string
	switch m.activeTab {
	case tabProgress:
		content = m.renderProgressTab()
	case tabTfFiles:
		// Files tab handles its own scrolling via fileCursor and fileScroll
		return m.renderTerraformFilesTab()
	case tabTfOutput:
		// Output tab handles its own scrolling
		return m.renderTerraformOutputTab()
	case tabLogs:
		// Logs tab handles its own scrolling
		return m.renderLogsTab()
	case tabOpHistory:
		// History tab handles its own scrolling
		return m.renderOperationHistoryTab()
	}

	lines := strings.Split(content, "\n")

	bodyH := m.bodyHeight()
	// Clamp scroll
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}

	start := m.scrollY
	end := start + bodyH
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]
	for len(visible) < bodyH {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

func (m terraformDetailModel) renderFooter() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	var text string
	if m.viewingFile {
		text = "esc: back to files  ↑↓/pgup/pgdn: scroll  y: copy  q: quit"
	} else if m.activeTab == tabTfFiles && m.fileTree != nil && len(m.fileTree.Flat) > 0 {
		text = "↑↓: navigate  enter: open/expand  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else if m.activeTab == tabTfOutput && len(m.outputTree) > 0 {
		text = "↑↓: navigate  enter: expand/collapse  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else if m.activeTab == tabLogs {
		text = "↑↓/pgup/pgdn: scroll  f: toggle follow  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else if m.activeTab == tabOpHistory && len(m.historyDates) > 0 {
		text = "↑↓: navigate  enter: expand/collapse  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else {
		text = "tab/shift+tab: switch tabs  ↑↓: scroll  esc: back  q: quit"
	}
	if m.clipboardMsg != "" {
		clipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		text = clipStyle.Render(m.clipboardMsg) + "  " + text
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func (m terraformDetailModel) renderProgressTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching terraform progress...", m.spinner.View())
	}

	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}

	if m.tfProgress == nil {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No terraform progress data available for this resource."))
	}

	var b strings.Builder
	p := m.tfProgress

	// Overall status header
	b.WriteString("\n")
	statusStyle := styleForStatus(p.Status)
	statusLine := fmt.Sprintf("  Status: %s", statusStyle.Render(p.Status))
	if m.isProgressInFlight() {
		statusLine += fmt.Sprintf("  %s", m.spinner.View())
	}
	b.WriteString(statusLine + "\n")

	if p.OperationID != "" {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		b.WriteString(fmt.Sprintf("  Operation: %s\n", subtleStyle.Render(p.OperationID)))
	}

	// Progress bar
	b.WriteString("\n")
	total := p.TotalResources
	if total == 0 {
		total = len(p.PlannedResources)
	}
	ready := countByState(p.Resources, "ready")
	var percent float64
	if total > 0 {
		percent = float64(ready) / float64(total)
	}

	bar := m.progressBar.ViewAs(percent)
	readyText := fmt.Sprintf("%d/%d resources ready", ready, total)
	b.WriteString(fmt.Sprintf("  %s  %s\n", bar, readyText))

	// Status counts
	b.WriteString("\n")
	counts := countResourceStates(p.Resources)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	b.WriteString(fmt.Sprintf("  %s\n", headerStyle.Render("Resource Status Summary")))

	for _, entry := range counts {
		icon := stateIcon(entry.state)
		sStyle := styleForStatus(entry.state)
		b.WriteString(fmt.Sprintf("    %s %s %d\n", icon, sStyle.Render(entry.state), entry.count))
	}

	// Timing
	if p.StartedAt != "" || p.CompletedAt != "" {
		b.WriteString("\n")
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		if p.StartedAt != "" {
			b.WriteString(fmt.Sprintf("  Started:   %s\n", subtleStyle.Render(p.StartedAt)))
		}
		if p.CompletedAt != "" {
			b.WriteString(fmt.Sprintf("  Completed: %s\n", subtleStyle.Render(p.CompletedAt)))
		}
	}

	// Resource list
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", headerStyle.Render(fmt.Sprintf("Resources (%d)", len(p.Resources)))))
	b.WriteString("\n")

	// Table header
	addrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	for _, res := range p.Resources {
		icon := stateIcon(res.State)
		sStyle := styleForStatus(res.State)
		stateStr := sStyle.Render(fmt.Sprintf("%-12s", res.State))
		addr := addrStyle.Render(res.Address)
		resType := typeStyle.Render(res.Type)
		b.WriteString(fmt.Sprintf("  %s %s  %s  %s\n", icon, stateStr, addr, resType))
	}

	return b.String()
}

type dateSection struct {
	date     string // "2026-02-24"
	groups   []operationGroup
	expanded bool
}

type operationGroup struct {
	operationID string
	entries     []TerraformHistoryEntry
	summary     string // "diff → apply → output"
	status      string // overall status from last entry
	startedAt   string
	completedAt string
	expanded    bool
}

// timelineRow is a single renderable row in the flattened timeline
type timelineRow struct {
	isDateHeader  bool
	isGroupHeader bool
	date          *dateSection
	group         *operationGroup
	entry         *TerraformHistoryEntry
	isLastChild   bool
}

func dateFromTimestamp(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return "(unknown date)"
}

func buildTimelineSections(history []TerraformHistoryEntry) []dateSection {
	if len(history) == 0 {
		return nil
	}

	// First build operation groups (newest first)
	groupMap := make(map[string]*operationGroup)
	var order []string

	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		opID := entry.OperationID
		if opID == "" {
			opID = "(unknown)"
		}
		g, exists := groupMap[opID]
		if !exists {
			g = &operationGroup{operationID: opID}
			groupMap[opID] = g
			order = append(order, opID)
		}
		g.entries = append(g.entries, entry)
	}

	// Finalize each group
	for _, opID := range order {
		g := groupMap[opID]

		// Reverse entries to chronological order
		for i, j := 0, len(g.entries)-1; i < j; i, j = i+1, j-1 {
			g.entries[i], g.entries[j] = g.entries[j], g.entries[i]
		}

		var ops []string
		for _, e := range g.entries {
			ops = append(ops, e.Operation)
		}
		g.summary = strings.Join(ops, " → ")
		g.status = g.entries[len(g.entries)-1].Status
		g.startedAt = g.entries[0].StartedAt
		for _, e := range g.entries {
			if e.CompletedAt != "" {
				g.completedAt = e.CompletedAt
			}
		}
	}

	// Group operation groups by date (newest first)
	dateMap := make(map[string]*dateSection)
	var dateOrder []string

	for _, opID := range order {
		g := groupMap[opID]
		d := dateFromTimestamp(g.startedAt)
		ds, exists := dateMap[d]
		if !exists {
			ds = &dateSection{date: d, expanded: true}
			dateMap[d] = ds
			dateOrder = append(dateOrder, d)
		}
		ds.groups = append(ds.groups, *g)
	}

	var sections []dateSection
	for _, d := range dateOrder {
		ds := dateMap[d]
		// Auto-expand first group of the newest date
		if len(sections) == 0 && len(ds.groups) > 0 {
			ds.groups[0].expanded = true
		}
		sections = append(sections, *ds)
	}

	return sections
}

func flattenTimeline(dates []dateSection) []timelineRow {
	var rows []timelineRow
	for i := range dates {
		d := &dates[i]
		rows = append(rows, timelineRow{
			isDateHeader: true,
			date:         d,
		})
		if d.expanded {
			for j := range d.groups {
				g := &d.groups[j]
				rows = append(rows, timelineRow{
					isGroupHeader: true,
					group:         g,
					date:          d,
				})
				if g.expanded {
					for k := range g.entries {
						rows = append(rows, timelineRow{
							group:       g,
							entry:       &g.entries[k],
							isLastChild: k == len(g.entries)-1,
							date:        d,
						})
					}
				}
			}
		}
	}
	return rows
}

func (m terraformDetailModel) renderOperationHistoryTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching operation history...", m.spinner.View())
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}

	if len(m.historyDates) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No operation history available for this resource."))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	totalOps := 0
	for _, d := range m.historyDates {
		totalOps += len(d.groups)
	}
	b.WriteString(fmt.Sprintf("  %s\n\n", headerStyle.Render(
		fmt.Sprintf("Operation History (%d days, %d operations, %d entries)", len(m.historyDates), totalOps, len(m.history)))))

	rows := flattenTimeline(m.historyDates)

	// Simple row-based viewport clipping (each row = 1 line, no inline expansion)
	totalRows := len(rows)
	visibleRows := m.bodyHeight() - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	scrollOffset := 0
	if m.historyCursor >= visibleRows {
		scrollOffset = m.historyCursor - visibleRows + 1
	}
	if scrollOffset > totalRows-visibleRows {
		scrollOffset = totalRows - visibleRows
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + visibleRows
	if end > totalRows {
		end = totalRows
	}

	// Styles
	dateStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	opIDStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedBg := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	opNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	for idx := scrollOffset; idx < end; idx++ {
		row := rows[idx]
		selected := idx == m.historyCursor

		cursor := "  "
		if selected {
			cursor = "▶ "
		}

		if row.isDateHeader {
			d := row.date
			arrow := "▸"
			if d.expanded {
				arrow = "▾"
			}
			countStr := dimStyle.Render(fmt.Sprintf("(%d operations)", len(d.groups)))
			line := fmt.Sprintf("%s %s  %s", arrow, dateStyle.Render(d.date), countStr)
			if selected {
				line = selectedBg.Render(line)
			}
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, line))
		} else if row.isGroupHeader {
			g := row.group

			statusIcon := timelineStatusIcon(g.status)

			displayID := g.operationID
			if len(displayID) > 8 {
				displayID = displayID[:8] + "…"
			}

			// Show only time portion since date is in the section header
			timeRange := formatHistoryTimeOnly(g.startedAt)
			if g.completedAt != "" && g.completedAt != g.startedAt {
				timeRange += " → " + formatHistoryTimeOnly(g.completedAt)
			}

			arrow := "▸"
			if g.expanded {
				arrow = "▾"
			}

			line := fmt.Sprintf("  %s %s %s  %s  %s  %s",
				statusIcon, arrow,
				opIDStyle.Render(displayID),
				summaryStyle.Render(g.summary),
				styleForStatus(g.status).Render(g.status),
				timeStyle.Render(timeRange),
			)

			if selected {
				line = selectedBg.Render(line)
			}

			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, line))
		} else {
			// Child entry row
			e := row.entry
			connector := "│   ├─"
			if row.isLastChild {
				connector = "│   └─"
			}

			sIcon := timelineStatusIcon(e.Status)
			operation := opNameStyle.Render(fmt.Sprintf("%-10s", e.Operation))
			status := styleForStatus(e.Status).Render(fmt.Sprintf("%-12s", e.Status))

			timeRange := formatHistoryTimeOnly(e.StartedAt)
			if e.CompletedAt != "" && e.CompletedAt != e.StartedAt {
				timeRange += " → " + formatHistoryTimeOnly(e.CompletedAt)
			}

			errorHint := ""
			if e.Error != "" {
				errorHint = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("  ▸ error (enter to view)")
			}

			line := fmt.Sprintf("  %s %s %s  %s  %s%s",
				dimStyle.Render(connector),
				sIcon,
				operation,
				status,
				timeStyle.Render(timeRange),
				errorHint,
			)

			if selected {
				line = selectedBg.Render(line)
			}

			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, line))
		}
	}

	// Scroll indicator
	if totalRows > visibleRows {
		pos := ""
		maxOffset := totalRows - visibleRows
		if maxOffset < 1 {
			maxOffset = 1
		}
		if scrollOffset == 0 {
			pos = "top"
		} else if end >= totalRows {
			pos = "end"
		} else {
			pct := (scrollOffset * 100) / maxOffset
			pos = fmt.Sprintf("%d%%", pct)
		}
		b.WriteString(fmt.Sprintf("\n  %s\n", dimStyle.Render(
			fmt.Sprintf("↑↓: navigate  enter: expand/collapse  [%d/%d %s]", m.historyCursor+1, totalRows, pos))))
	}

	return b.String()
}

func timelineStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "completed", "success", "ready":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("●")
	case "failed", "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("●")
	case "in_progress", "running", "creating", "updating":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("◐")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○")
	}
}

func formatHistoryTimeOnly(t string) string {
	if t == "" {
		return "—"
	}
	if len(t) > 19 {
		t = t[:19]
	}
	// Extract just the time portion after T
	parts := strings.SplitN(t, "T", 2)
	if len(parts) == 2 {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(parts[1])
	}
	return t
}

func (m terraformDetailModel) fetchFileContent(filePath string) tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		ctx := context.Background()
		content, err := fetchFileContentFromPod(ctx, m.k8sConn, m.fileTree.Namespace, m.fileTree.PodName, filePath)
		return fileContentMsg{content: content, err: err}
	})
}

func (m terraformDetailModel) renderTerraformFilesTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching terraform files...", m.spinner.View())
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}

	// If viewing a file, show file content
	if m.viewingFile {
		return m.renderFileContent()
	}

	// Show the file tree
	if m.fileTree == nil || len(m.fileTree.Flat) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No terraform files found for this resource."))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	b.WriteString(fmt.Sprintf("  %s\n\n", headerStyle.Render(fmt.Sprintf("Files in %s", m.fileTree.BasePath))))

	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true).Background(lipgloss.Color("62"))

	// Viewport: reserve 3 lines for header + 1 for footer hint
	visibleRows := m.bodyHeight() - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	totalEntries := len(m.fileTree.Flat)

	// Compute scroll offset to keep cursor visible
	scrollOffset := 0
	if m.fileCursor >= visibleRows {
		scrollOffset = m.fileCursor - visibleRows + 1
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

	for i := scrollOffset; i < end; i++ {
		entry := m.fileTree.Flat[i]
		indent := strings.Repeat("  ", entry.Depth)
		cursor := "  "
		if i == m.fileCursor {
			cursor = "▶ "
		}

		var icon, name string
		if entry.IsDir {
			if entry.Expanded {
				icon = "▾"
			} else {
				icon = "▸"
			}
			name = dirStyle.Render(entry.Name + "/")
		} else {
			icon = fileIcon(entry.Name)
			name = fileStyle.Render(entry.Name)
		}

		if i == m.fileCursor {
			name = selectedStyle.Render(entry.Name)
			if entry.IsDir {
				name += "/"
			}
		}

		b.WriteString(fmt.Sprintf("  %s%s%s %s\n", cursor, indent, icon, name))
	}

	// Scroll indicator
	if totalEntries > visibleRows {
		pos := ""
		if scrollOffset == 0 {
			pos = "top"
		} else if end >= totalEntries {
			pos = "end"
		} else {
			pct := (scrollOffset * 100) / (totalEntries - visibleRows)
			pos = fmt.Sprintf("%d%%", pct)
		}
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(fmt.Sprintf("\n  %s\n", dimStyle.Render(fmt.Sprintf("↑↓: navigate  enter: open/expand  esc: back  [%d/%d %s]", m.fileCursor+1, totalEntries, pos))))
	} else {
		b.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑↓: navigate  enter: open/expand  esc: back")))
	}

	return b.String()
}

func (m terraformDetailModel) renderFileContent() string {
	if m.fileLoading {
		return fmt.Sprintf("\n  %s Loading file...", m.spinner.View())
	}
	if m.fileContentErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.fileContentErr)))
	}

	var b strings.Builder
	headerLines := 0
	if m.fileTree != nil && m.fileCursor >= 0 && m.fileCursor < len(m.fileTree.Flat) {
		entry := m.fileTree.Flat[m.fileCursor]
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
		b.WriteString(fmt.Sprintf("  %s\n", headerStyle.Render(entry.RelPath)))
		b.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("esc: back to file list  ↑↓/pgup/pgdn: scroll")))
		headerLines = 3
	}

	lines := strings.Split(m.fileContent, "\n")
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Determine filename for syntax highlighting
	filename := ""
	if m.fileTree != nil && m.fileCursor >= 0 && m.fileCursor < len(m.fileTree.Flat) {
		filename = m.fileTree.Flat[m.fileCursor].Name
	}

	bodyH := m.bodyHeight() - headerLines
	if bodyH < 1 {
		bodyH = 1
	}

	// Max width for code: content width minus "  " (2) + linenum (4) + " │ " (3) = 9
	maxCodeWidth := m.contentWidth() - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	// Clamp file scroll
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.fileScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + bodyH
	if end > len(lines) {
		end = len(lines)
	}

	for i := scroll; i < end; i++ {
		line := lines[i]
		// Truncate to fit window width
		runes := []rune(line)
		if len(runes) > maxCodeWidth {
			runes = runes[:maxCodeWidth-1]
			line = string(runes) + "…"
		}
		lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", i+1))
		code := syntaxHighlightLine(line, filename)
		b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, code))
	}

	return b.String()
}

func fileIcon(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tf"):
		return "⬡"
	case strings.HasSuffix(lower, ".tfvars"):
		return "≡"
	case strings.HasSuffix(lower, ".json"):
		return "{ }"
	case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "―"
	case strings.HasSuffix(lower, ".sh"):
		return "$"
	case strings.HasSuffix(lower, ".lock"), strings.HasSuffix(lower, ".lock.hcl"):
		return "⊘"
	default:
		return "·"
	}
}

// Helper types and functions

type stateCount struct {
	state string
	count int
}

func countResourceStates(resources []TerraformResourceDetail) []stateCount {
	counts := make(map[string]int)
	for _, r := range resources {
		counts[r.State]++
	}

	result := make([]stateCount, 0, len(counts))
	for state, count := range counts {
		result = append(result, stateCount{state: state, count: count})
	}

	// Sort: ready first, then alphabetical
	sortOrder := map[string]int{"ready": 0, "in_progress": 1, "creating": 2, "failed": 3}
	for i := range result {
		if _, ok := sortOrder[result[i].state]; !ok {
			sortOrder[result[i].state] = 10
		}
	}

	sort.Slice(result, func(i, j int) bool {
		oi, oj := sortOrder[result[i].state], sortOrder[result[j].state]
		if oi != oj {
			return oi < oj
		}
		return result[i].state < result[j].state
	})

	return result
}

func countByState(resources []TerraformResourceDetail, state string) int {
	count := 0
	for _, r := range resources {
		if r.State == state {
			count++
		}
	}
	return count
}

func stateIcon(state string) string {
	switch strings.ToLower(state) {
	case "ready":
		return "✓"
	case "in_progress", "creating", "updating":
		return "◌"
	case "failed", "error":
		return "✗"
	default:
		return "·"
	}
}

func styleForStatus(status string) lipgloss.Style {
	switch strings.ToLower(status) {
	case "ready", "completed", "success":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	case "in_progress", "creating", "updating", "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	case "failed", "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	}
}
