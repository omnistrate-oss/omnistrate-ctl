package instance

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type dagModel struct {
	instanceID   string
	plan         *PlanDAG
	lines        []string
	contentWidth int
	scrollX      int
	scrollY      int
	width        int
	height       int
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
	return dagModel{
		instanceID: data.InstanceID,
		plan:       data.PlanDAG,
		lines:      []string{},
	}
}

func (m dagModel) Init() tea.Cmd {
	return nil
}

func (m dagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildLayout()
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
		}
		m.clampScroll()
	}

	return m, nil
}

func (m dagModel) View() string {
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
	text := "arrows: scroll  pgup/pgdn: page  home/end: jump  q: quit"
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
	m.lines = renderPlanDAGStyled(m.plan, bodyWidth)
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
