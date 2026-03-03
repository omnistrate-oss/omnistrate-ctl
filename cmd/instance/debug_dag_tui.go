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
	"github.com/charmbracelet/x/ansi"
)

// backToDagMsg signals the detail view wants to return to DAG
type backToDagMsg struct{}

// tfProgressUpdateMsg carries updated terraform progress for DAG nodes
type tfProgressUpdateMsg struct {
	progressByID map[string]ResourceProgress
}

// wfProgressMsg carries workflow progress results
type wfProgressMsg struct {
	progressByID   map[string]ResourceProgress
	progressByKey  map[string]ResourceProgress
	progressByName map[string]ResourceProgress
	workflowID     string
	errors         []string
}

// dagRefreshTickMsg triggers a periodic DAG progress refresh
type dagRefreshTickMsg struct{}

// dagRefreshMsg carries combined refresh results for both workflow + terraform
type dagRefreshMsg struct {
	wf wfProgressMsg
	tf tfProgressUpdateMsg
}

const dagRefreshInterval = 5 * time.Second

type dagModel struct {
	debugData    DebugData
	instanceID   string
	plan         *PlanDAG
	lines        []string
	contentWidth int
	scrollX      int
	scrollY      int
	width        int
	height       int

	// Node selection
	selectableNodes []string   // ordered node IDs (level by level)
	nodeLevels      [][]string // nodes grouped by level (sorted within each level)
	cursorIndex     int
	showCursor      bool

	// Sub-view
	detailModel tea.Model
	inDetail    bool

	// Progress loading
	progressLoading bool
	wfResolved      bool
	tfResolved      bool
	wfResult        *wfProgressMsg              // stored until both resolve
	tfNodeProgress  map[string]ResourceProgress // terraform progress by node ID
	refreshing      bool                        // true during periodic refresh
	spinner         spinner.Model
}

func launchDebugTUI(data DebugData) error {
	model := newDagModel(data)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}

func newDagModel(data DebugData) dagModel {
	nodes, levels := buildSelectableNodeList(data.PlanDAG)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	hasNodes := data.PlanDAG != nil && len(data.PlanDAG.Nodes) > 0
	if hasNodes {
		data.PlanDAG.ProgressLoading = true
	}
	return dagModel{
		debugData:       data,
		instanceID:      data.InstanceID,
		plan:            data.PlanDAG,
		lines:           []string{},
		selectableNodes: nodes,
		nodeLevels:      levels,
		showCursor:      len(nodes) > 0,
		progressLoading: hasNodes,
		spinner:         s,
	}
}

func buildSelectableNodeList(plan *PlanDAG) ([]string, [][]string) {
	if plan == nil || len(plan.Levels) == 0 {
		return nil, nil
	}
	// Use the same layout ordering as the visual render.
	layout := orderPlanLevels(plan)
	var nodes []string
	for _, level := range layout.levels {
		nodes = append(nodes, level...)
	}
	return nodes, layout.levels
}

func (m dagModel) Init() tea.Cmd {
	if m.progressLoading {
		return tea.Batch(m.spinner.Tick, m.fetchWorkflowProgressForDAG(), m.fetchTerraformProgressForDAG())
	}
	return nil
}

func (m dagModel) fetchWorkflowProgressForDAG() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data := m.debugData
		// Build progress in a temporary plan to avoid shared-state race
		tmpPlan := &PlanDAG{Nodes: m.plan.Nodes, Levels: m.plan.Levels}
		attachWorkflowProgress(ctx, data.Token, data.ServiceID, data.EnvironmentID, data.InstanceID, tmpPlan)
		return wfProgressMsg{
			progressByID:   tmpPlan.ProgressByID,
			progressByKey:  tmpPlan.ProgressByKey,
			progressByName: tmpPlan.ProgressByName,
			workflowID:     tmpPlan.WorkflowID,
			errors:         tmpPlan.Errors,
		}
	}
}

func (m dagModel) fetchTerraformProgressForDAG() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data := m.debugData

		instanceData, err := fetchInstanceDataForResource(
			ctx, data.Token, data.ServiceID, data.EnvironmentID, data.InstanceID,
		)
		if err != nil {
			return tfProgressUpdateMsg{}
		}

		index, _, err := loadTerraformConfigMapIndexForInstance(ctx, data.Token, instanceData, data.InstanceID)
		if err != nil || index == nil {
			return tfProgressUpdateMsg{}
		}

		result := make(map[string]ResourceProgress)
		normalizedInstanceID := strings.ToLower(data.InstanceID)

		for nodeID, node := range m.plan.Nodes {
			if !strings.Contains(strings.ToLower(node.Type), "terraform") {
				continue
			}

			// Find progress configmap for this resource
			lowerResourceID := strings.ToLower(nodeID)
			var best *TerraformProgressData
			var bestTime string

			for _, cm := range index.progress {
				progressJSON, ok := cm.Data["progress"]
				if !ok {
					continue
				}
				var pd TerraformProgressData
				if jsonErr := json.Unmarshal([]byte(progressJSON), &pd); jsonErr != nil {
					continue
				}
				if strings.ToLower(pd.ResourceID) != lowerResourceID || strings.ToLower(pd.InstanceID) != normalizedInstanceID {
					continue
				}
				if best == nil || pd.StartedAt > bestTime {
					best = &pd
					bestTime = pd.StartedAt
				}
			}

			if best != nil {
				total := best.TotalResources
				if total == 0 {
					total = len(best.PlannedResources)
				}
				ready := 0
				for _, r := range best.Resources {
					if strings.ToLower(r.State) == "ready" {
						ready++
					}
				}
				pct := 0
				if total > 0 {
					pct = int(float64(ready) * 100 / float64(total))
				}
				status := strings.ToLower(best.Status)
				if status == "" {
					status = "running"
				}
				if (status == "completed" || status == "success") && pct == 0 {
					pct = 100
				}

				if total == 0 && pct == 0 && status != "completed" && status != "success" {
					continue
				}
				result[nodeID] = ResourceProgress{
					Percent:        pct,
					Status:         status,
					CompletedSteps: ready,
					TotalSteps:     total,
				}
			}
		}

		return tfProgressUpdateMsg{progressByID: result}
	}
}

func (m dagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If in detail sub-view, delegate
	if m.inDetail && m.detailModel != nil {
		switch dmsg := msg.(type) {
		case backToDagMsg:
			m.inDetail = false
			m.detailModel = nil
			m.rebuildLayout()
			var cmds []tea.Cmd
			cmds = append(cmds, tea.ClearScreen)
			// Resume progress loading if it was interrupted
			if m.progressLoading && (!m.wfResolved || !m.tfResolved) {
				if !m.wfResolved {
					cmds = append(cmds, m.fetchWorkflowProgressForDAG())
				}
				if !m.tfResolved {
					cmds = append(cmds, m.fetchTerraformProgressForDAG())
				}
				cmds = append(cmds, m.spinner.Tick)
			} else if m.isAnyNodeInProgress() {
				cmds = append(cmds, scheduleDagRefresh())
			}
			return m, tea.Batch(cmds...)
		case tea.WindowSizeMsg:
			// Update parent dimensions too for when we return from detail
			m.width = dmsg.Width
			m.height = dmsg.Height
		case wfProgressMsg:
			// Capture workflow progress even while in detail view
			m.wfResolved = true
			m.wfResult = &dmsg
			if m.plan != nil && dmsg.workflowID != "" {
				m.plan.WorkflowID = dmsg.workflowID
			}
			if dmsg.errors != nil && m.plan != nil {
				m.plan.Errors = append(m.plan.Errors, dmsg.errors...)
			}
			m.applyProgressIfReady()
		case tfProgressUpdateMsg:
			// Capture terraform progress even while in detail view
			m.tfResolved = true
			m.tfNodeProgress = dmsg.progressByID
			m.applyProgressIfReady()
		case dagRefreshTickMsg:
			// Handle DAG refresh even while in detail view
			if !m.refreshing && m.isAnyNodeInProgress() {
				m.refreshing = true
				updated, cmd := m.detailModel.Update(msg)
				m.detailModel = updated
				return m, tea.Batch(cmd, m.fetchDagRefresh())
			}
		case dagRefreshMsg:
			m.refreshing = false
			m.wfResult = &dmsg.wf
			m.tfNodeProgress = dmsg.tf.progressByID
			if m.plan != nil && dmsg.wf.workflowID != "" {
				m.plan.WorkflowID = dmsg.wf.workflowID
			}
			m.wfResolved = true
			m.tfResolved = true
			m.applyProgressIfReady()
			// Don't return the schedule cmd — we'll reschedule when we come back
		}
		updated, cmd := m.detailModel.Update(msg)
		m.detailModel = updated
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildLayout()
		return m, tea.ClearScreen
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "pgup":
			m.scrollY -= m.pageSize()
		case "pgdown":
			m.scrollY += m.pageSize()
		case "home", "g":
			m.scrollY = 0
		case "end", "G":
			m.scrollY = m.maxScrollY()
		case "up", "k":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.moveCursorInLevel(-1)
				m.rebuildLayout()
			}
		case "down", "j":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.moveCursorInLevel(1)
				m.rebuildLayout()
			}
		case "left", "h":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.moveCursorToLevel(-1)
				m.rebuildLayout()
			}
		case "right", "l":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.moveCursorToLevel(1)
				m.rebuildLayout()
			}
		case "tab":
			if m.showCursor && len(m.selectableNodes) > 0 && m.cursorIndex < len(m.selectableNodes)-1 {
				m.cursorIndex++
				m.rebuildLayout()
			}
		case "shift+tab":
			if m.showCursor && len(m.selectableNodes) > 0 && m.cursorIndex > 0 {
				m.cursorIndex--
				m.rebuildLayout()
			}
		case "enter":
			if m.showCursor && len(m.selectableNodes) > 0 {
				return m.openNodeDetail()
			}
		}
		m.clampScroll()
	case wfProgressMsg:
		m.wfResolved = true
		m.wfResult = &msg
		if m.plan != nil {
			if msg.workflowID != "" {
				m.plan.WorkflowID = msg.workflowID
			}
			m.plan.Errors = append(m.plan.Errors, msg.errors...)
		}
		if cmd := m.applyProgressIfReady(); cmd != nil {
			return m, cmd
		}
	case tfProgressUpdateMsg:
		m.tfResolved = true
		m.tfNodeProgress = msg.progressByID
		if cmd := m.applyProgressIfReady(); cmd != nil {
			return m, cmd
		}
	case dagRefreshTickMsg:
		if !m.refreshing && m.isAnyNodeInProgress() {
			m.refreshing = true
			return m, m.fetchDagRefresh()
		}
	case dagRefreshMsg:
		m.refreshing = false
		m.wfResult = &msg.wf
		m.tfNodeProgress = msg.tf.progressByID
		if m.plan != nil {
			if msg.wf.workflowID != "" {
				m.plan.WorkflowID = msg.wf.workflowID
			}
		}
		// Re-apply merged progress
		m.wfResolved = true
		m.tfResolved = true
		if cmd := m.applyProgressIfReady(); cmd != nil {
			return m, cmd
		}
	case spinner.TickMsg:
		if m.progressLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m.rebuildLayout()
			return m, cmd
		}
	}

	return m, nil
}

func (m *dagModel) applyProgressIfReady() tea.Cmd {
	if !m.wfResolved || !m.tfResolved {
		return nil
	}
	// Both resolved — stop loading, merge results
	m.progressLoading = false
	if m.plan != nil {
		m.plan.ProgressLoading = false
		m.plan.ProgressByID = make(map[string]ResourceProgress)
		m.plan.ProgressByKey = make(map[string]ResourceProgress)
		m.plan.ProgressByName = make(map[string]ResourceProgress)

		// Apply workflow progress as base
		if m.wfResult != nil {
			for id, prog := range m.wfResult.progressByID {
				m.plan.ProgressByID[id] = prog
			}
			for key, prog := range m.wfResult.progressByKey {
				m.plan.ProgressByKey[key] = prog
			}
			for name, prog := range m.wfResult.progressByName {
				m.plan.ProgressByName[name] = prog
			}
		}

		// Overwrite with terraform progress (terraform wins for tf nodes)
		for id, prog := range m.tfNodeProgress {
			m.plan.ProgressByID[id] = prog
			node, ok := m.plan.Nodes[id]
			if !ok {
				continue
			}
			if node.Key != "" {
				m.plan.ProgressByKey[node.Key] = prog
			}
			if node.Name != "" {
				m.plan.ProgressByName[node.Name] = prog
			}
		}
	}
	m.rebuildLayout()

	// Schedule periodic refresh if any node is still in progress
	if m.isAnyNodeInProgress() {
		return scheduleDagRefresh()
	}
	return nil
}

func (m dagModel) isAnyNodeInProgress() bool {
	if m.plan == nil {
		return false
	}
	for _, node := range m.plan.Nodes {
		prog, ok := progressForNode(m.plan, node)
		if !ok {
			continue
		}
		s := strings.ToLower(prog.Status)
		if s == "running" || s == "in_progress" || s == "creating" || s == "updating" || s == "pending" {
			return true
		}
		if prog.Percent > 0 && prog.Percent < 100 {
			return true
		}
	}
	return false
}

func scheduleDagRefresh() tea.Cmd {
	return tea.Tick(dagRefreshInterval, func(time.Time) tea.Msg {
		return dagRefreshTickMsg{}
	})
}

func (m dagModel) fetchDagRefresh() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data := m.debugData

		// Fetch workflow progress
		var wf wfProgressMsg
		if m.plan != nil {
			tmpPlan := &PlanDAG{Nodes: m.plan.Nodes, Levels: m.plan.Levels}
			attachWorkflowProgress(ctx, data.Token, data.ServiceID, data.EnvironmentID, data.InstanceID, tmpPlan)
			wf = wfProgressMsg{
				progressByID:   tmpPlan.ProgressByID,
				progressByKey:  tmpPlan.ProgressByKey,
				progressByName: tmpPlan.ProgressByName,
				workflowID:     tmpPlan.WorkflowID,
			}
		}

		// Fetch terraform progress
		tf := tfProgressUpdateMsg{progressByID: make(map[string]ResourceProgress)}
		instanceData, err := fetchInstanceDataForResource(ctx, data.Token, data.ServiceID, data.EnvironmentID, data.InstanceID)
		if err == nil {
			index, _, err := loadTerraformConfigMapIndexForInstance(ctx, data.Token, instanceData, data.InstanceID)
			if err == nil && index != nil {
				normalizedInstanceID := strings.ToLower(data.InstanceID)
				for nodeID, node := range m.plan.Nodes {
					if !strings.Contains(strings.ToLower(node.Type), "terraform") {
						continue
					}
					lowerResourceID := strings.ToLower(nodeID)
					var best *TerraformProgressData
					var bestTime string
					for _, cm := range index.progress {
						progressJSON, ok := cm.Data["progress"]
						if !ok {
							continue
						}
						var pd TerraformProgressData
						if jsonErr := json.Unmarshal([]byte(progressJSON), &pd); jsonErr != nil {
							continue
						}
						if strings.ToLower(pd.ResourceID) != lowerResourceID || strings.ToLower(pd.InstanceID) != normalizedInstanceID {
							continue
						}
						if best == nil || pd.StartedAt > bestTime {
							best = &pd
							bestTime = pd.StartedAt
						}
					}
					if best != nil {
						total := best.TotalResources
						if total == 0 {
							total = len(best.PlannedResources)
						}
						ready := 0
						for _, r := range best.Resources {
							if strings.ToLower(r.State) == "ready" {
								ready++
							}
						}
						pct := 0
						if total > 0 {
							pct = int(float64(ready) * 100 / float64(total))
						}
						status := strings.ToLower(best.Status)
						if status == "" {
							status = "running"
						}
						if (status == "completed" || status == "success") && pct == 0 {
							pct = 100
						}
						if total == 0 && pct == 0 && status != "completed" && status != "success" {
							continue
						}
						tf.progressByID[nodeID] = ResourceProgress{
							Percent:        pct,
							Status:         status,
							CompletedSteps: ready,
							TotalSteps:     total,
						}
					}
				}
			}
		}

		return dagRefreshMsg{wf: wf, tf: tf}
	}
}

// cursorLevelPos returns the current level index and position within that level.
func (m *dagModel) cursorLevelPos() (int, int) {
	curNode := m.selectableNodes[m.cursorIndex]
	for li, lv := range m.nodeLevels {
		for pi, id := range lv {
			if id == curNode {
				return li, pi
			}
		}
	}
	return 0, 0
}

// moveCursorInLevel moves up (dir=-1) or down (dir=1) within the current level.
// Stops at boundaries — does not wrap or cross levels.
func (m *dagModel) moveCursorInLevel(dir int) {
	if len(m.nodeLevels) == 0 {
		return
	}
	lvl, pos := m.cursorLevelPos()
	newPos := pos + dir
	if newPos < 0 || newPos >= len(m.nodeLevels[lvl]) {
		return
	}
	targetID := m.nodeLevels[lvl][newPos]
	for i, id := range m.selectableNodes {
		if id == targetID {
			m.cursorIndex = i
			return
		}
	}
}

// moveCursorToLevel moves to the next (dir=1) or previous (dir=-1) DAG level,
// keeping within-level position as close as possible. Stops at boundaries.
func (m *dagModel) moveCursorToLevel(dir int) {
	if len(m.nodeLevels) <= 1 {
		return
	}
	lvl, pos := m.cursorLevelPos()
	newLvl := lvl + dir
	if newLvl < 0 || newLvl >= len(m.nodeLevels) {
		return
	}
	tgtNodes := m.nodeLevels[newLvl]
	tgtPos := pos
	if tgtPos >= len(tgtNodes) {
		tgtPos = len(tgtNodes) - 1
	}
	targetID := tgtNodes[tgtPos]
	for i, id := range m.selectableNodes {
		if id == targetID {
			m.cursorIndex = i
			return
		}
	}
}

func (m dagModel) openNodeDetail() (tea.Model, tea.Cmd) {
	nodeID := m.selectableNodes[m.cursorIndex]
	node, ok := m.plan.Nodes[nodeID]
	if !ok {
		return m, nil
	}

	lower := strings.ToLower(node.Type)
	if strings.Contains(lower, "terraform") {
		detail := newTerraformDetailModel(node, m.debugData)
		detail.width = m.width
		detail.height = m.height
		detail.progressBar.Width = m.width - 40
		if detail.progressBar.Width < 20 {
			detail.progressBar.Width = 20
		}
		m.detailModel = detail
		m.inDetail = true
		return m, detail.Init()
	}

	if strings.Contains(lower, "helm") {
		detail := newHelmDetailModel(node, m.debugData)
		detail.width = m.width
		detail.height = m.height
		m.detailModel = detail
		m.inDetail = true
		return m, detail.Init()
	}

	// For other resource types, do nothing for now
	return m, nil
}

func (m dagModel) View() string {
	if m.inDetail && m.detailModel != nil {
		return m.detailModel.View()
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	bodyWidth, bodyHeight := m.bodySize()
	body := m.renderBody(bodyWidth, bodyHeight)

	header := m.renderHeader()
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help)
}

func (m dagModel) bodySize() (int, int) {
	headerHeight := 1
	helpHeight := 1
	bodyHeight := m.height - headerHeight - helpHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	bodyWidth := m.width
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	return bodyWidth, bodyHeight
}

func (m dagModel) renderHeader() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1)
	text := fmt.Sprintf("Deployment Plan · %s", m.instanceID)
	if m.plan != nil && m.plan.WorkflowID != "" {
		text += fmt.Sprintf(" · workflow: %s", m.plan.WorkflowID)
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func (m dagModel) renderHelp() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	var text string
	if m.showCursor && len(m.selectableNodes) > 0 {
		nodeID := m.selectableNodes[m.cursorIndex]
		node := m.plan.Nodes[nodeID]
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Bold(true)
		text = fmt.Sprintf("tab/shift+tab: select resource  enter: open  arrows: scroll  q: quit  │  %s", selectedStyle.Render(nodeLabel(node)))
	} else {
		text = "arrows: scroll  pgup/pgdn: page  home/end: jump  q: quit"
	}
	return lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top, style.Render(text))
}

func (m dagModel) renderBody(width, height int) string {
	if len(m.lines) == 0 {
		return ""
	}

	startY := clamp(m.scrollY, 0, m.maxScrollY())

	visible := make([]string, height)
	for i := 0; i < height; i++ {
		lineIndex := startY + i
		line := ""
		if lineIndex < len(m.lines) {
			line = sliceLineANSI(m.lines[lineIndex], m.scrollX, width)
		}
		visible[i] = padRightANSI(line, width)
	}

	return strings.Join(visible, "\n")
}

func (m dagModel) pageSize() int {
	_, height := m.bodySize()
	if height < 1 {
		return 1
	}
	return height
}

func (m dagModel) maxScrollY() int {
	_, height := m.bodySize()
	maxVal := len(m.lines) - height
	if maxVal < 0 {
		return 0
	}
	return maxVal
}

func (m dagModel) maxScrollX() int {
	bodyWidth, _ := m.bodySize()
	maxVal := m.contentWidth - bodyWidth
	if maxVal < 0 {
		return 0
	}
	return maxVal
}

func (m *dagModel) clampScroll() {
	if m.scrollY < 0 {
		m.scrollY = 0
	}
	if m.scrollY > m.maxScrollY() {
		m.scrollY = m.maxScrollY()
	}
	if m.scrollX < 0 {
		m.scrollX = 0
	}
	if m.scrollX > m.maxScrollX() {
		m.scrollX = m.maxScrollX()
	}
}

func (m *dagModel) rebuildLayout() {
	bodyWidth, _ := m.bodySize()
	if bodyWidth < 1 {
		bodyWidth = m.width
	}

	selectedNodeID := ""
	if m.showCursor && len(m.selectableNodes) > 0 {
		selectedNodeID = m.selectableNodes[m.cursorIndex]
	}

	// Advance spinner tick for loading nodes
	if m.plan != nil && m.progressLoading {
		m.plan.SpinnerTick++
	}

	m.lines = renderPlanDAGStyledWithSelection(m.plan, bodyWidth, selectedNodeID)
	m.contentWidth = maxLineWidthANSI(m.lines)
	m.clampScroll()
}

func maxLineWidthANSI(lines []string) int {
	maxVal := 0
	for _, line := range lines {
		width := ansi.StringWidth(line)
		if width > maxVal {
			maxVal = width
		}
	}
	return maxVal
}

func sliceLineANSI(line string, start, width int) string {
	if width <= 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	end := start + width
	return ansi.Cut(line, start, end)
}

func padRightANSI(text string, width int) string {
	length := ansi.StringWidth(text)
	if length >= width {
		return text
	}
	return text + strings.Repeat(" ", width-length)
}

func clamp(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}
