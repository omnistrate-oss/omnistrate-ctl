package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InteractiveListItem represents a single item in the interactive list.
type InteractiveListItem struct {
	name        string
	description string
	id          string
	status      string
	jsonData    string
}

func (i InteractiveListItem) Title() string       { return i.name }
func (i InteractiveListItem) Description() string { return i.description }
func (i InteractiveListItem) FilterValue() string { return i.name + " " + i.description }
func (i InteractiveListItem) ID() string          { return i.id }
func (i InteractiveListItem) Status() string      { return i.status }
func (i InteractiveListItem) JSONData() string    { return i.jsonData }

// InteractiveListConfig configures the interactive list behavior.
type InteractiveListConfig struct {
	Title    string
	Items    []InteractiveListItem
	OnSelect func(item InteractiveListItem) error
	ShowJSON bool
}

// NewInteractiveListItem creates a new list item from structured data.
func NewInteractiveListItem(name, description, id, status string, rawJSON json.RawMessage) InteractiveListItem {
	return InteractiveListItem{
		name:        name,
		description: description,
		id:          id,
		status:      status,
		jsonData:    string(rawJSON),
	}
}

// statusStyle returns styled status text with color-coded badges.
func statusStyle(status string) string {
	s := strings.ToUpper(status)
	switch {
	case strings.Contains(s, "RUNNING") || strings.Contains(s, "READY") || strings.Contains(s, "COMPLETE") || strings.Contains(s, "SUCCESS"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("● " + status)
	case strings.Contains(s, "FAIL") || strings.Contains(s, "ERROR") || strings.Contains(s, "DEGRADED"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("● " + status)
	case strings.Contains(s, "PENDING") || strings.Contains(s, "PROGRESS") || strings.Contains(s, "CREATING") || strings.Contains(s, "UPDATING") || strings.Contains(s, "DELETING"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("● " + status)
	case strings.Contains(s, "STOP") || strings.Contains(s, "DISABLED"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("● " + status)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("● " + status)
	}
}

// interactiveListDelegate renders list items with status badges.
type interactiveListDelegate struct {
	list.DefaultDelegate
}

func newInteractiveListDelegate() interactiveListDelegate {
	d := interactiveListDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	d.SetHeight(2)
	d.SetSpacing(0)
	return d
}

func (d interactiveListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(InteractiveListItem)
	if !ok {
		d.DefaultDelegate.Render(w, m, index, item)
		return
	}

	isSelected := index == m.Index()

	titleStyle := lipgloss.NewStyle().Padding(0, 0, 0, 2)
	descStyle := lipgloss.NewStyle().Padding(0, 0, 0, 2).
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

	if isSelected {
		titleStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
			Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
			Padding(0, 0, 0, 1)
		descStyle = titleStyle.
			Foreground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"})
	}

	title := i.Title()
	if i.Status() != "" {
		title = fmt.Sprintf("%s  %s", title, statusStyle(i.Status()))
	}

	fmt.Fprintf(w, "%s\n%s", titleStyle.Render(title), descStyle.Render(i.Description()))
}

// interactiveListModel wraps the bubbles list with select behavior.
type interactiveListModel struct {
	list     list.Model
	selected *InteractiveListItem
	showJSON bool
	quitting bool
}

func (m interactiveListModel) Init() tea.Cmd {
	return nil
}

func (m interactiveListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, m.list.KeyMap.Quit):
			m.quitting = true
			return m, tea.Quit
		case msg.String() == "enter":
			if item, ok := m.list.SelectedItem().(InteractiveListItem); ok {
				m.selected = &item
			}
			return m, tea.Quit
		case msg.String() == "j" && m.showJSON:
			if item, ok := m.list.SelectedItem().(InteractiveListItem); ok {
				m.selected = &item
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m interactiveListModel) View() string {
	return m.list.View()
}

// RunInteractiveList launches an interactive list TUI. Returns the selected item or nil if cancelled.
func RunInteractiveList(cfg InteractiveListConfig) (*InteractiveListItem, error) {
	items := make([]list.Item, len(cfg.Items))
	for i, item := range cfg.Items {
		items[i] = item
	}

	delegate := newInteractiveListDelegate()
	l := list.New(items, delegate, 80, 24)
	l.Title = cfg.Title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().Padding(0, 0, 1, 2)
	l.Styles.HelpStyle = helpStyle

	m := interactiveListModel{
		list:     l,
		showJSON: cfg.ShowJSON,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("interactive list error: %w", err)
	}

	final := result.(interactiveListModel)
	if final.quitting || final.selected == nil {
		return nil, nil
	}

	return final.selected, nil
}
