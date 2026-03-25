package utils

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	spinnerRunning  = 0
	spinnerComplete = 1
	spinnerError    = 2
)

type spinnerEntry struct {
	message string
	state   int
}

// SpinnerManager manages one or more spinners rendered inline in the terminal.
type SpinnerManager interface {
	AddSpinner(msg string) *Spinner
	Start()
	Stop()
	Running() bool
}

// Spinner is a handle to a single spinner entry in a SpinnerManager.
type Spinner struct {
	mgr *spinnerMgr
	idx int
}

// UpdateMessage changes the spinner's display message.
func (s *Spinner) UpdateMessage(msg string) {
	s.mgr.mu.Lock()
	defer s.mgr.mu.Unlock()
	if s.idx < len(s.mgr.entries) {
		s.mgr.entries[s.idx].message = msg
	}
}

// Complete marks the spinner as successfully completed.
func (s *Spinner) Complete() {
	s.mgr.mu.Lock()
	defer s.mgr.mu.Unlock()
	if s.idx < len(s.mgr.entries) {
		s.mgr.entries[s.idx].state = spinnerComplete
	}
}

// Error marks the spinner as failed.
func (s *Spinner) Error() {
	s.mgr.mu.Lock()
	defer s.mgr.mu.Unlock()
	if s.idx < len(s.mgr.entries) {
		s.mgr.entries[s.idx].state = spinnerError
	}
}

// NewSpinnerManager creates a new SpinnerManager backed by a bubbletea program.
func NewSpinnerManager() SpinnerManager {
	return &spinnerMgr{
		done: make(chan struct{}),
	}
}

type spinnerMgr struct {
	entries     []*spinnerEntry
	program     *tea.Program
	done        chan struct{}
	startOffset int
	mu          sync.RWMutex
}

// Running returns true if the spinner manager's bubbletea program is active.
func (sm *spinnerMgr) Running() bool {
	return sm.program != nil
}

// AddSpinner adds a new spinner with the given message. Can be called before or after Start().
func (sm *spinnerMgr) AddSpinner(msg string) *Spinner {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	entry := &spinnerEntry{message: msg}
	sm.entries = append(sm.entries, entry)
	return &Spinner{mgr: sm, idx: len(sm.entries) - 1}
}

// Start begins rendering the spinners. Entries added before and after Start() are rendered.
func (sm *spinnerMgr) Start() {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := spinnerModel{mgr: sm, spin: s}
	sm.program = tea.NewProgram(m)
	go func() {
		_, _ = sm.program.Run()
		close(sm.done)
	}()
}

// Stop sends a quit signal and waits for the bubbletea program to finish.
func (sm *spinnerMgr) Stop() {
	if sm.program != nil {
		sm.program.Send(spinnerQuitMsg{})
		<-sm.done
		sm.program = nil
		sm.done = make(chan struct{})

		sm.mu.Lock()
		sm.startOffset = len(sm.entries)
		sm.mu.Unlock()
	}
}

// spinnerQuitMsg signals the bubbletea program to quit.
type spinnerQuitMsg struct{}

// spinnerModel is the bubbletea model that renders spinner entries.
type spinnerModel struct {
	mgr  *spinnerMgr
	spin spinner.Model
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spin.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerQuitMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	m.mgr.mu.RLock()
	defer m.mgr.mu.RUnlock()

	completeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("✓")
	errorIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✗")

	entries := m.mgr.entries[m.mgr.startOffset:]
	if len(entries) == 0 {
		return ""
	}

	var lines []string
	for _, e := range entries {
		switch e.state {
		case spinnerComplete:
			lines = append(lines, fmt.Sprintf("  %s %s", completeIcon, e.message))
		case spinnerError:
			lines = append(lines, fmt.Sprintf("  %s %s", errorIcon, e.message))
		default:
			lines = append(lines, fmt.Sprintf("  %s %s", m.spin.View(), e.message))
		}
	}
	return strings.Join(lines, "\n")
}
