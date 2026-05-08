package utils

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
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

type spinnerStepEntry struct {
	entry  *spinnerEntry
	label  string
	detail bool
}

type spinnerStepGroup struct {
	index   int
	total   int
	title   string
	entries []spinnerStepEntry
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
	entries      []*spinnerEntry
	program      *tea.Program
	done         chan struct{}
	startOffset  int
	width        int
	useAltScreen bool
	mu           sync.RWMutex
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
	if sm.program != nil {
		return
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878"))

	p := progress.New(
		progress.WithSolidFill("#50C878"),
		progress.WithWidth(48),
	)

	sm.mu.RLock()
	useAltScreen := spinnerEntriesContainSteps(sm.entries[sm.startOffset:])
	width := sm.width
	sm.mu.RUnlock()
	if width <= 0 {
		width = 96
	}

	sm.useAltScreen = useAltScreen

	m := spinnerModel{mgr: sm, spin: s, progress: p, width: width, pulseOn: true}
	var opts []tea.ProgramOption
	if useAltScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	sm.program = tea.NewProgram(m, opts...)
	go func() {
		_, _ = sm.program.Run()
		close(sm.done)
	}()
}

// Stop sends a quit signal and waits for the bubbletea program to finish.
func (sm *spinnerMgr) Stop() {
	if sm.program != nil {
		finalView := ""
		if sm.useAltScreen {
			finalView = sm.finalGroupedDeploymentView()
		}
		sm.program.Send(spinnerQuitMsg{})
		<-sm.done
		sm.program = nil
		sm.done = make(chan struct{})
		if finalView != "" {
			fmt.Println(finalView)
		}

		sm.mu.Lock()
		sm.startOffset = len(sm.entries)
		sm.useAltScreen = false
		sm.mu.Unlock()
	}
}

func (sm *spinnerMgr) finalGroupedDeploymentView() string {
	sm.mu.RLock()
	width := sm.width
	if width <= 0 {
		width = 96
	}
	entries := append([]*spinnerEntry(nil), sm.entries[sm.startOffset:]...)
	sm.mu.RUnlock()

	if len(entries) == 0 {
		return ""
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878"))
	p := progress.New(
		progress.WithSolidFill("#50C878"),
		progress.WithWidth(48),
	)
	view, ok := spinnerModel{mgr: sm, spin: s, progress: p, width: width, pulseOn: true}.groupedDeploymentView(entries)
	if !ok {
		return ""
	}
	return view
}

// spinnerQuitMsg signals the bubbletea program to quit.
type spinnerQuitMsg struct{}

// spinnerModel is the bubbletea model that renders spinner entries.
type spinnerModel struct {
	mgr        *spinnerMgr
	spin       spinner.Model
	progress   progress.Model
	width      int
	pulseOn    bool
	pulseTicks int
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.mgr.mu.Lock()
		m.mgr.width = msg.Width
		m.mgr.mu.Unlock()
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		m.pulseTicks++
		if m.pulseTicks >= RegionGlobePulseFrames {
			m.pulseTicks = 0
			m.pulseOn = !m.pulseOn
		}
		return m, cmd
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

	if view, ok := m.groupedDeploymentView(entries); ok {
		return view
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

func (m spinnerModel) groupedDeploymentView(entries []*spinnerEntry) (string, bool) {
	groups, ok := spinnerStepGroups(entries)
	if !ok {
		return "", false
	}

	cardWidth := spinnerClamp(m.width-6, 72, 120)
	contentWidth := cardWidth - 6
	bar := m.progress
	bar.Width = spinnerClamp(contentWidth-28, 28, 64)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F8FAFC"))
	commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	categoryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CBD5E1")).Bold(true)
	completeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878")).Bold(true)
	runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7DD3FC")).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true)
	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#334155")).
		Foreground(lipgloss.Color("#D1D5DB")).
		Padding(1, 2).
		Width(cardWidth)

	var body strings.Builder
	body.WriteString(titleStyle.Render("omnistrate-ctl deploy"))
	body.WriteString("\n\n")
	body.WriteString(commandStyle.Render("$ resolving service build and instance deployment"))
	if provider, region, ok := spinnerDeploymentTarget(entries); ok {
		body.WriteString("\n\n")
		body.WriteString(RenderRegionGlobeWithProvider(provider, region, spinnerClamp(contentWidth, 42, 64), m.pulseOn))
	}

	for _, group := range groups {
		complete, total, hasRunning, hasError := spinnerStepProgress(group.entries)
		percent := spinnerStepPercent(complete, total, hasRunning)
		submitted := spinnerStepGroupSubmitted(group)
		pendingSubmit := group.index == 2 && !submitted && !hasError
		if pendingSubmit && !hasRunning && percent >= 1 {
			percent = 0.92
		}
		status := mutedStyle.Render(fmt.Sprintf("%d/%d complete", complete, total))
		if hasError {
			status = errorStyle.Render("failed")
		} else if submitted {
			status = runningStyle.Render("submitted")
		} else if pendingSubmit {
			status = runningStyle.Render("running")
		} else if total > 0 && complete == total {
			status = completeStyle.Render("complete")
		} else if hasRunning {
			status = runningStyle.Render("running")
		}

		body.WriteString("\n\n")
		fmt.Fprintf(&body, "%s  %s\n", completeStyle.Render(group.title), status)
		if submitted {
			body.WriteString(mutedStyle.Render("Deployment workflow continues below."))
			body.WriteString("\n")
		} else {
			body.WriteString(bar.ViewAs(percent))
			body.WriteString("\n")
		}

		currentCategory := ""
		for _, item := range group.entries {
			category := spinnerStepCategory(group.index, item.label, item.detail)
			if category != "" && category != currentCategory {
				currentCategory = category
				fmt.Fprintf(&body, "  %s\n", categoryStyle.Render(category))
			}

			icon := spinnerStepIcon(item.entry.state, m.spin.View(), completeStyle, runningStyle, errorStyle)
			label := cleanSpinnerStepLabel(item.label)
			if item.detail {
				label = strings.TrimSpace(label)
			}
			indent := "    "
			if item.detail {
				indent = "      "
			}
			for i, line := range spinnerWrapText(label, contentWidth-lipgloss.Width(indent)-lipgloss.Width(icon)-1) {
				if item.detail {
					line = mutedStyle.Render(line)
				}
				if i == 0 {
					fmt.Fprintf(&body, "%s%s %s\n", indent, icon, line)
					continue
				}
				fmt.Fprintf(&body, "%s%s%s\n", indent, strings.Repeat(" ", lipgloss.Width(icon)+1), line)
			}
		}
	}

	return frameStyle.Render(strings.TrimRight(body.String(), "\n")), true
}

func spinnerStepGroups(entries []*spinnerEntry) ([]spinnerStepGroup, bool) {
	groupIndexByStep := make(map[int]int)
	var groups []spinnerStepGroup
	currentIndex := -1
	hasStep := false

	for _, entry := range entries {
		index, total, label, ok := parseSpinnerStepMessage(entry.message)
		if ok {
			hasStep = true
			groupIndex, exists := groupIndexByStep[index]
			if !exists {
				groups = append(groups, spinnerStepGroup{
					index: index,
					total: total,
					title: spinnerStepTitle(index, total),
				})
				groupIndex = len(groups) - 1
				groupIndexByStep[index] = groupIndex
			}
			currentIndex = groupIndex
			groups[currentIndex].entries = append(groups[currentIndex].entries, spinnerStepEntry{
				entry: entry,
				label: label,
			})
			continue
		}

		if currentIndex < 0 {
			continue
		}
		label = cleanSpinnerStepLabel(entry.message)
		detail := strings.HasPrefix(label, "-") || strings.HasPrefix(label, "Using ")
		label = strings.TrimSpace(strings.TrimPrefix(label, "-"))
		groups[currentIndex].entries = append(groups[currentIndex].entries, spinnerStepEntry{
			entry:  entry,
			label:  label,
			detail: detail,
		})
	}

	return groups, hasStep
}

func spinnerEntriesContainSteps(entries []*spinnerEntry) bool {
	for _, entry := range entries {
		if _, _, _, ok := parseSpinnerStepMessage(entry.message); ok {
			return true
		}
	}
	return false
}

func spinnerDeploymentTarget(entries []*spinnerEntry) (provider, region string, ok bool) {
	for _, entry := range entries {
		if provider, region, ok = ParseDeploymentTarget(entry.message); ok {
			return provider, region, true
		}
	}
	return "", "", false
}

func parseSpinnerStepMessage(message string) (int, int, string, bool) {
	message = strings.TrimSpace(message)
	if !strings.HasPrefix(message, "Step ") {
		return 0, 0, "", false
	}

	rest := strings.TrimPrefix(message, "Step ")
	slash := strings.Index(rest, "/")
	colon := strings.Index(rest, ":")
	if slash < 1 || colon < 0 || slash > colon {
		return 0, 0, "", false
	}

	index, err := strconv.Atoi(strings.TrimSpace(rest[:slash]))
	if err != nil {
		return 0, 0, "", false
	}
	total, err := strconv.Atoi(strings.TrimSpace(rest[slash+1 : colon]))
	if err != nil {
		return 0, 0, "", false
	}

	label := strings.TrimSpace(rest[colon+1:])
	if label == "" {
		label = fmt.Sprintf("Step %d/%d", index, total)
	}
	return index, total, cleanSpinnerStepLabel(label), true
}

func spinnerStepTitle(index, total int) string {
	switch {
	case total == 2 && index == 1:
		return "Service creation"
	case total == 2 && index == 2:
		return "Instance deployment"
	default:
		return fmt.Sprintf("Step %d/%d", index, total)
	}
}

func spinnerStepProgress(entries []spinnerStepEntry) (complete, total int, hasRunning, hasError bool) {
	for _, item := range entries {
		if strings.TrimSpace(item.label) == "" {
			continue
		}
		total++
		switch item.entry.state {
		case spinnerComplete:
			complete++
		case spinnerError:
			hasError = true
		default:
			hasRunning = true
		}
	}
	return complete, total, hasRunning, hasError
}

func spinnerStepPercent(complete, total int, hasRunning bool) float64 {
	if total == 0 {
		return 0
	}
	weighted := float64(complete)
	if hasRunning {
		weighted += 0.5
	}
	percent := weighted / float64(total)
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}

func spinnerStepGroupSubmitted(group spinnerStepGroup) bool {
	if group.index != 2 {
		return false
	}
	for _, item := range group.entries {
		if item.entry.state != spinnerComplete {
			return false
		}
		label := strings.ToLower(item.label)
		if strings.Contains(label, "deploying a new instance: success") ||
			strings.Contains(label, "upgrading existing instance: success") ||
			strings.Contains(label, "instance creation submitted") ||
			strings.Contains(label, "instance upgrade submitted") {
			return true
		}
	}
	return false
}

func spinnerStepIcon(state int, spinnerView string, completeStyle, runningStyle, errorStyle lipgloss.Style) string {
	switch state {
	case spinnerComplete:
		return completeStyle.Render("✓")
	case spinnerError:
		return errorStyle.Render("✗")
	default:
		if spinnerView == "" {
			return runningStyle.Render("•")
		}
		return runningStyle.Render(spinnerView)
	}
}

func spinnerStepCategory(stepIndex int, label string, detail bool) string {
	label = cleanSpinnerStepLabel(label)
	lower := strings.ToLower(label)

	if stepIndex == 1 {
		switch {
		case strings.Contains(lower, "cloud provider") || strings.Contains(lower, "cloud account") || strings.Contains(lower, "aws account") || strings.Contains(lower, "gcp project") || strings.Contains(lower, "azure subscription"):
			return "Cloud accounts"
		case strings.Contains(lower, "service plan"):
			return "Service plan"
		default:
			return "Service"
		}
	}

	if stepIndex == 2 {
		switch {
		case strings.Contains(lower, "resource") || strings.Contains(lower, "cloud provider") || strings.Contains(lower, "region"):
			return "Target"
		case strings.Contains(lower, "deploying") || strings.Contains(lower, "upgrading") || strings.Contains(lower, "submitted"):
			return "Submit"
		default:
			return "Instance"
		}
	}

	if detail {
		return "Details"
	}
	return ""
}

func cleanSpinnerStepLabel(label string) string {
	label = strings.TrimSpace(label)
	label = strings.TrimPrefix(label, "📝")
	label = strings.TrimSpace(label)
	label = strings.TrimPrefix(label, "Note:")
	label = strings.TrimSpace(label)
	return label
}

func spinnerWrapText(text string, maxWidth int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	if maxWidth <= 0 || lipgloss.Width(text) <= maxWidth {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	lines := make([]string, 0, 2)
	current := ""
	for _, word := range words {
		if current == "" {
			if lipgloss.Width(word) <= maxWidth {
				current = word
				continue
			}
			chunks := spinnerHardWrapWord(word, maxWidth)
			lines = append(lines, chunks[:len(chunks)-1]...)
			current = chunks[len(chunks)-1]
			continue
		}

		next := current + " " + word
		if lipgloss.Width(next) <= maxWidth {
			current = next
			continue
		}

		lines = append(lines, current)
		if lipgloss.Width(word) <= maxWidth {
			current = word
			continue
		}
		chunks := spinnerHardWrapWord(word, maxWidth)
		lines = append(lines, chunks[:len(chunks)-1]...)
		current = chunks[len(chunks)-1]
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func spinnerHardWrapWord(word string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{word}
	}
	var chunks []string
	var current strings.Builder
	for _, r := range word {
		next := current.String() + string(r)
		if current.Len() > 0 && lipgloss.Width(next) > maxWidth {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func spinnerClamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
