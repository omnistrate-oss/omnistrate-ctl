package deploymentcell

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/syntax"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/tui"
)

// amenityDetailModel is the full-screen, scrollable detail view for a single
// amenity. The active tab's content is wrapped to the viewport width and, for
// chart-value tabs, rendered as syntax-highlighted JSON.
type amenityDetailModel struct {
	data         deploymentCellDebugData
	status       deploymentCellAmenityStatus
	views        []string
	viewIdx      int
	scroll       int
	width        int
	height       int
	clipboardMsg string
}

func newAmenityDetailModel(data deploymentCellDebugData, status deploymentCellAmenityStatus, width, height int) amenityDetailModel {
	return amenityDetailModel{
		data:   data,
		status: status,
		views:  availableViewsFor(status),
		width:  width,
		height: height,
	}
}

// isManaged reports whether an amenity is managed by Omnistrate. Managed
// amenities expose no debug content — only a badge.
func isManaged(status deploymentCellAmenityStatus) bool {
	return status.IsManaged != nil && *status.IsManaged
}

// availableViewsFor returns the views available for an amenity. Managed
// amenities have none (badge only). Custom amenities show a single rendered
// view: rendered values for Helm, rendered manifest for Kubernetes manifests.
func availableViewsFor(status deploymentCellAmenityStatus) []string {
	if isManaged(status) {
		return nil
	}
	switch status.Type {
	case amenityTypeHelm:
		return []string{"rendered values"}
	case amenityTypeKubernetesManifest:
		return []string{"rendered manifest"}
	default:
		return []string{"status"}
	}
}

func (m amenityDetailModel) currentView() string {
	if m.viewIdx < 0 || m.viewIdx >= len(m.views) {
		return ""
	}
	return m.views[m.viewIdx]
}

func (m amenityDetailModel) Update(msg tea.Msg) (amenityDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		(&m).clampScroll()
	case clipboardResultMsg:
		if msg.err != nil {
			m.clipboardMsg = "copy failed"
		} else {
			m.clipboardMsg = "copied to clipboard"
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return backToListMsg{} }
		case "tab", "right", "l":
			if len(m.views) > 0 {
				m.viewIdx = (m.viewIdx + 1) % len(m.views)
			}
			m.scroll = 0
			m.clipboardMsg = ""
		case "shift+tab", "left", "h":
			if len(m.views) > 0 {
				m.viewIdx = (m.viewIdx - 1 + len(m.views)) % len(m.views)
			}
			m.scroll = 0
			m.clipboardMsg = ""
		case "up", "k":
			m.scroll--
			(&m).clampScroll()
		case "down", "j":
			m.scroll++
			(&m).clampScroll()
		case "pgup", "b":
			m.scroll -= m.bodyHeight()
			(&m).clampScroll()
		case "pgdown", "pgdn", " ":
			m.scroll += m.bodyHeight()
			(&m).clampScroll()
		case "home", "g":
			m.scroll = 0
		case "end", "G":
			m.scroll = 1 << 30
			(&m).clampScroll()
		case "y":
			return m, copyToClipboardCmd(m.rawContent())
		}
	}
	return m, nil
}

func (m amenityDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if isManaged(m.status) {
		return m.renderManagedView()
	}

	vlines := m.visualLines()
	scroll := clampInt(m.scroll, 0, maxScroll(len(vlines), m.bodyHeight()))

	header := m.renderHeader()
	tabBar := m.renderTabBar()
	body := m.renderBody(vlines, scroll)
	footer := m.renderFooter(scroll, len(vlines))

	return lipgloss.JoinVertical(lipgloss.Left, header, tabBar, body, footer)
}

// renderManagedView shows only a green "Omnistrate Managed" badge for amenities
// managed by Omnistrate — no configuration is exposed.
func (m amenityDetailModel) renderManagedView() string {
	title := titleStyle.Render(fmt.Sprintf("Amenity · %s · %s", m.status.Name, m.status.Type))
	footer := mutedStyle.Render("esc: back   q: quit")

	badge := managedBadgeStyle.Render("✓ Omnistrate Managed")
	note := mutedStyle.Render("Managed by Omnistrate — configuration is not exposed.")
	center := lipgloss.JoinVertical(lipgloss.Center, badge, "", note)

	fillH := m.height - 2 // title + footer
	if fillH < 1 {
		fillH = 1
	}
	body := lipgloss.Place(m.width, fillH, lipgloss.Center, lipgloss.Center, center)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, footer)
}

func (m amenityDetailModel) renderHeader() string {
	title := titleStyle.Render(fmt.Sprintf("Amenity · %s · %s · %s", m.status.Name, m.status.Type, m.status.DesiredStatus))

	var metaLine string
	if m.status.LastError != nil && *m.status.LastError != "" {
		metaLine = errorStyle.Render(padOrTruncate("error: "+collapseWhitespace(*m.status.LastError), max(m.width-1, 10)))
	} else {
		var meta []string
		if m.status.Source != nil && *m.status.Source != "" {
			meta = append(meta, "source="+*m.status.Source)
		}
		if m.status.SourceEnvironmentType != nil && *m.status.SourceEnvironmentType != "" {
			meta = append(meta, "env="+*m.status.SourceEnvironmentType)
		}
		if m.status.SourceCloudProviderID != nil && *m.status.SourceCloudProviderID != "" {
			meta = append(meta, "cloud="+*m.status.SourceCloudProviderID)
		}
		if m.status.Generation > 0 {
			meta = append(meta, fmt.Sprintf("generation=%d", m.status.Generation))
		}
		if wf := workflowLine(m.status); wf != "" {
			meta = append(meta, wf)
		}
		metaLine = mutedStyle.Render(strings.Join(meta, "   "))
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, metaLine)
}

func (m amenityDetailModel) renderTabBar() string {
	if len(m.views) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m.views))
	for i, v := range m.views {
		label := " " + displayName(v) + " "
		if i == m.viewIdx {
			parts = append(parts, activeTabStyle.Render(label))
		} else {
			parts = append(parts, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(parts, mutedStyle.Render(" "))
}

func (m amenityDetailModel) renderBody(vlines []tui.VisualLine, scroll int) string {
	bodyH := m.bodyHeight()
	highlight := isJSONView(m.currentView())

	end := scroll + bodyH
	if end > len(vlines) {
		end = len(vlines)
	}

	var b strings.Builder
	for i := scroll; i < end; i++ {
		text := vlines[i].Text
		if highlight {
			text = syntax.HighlightLine(text, "values.json")
		}
		b.WriteString(text)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	winWidth := m.width - 2
	if winWidth < 1 {
		winWidth = 1
	}
	return detailWindowStyle.Width(winWidth).Height(bodyH).Render(b.String())
}

func (m amenityDetailModel) renderFooter(scroll, total int) string {
	help := "↑↓/pgup/pgdn: scroll   y: copy   esc: back   q: quit"
	if len(m.views) > 1 {
		help = "tab: switch view   " + help
	}
	text := help + "   " + positionLabel(scroll, m.bodyHeight(), total)
	if m.clipboardMsg != "" {
		text = clipStyle.Render(m.clipboardMsg) + "   " + text
	}
	return mutedStyle.Render(text)
}

// bodyHeight is the number of content rows visible at once: terminal height
// minus header (2), tab bar (1), body border (2) and footer (1).
func (m amenityDetailModel) bodyHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

// contentWidth is the wrap width for body text: terminal width minus the body
// border (2) and padding (2).
func (m amenityDetailModel) contentWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m amenityDetailModel) visualLines() []tui.VisualLine {
	return tui.ExpandLinesToVisual(strings.Split(m.rawContent(), "\n"), m.contentWidth())
}

func (m *amenityDetailModel) clampScroll() {
	m.scroll = clampInt(m.scroll, 0, maxScroll(len(m.visualLines()), m.bodyHeight()))
}

func (m amenityDetailModel) rawContent() string {
	return rawContentForView(m.data, m.status, m.currentView())
}

// rawContentForView returns the unstyled text shown for a given amenity view.
func rawContentForView(data deploymentCellDebugData, status deploymentCellAmenityStatus, view string) string {
	switch view {
	case "rendered values":
		return valuesContent(data, status.Name, artifactHelmValuesRendered)
	case "rendered manifest":
		payload, _ := artifactPayload(data, status.Name, artifactKubernetesManifestRendered)
		return payload
	default:
		return "No detail available."
	}
}

// valuesContent returns chart values converted to pretty JSON when possible, or
// the raw/placeholder payload otherwise.
func valuesContent(data deploymentCellDebugData, amenityName, artifactKind string) string {
	payload, found := artifactPayload(data, amenityName, artifactKind)
	if !found {
		return payload
	}
	if pretty, ok := toPrettyJSON(payload); ok {
		return pretty
	}
	return payload
}

// isJSONView reports whether a tab's content should be JSON syntax highlighted.
func isJSONView(view string) bool {
	return view == "rendered values" || view == "template values"
}

// maxScroll is the largest valid scroll offset for total visual lines shown in
// a viewport of bodyH rows.
func maxScroll(total, bodyH int) int {
	if total <= bodyH {
		return 0
	}
	return total - bodyH
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// positionLabel describes the scroll position as top/end/all or a percentage.
func positionLabel(scroll, bodyH, total int) string {
	if total <= bodyH {
		return "[all]"
	}
	if scroll <= 0 {
		return "[top]"
	}
	if scroll+bodyH >= total {
		return "[end]"
	}
	return fmt.Sprintf("[%d%%]", scroll*100/maxScroll(total, bodyH))
}

// displayName title-cases a lowercase view key for tab labels.
func displayName(view string) string {
	words := strings.Fields(view)
	for i, w := range words {
		r := []rune(w)
		if len(r) > 0 {
			r[0] = unicode.ToUpper(r[0])
		}
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// collapseWhitespace flattens newlines/tabs into single spaces so a multi-line
// error renders on the single header meta line.
func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// clipboardResultMsg is delivered after a clipboard copy attempt.
type clipboardResultMsg struct{ err error }

func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return clipboardResultMsg{err: tui.CopyToClipboard(text)}
	}
}
