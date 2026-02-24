package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabProgress  = 0
	tabTfFiles   = 1
	tabTfOutput  = 2
	tabOpHistory = 3
	numTabs      = 4
)

var tabNames = []string{"Progress", "Terraform Files", "Terraform Output", "Operation History"}

// fileContentMsg is sent when file content has been fetched from the pod
type fileContentMsg struct {
	content string
	err     error
}

// Messages
type terraformDataMsg struct {
	progress *TerraformProgressData
	history  []TerraformHistoryEntry
	k8sConn  *k8sConnection
	fileTree *TerraformFileTree
	err      error
}

type terraformDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int
	scrollY   int

	// Loading state
	loading bool
	spinner spinner.Model

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
	}
}

func (m terraformDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchData())
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
			progress: progress,
			history:  history,
			k8sConn:  conn,
			fileTree: fileTree,
			err:      err,
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
			return m, tea.Quit
		case "esc":
			if m.viewingFile {
				m.viewingFile = false
				m.fileContent = ""
				m.fileContentErr = nil
				m.fileScroll = 0
				return m, nil
			}
			// Signal back to DAG view
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab", "right":
			if !m.viewingFile {
				m.activeTab = (m.activeTab + 1) % numTabs
				m.scrollY = 0
			}
		case "shift+tab", "left":
			if !m.viewingFile {
				m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
				m.scrollY = 0
			}
		case "up", "k":
			if m.activeTab == tabTfFiles && !m.viewingFile && m.fileTree != nil {
				if m.fileCursor > 0 {
					m.fileCursor--
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
			if m.activeTab == tabTfFiles && !m.viewingFile && m.fileTree != nil {
				if m.fileCursor < len(m.fileTree.Flat)-1 {
					m.fileCursor++
				}
			} else if m.viewingFile {
				m.fileScroll++
			} else {
				m.scrollY++
			}
		case "enter":
			if m.activeTab == tabTfFiles && m.fileTree != nil && !m.viewingFile {
				if m.fileCursor >= 0 && m.fileCursor < len(m.fileTree.Flat) {
					entry := m.fileTree.Flat[m.fileCursor]
					if entry.IsDir {
						entry.Expanded = !entry.Expanded
						m.fileTree.rebuildFlat()
						// Clamp cursor
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
			}
		case "pgup":
			if m.viewingFile {
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
			if m.viewingFile {
				m.fileScroll += m.bodyHeight()
			} else {
				m.scrollY += m.bodyHeight()
			}
		}
	case terraformDataMsg:
		m.loading = false
		m.loadErr = msg.err
		m.tfProgress = msg.progress
		m.history = msg.history
		m.k8sConn = msg.k8sConn
		m.fileTree = msg.fileTree
	case fileContentMsg:
		m.fileLoading = false
		m.fileContent = msg.content
		m.fileContentErr = msg.err
	case spinner.TickMsg:
		if m.loading || m.fileLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m terraformDetailModel) bodyHeight() int {
	// header(1) + tab row(3) + window bottom border(1) + window padding(2) + footer(1) = 8
	h := m.height - 8
	if h < 1 {
		h = 1
	}
	return h
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

func (m terraformDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
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
		Width(m.width - 2).
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
		content = m.renderPlaceholderTab("Terraform Output")
	case tabOpHistory:
		content = m.renderPlaceholderTab("Operation History")
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
		text = "esc: back to files  ↑↓/pgup/pgdn: scroll  q: quit"
	} else if m.activeTab == tabTfFiles && m.fileTree != nil && len(m.fileTree.Flat) > 0 {
		text = "↑↓: navigate  enter: open/expand  tab/shift+tab: switch tabs  esc: back  q: quit"
	} else {
		text = "tab/shift+tab: switch tabs  ↑↓: scroll  esc: back  q: quit"
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
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(p.Status)))

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

func (m terraformDetailModel) renderPlaceholderTab(name string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	return fmt.Sprintf("\n  %s\n", style.Render(fmt.Sprintf("%s — not yet implemented", name)))
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
