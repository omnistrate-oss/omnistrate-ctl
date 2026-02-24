package instance

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// backToDagMsg signals the detail view wants to return to DAG
type backToDagMsg struct{}

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
	selectableNodes []string // ordered node IDs
	cursorIndex     int
	showCursor      bool

	// Sub-view
	detailModel tea.Model
	inDetail    bool
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
	nodes := buildSelectableNodeList(data.PlanDAG)
	return dagModel{
		debugData:       data,
		instanceID:      data.InstanceID,
		plan:            data.PlanDAG,
		lines:           []string{},
		selectableNodes: nodes,
		showCursor:      len(nodes) > 0,
	}
}

func buildSelectableNodeList(plan *PlanDAG) []string {
	if plan == nil || len(plan.Levels) == 0 {
		return nil
	}
	var nodes []string
	for _, level := range plan.Levels {
		sorted := append([]string{}, level...)
		sort.Strings(sorted)
		nodes = append(nodes, sorted...)
	}
	return nodes
}

func (m dagModel) Init() tea.Cmd {
	return nil
}

func (m dagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If in detail sub-view, delegate
	if m.inDetail && m.detailModel != nil {
		switch msg.(type) {
		case backToDagMsg:
			m.inDetail = false
			m.detailModel = nil
			m.rebuildLayout()
			return m, tea.ClearScreen
		case tea.WindowSizeMsg:
			// Update parent dimensions too for when we return from detail
			wsm := msg.(tea.WindowSizeMsg)
			m.width = wsm.Width
			m.height = wsm.Height
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
		case "up":
			m.scrollY--
		case "down":
			m.scrollY++
		case "left":
			m.scrollX--
		case "right":
			m.scrollX++
		case "pgup":
			m.scrollY -= m.pageSize()
		case "pgdown":
			m.scrollY += m.pageSize()
		case "home", "g":
			m.scrollY = 0
		case "end", "G":
			m.scrollY = m.maxScrollY()
		case "tab":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.cursorIndex = (m.cursorIndex + 1) % len(m.selectableNodes)
				m.rebuildLayout()
			}
		case "shift+tab":
			if m.showCursor && len(m.selectableNodes) > 0 {
				m.cursorIndex = (m.cursorIndex - 1 + len(m.selectableNodes)) % len(m.selectableNodes)
				m.rebuildLayout()
			}
		case "enter":
			if m.showCursor && len(m.selectableNodes) > 0 {
				return m.openNodeDetail()
			}
		}
		m.clampScroll()
	}

	return m, nil
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

	// For non-terraform resources, do nothing for now
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
	max := len(m.lines) - height
	if max < 0 {
		return 0
	}
	return max
}

func (m dagModel) maxScrollX() int {
	bodyWidth, _ := m.bodySize()
	max := m.contentWidth - bodyWidth
	if max < 0 {
		return 0
	}
	return max
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

	m.lines = renderPlanDAGStyledWithSelection(m.plan, bodyWidth, selectedNodeID)
	m.contentWidth = maxLineWidthANSI(m.lines)
	m.clampScroll()
}

func maxLineWidthANSI(lines []string) int {
	max := 0
	for _, line := range lines {
		width := ansi.StringWidth(line)
		if width > max {
			max = width
		}
	}
	return max
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

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
