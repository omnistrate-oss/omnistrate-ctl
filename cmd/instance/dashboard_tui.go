package instance

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/browser"
	"golang.org/x/term"
)

const (
	defaultDashboardWidth      = 120
	defaultDashboardHeight     = 32
	dashboardListMinWidth      = 34
	dashboardListPreferredWide = 42
	dashboardHelpHeight        = 2
	dashboardHeaderHeight      = 5
)

type dashboardFocus int

const (
	dashboardFocusSections dashboardFocus = iota
	dashboardFocusDetails
)

type dashboardNode struct {
	key         string
	title       string
	description string
	content     string
	copyText    string
	openURL     string
	expandable  bool
	expanded    bool
	children    []*dashboardNode
}

type dashboardItem struct {
	key         string
	parentKey   string
	title       string
	description string
	content     string
	copyText    string
	openURL     string
	level       int
	expandable  bool
	expanded    bool
}

func (i dashboardItem) Title() string       { return i.title }
func (i dashboardItem) Description() string { return i.description }
func (i dashboardItem) FilterValue() string {
	return i.title + " " + i.description
}

type dashboardDelegate struct {
	list.DefaultDelegate
}

func newDashboardDelegate() dashboardDelegate {
	delegate := dashboardDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	delegate.SetHeight(2)
	delegate.SetSpacing(0)
	return delegate
}

func (d dashboardDelegate) Render(writer io.Writer, model list.Model, index int, item list.Item) {
	dashboardItem, ok := item.(dashboardItem)
	if !ok {
		d.DefaultDelegate.Render(writer, model, index, item)
		return
	}

	isSelected := index == model.Index()
	titleStyle := lipgloss.NewStyle().Padding(0, 0, 0, 1)
	descriptionStyle := lipgloss.NewStyle().Padding(0, 0, 0, 1).Foreground(lipgloss.Color("245"))
	if isSelected {
		titleStyle = titleStyle.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
		descriptionStyle = descriptionStyle.Foreground(lipgloss.Color("252"))
	}

	indent := strings.Repeat("  ", dashboardItem.level)
	titlePrefix := ""
	switch {
	case dashboardItem.expandable && dashboardItem.expanded:
		titlePrefix = "▾ "
	case dashboardItem.expandable:
		titlePrefix = "▸ "
	case dashboardItem.level > 0:
		titlePrefix = "• "
	}

	title := indent + titlePrefix + dashboardItem.title
	description := dashboardItem.description
	if description != "" {
		description = indent + "  " + description
	}

	fmt.Fprintf(writer, "%s\n%s", titleStyle.Render(title), descriptionStyle.Render(description))
}

type dashboardModel struct {
	catalog          *dataaccess.DashboardCatalog
	rootNodes        []*dashboardNode
	items            []dashboardItem
	list             list.Model
	viewport         viewport.Model
	focus            dashboardFocus
	width            int
	height           int
	listPanelWidth   int
	detailPanelWidth int
	statusMessage    string
	linkOptions      []dashboardLinkOption
	linkIndex        int
}

type dashboardActionResultMsg struct {
	message string
	err     error
}

type dashboardLinkOption struct {
	label string
	url   string
}

var dashboardURLPattern = regexp.MustCompile(`https?://[^\s]+`)

func newDashboardModel(catalog *dataaccess.DashboardCatalog) dashboardModel {
	rootNodes := buildDashboardNodes(catalog)

	delegate := newDashboardDelegate()
	sectionList := list.New(nil, delegate, 0, 0)
	sectionList.Title = "Dashboards"
	sectionList.SetShowHelp(false)
	sectionList.SetShowFilter(false)
	sectionList.SetShowStatusBar(false)
	sectionList.SetFilteringEnabled(false)
	sectionList.DisableQuitKeybindings()
	sectionList.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Padding(0, 1)

	detailViewport := viewport.New(0, 0)

	model := dashboardModel{
		catalog:   catalog,
		rootNodes: rootNodes,
		list:      sectionList,
		viewport:  detailViewport,
		focus:     dashboardFocusSections,
	}
	model.setSize(defaultDashboardWidth, defaultDashboardHeight)
	model.rebuildVisibleItems("")
	return model
}

func (m dashboardModel) Init() tea.Cmd {
	return nil
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case dashboardActionResultMsg:
		if msg.err != nil {
			m.statusMessage = msg.err.Error()
		} else {
			m.statusMessage = msg.message
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "Q", "esc", "ctrl+c"))):
			if m.linkPickerActive() && !key.Matches(msg, key.NewBinding(key.WithKeys("q", "Q", "ctrl+c"))) {
				m.clearLinkPicker()
				return m, nil
			}
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			if m.focus == dashboardFocusSections {
				m.focus = dashboardFocusDetails
			} else {
				m.focus = dashboardFocusSections
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("c", "C", "y", "Y"))):
			if m.linkPickerActive() {
				return m.copySelectedLink()
			}
			return m.copySelectedValue()
		case key.Matches(msg, key.NewBinding(key.WithKeys("o", "O"))):
			if m.linkPickerActive() {
				return m.openSelectedLink()
			}
			return m.openSelectedURL()
		}

		if m.linkPickerActive() {
			switch msg.String() {
			case "enter":
				return m.openSelectedLink()
			}
			return m.handleLinkPickerKey(msg), nil
		}

		if m.focus == dashboardFocusDetails {
			return m.handleViewportKey(msg), nil
		}

		switch msg.String() {
		case "enter", " ", "right", "l":
			if m.toggleSelectedExpandable() {
				return m, nil
			}
		case "left", "h":
			if m.collapseSelectedItem() {
				return m, nil
			}
		}
	}

	previousIndex := m.list.Index()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.list.Index() != previousIndex {
		m.clearLinkPicker()
		m.syncViewportContent()
	}
	return m, cmd
}

func (m dashboardModel) handleViewportKey(msg tea.KeyMsg) dashboardModel {
	switch msg.String() {
	case "down", "j":
		m.viewport.ScrollDown(1)
	case "up", "k":
		m.viewport.ScrollUp(1)
	case "pgdown", "f", " ":
		m.viewport.PageDown()
	case "pgup", "b":
		m.viewport.PageUp()
	case "g", "home":
		m.viewport.GotoTop()
	case "G", "end":
		m.viewport.GotoBottom()
	}
	return m
}

func (m dashboardModel) handleLinkPickerKey(msg tea.KeyMsg) dashboardModel {
	if len(m.linkOptions) == 0 {
		return m
	}

	switch msg.String() {
	case "down", "j":
		m.linkIndex = min(m.linkIndex+1, len(m.linkOptions)-1)
	case "up", "k":
		m.linkIndex = max(m.linkIndex-1, 0)
	}

	return m
}

func (m *dashboardModel) setSize(width, height int) {
	if width <= 0 {
		width = defaultDashboardWidth
	}
	if height <= 0 {
		height = defaultDashboardHeight
	}

	m.width = width
	m.height = height

	listPanelWidth := min(max(width/3, dashboardListMinWidth), dashboardListPreferredWide)
	bodyHeight := max(height-dashboardHeaderHeight-dashboardHelpHeight, 12)
	detailPanelWidth := max(width-listPanelWidth-1, 40)
	panelFrameWidth := dashboardPanelStyle(lipgloss.Color("240")).GetHorizontalFrameSize()

	m.listPanelWidth = listPanelWidth
	m.detailPanelWidth = detailPanelWidth
	m.list.SetWidth(max(listPanelWidth-panelFrameWidth, 10))
	m.list.SetHeight(bodyHeight)
	m.viewport.Width = max(detailPanelWidth-panelFrameWidth, 10)
	m.viewport.Height = bodyHeight
	m.syncViewportContent()
}

func (m *dashboardModel) selectedItem() *dashboardItem {
	index := m.list.Index()
	if index < 0 || index >= len(m.items) {
		return nil
	}
	return &m.items[index]
}

func (m *dashboardModel) toggleSelectedExpandable() bool {
	selected := m.selectedItem()
	if selected == nil || !selected.expandable {
		return false
	}
	if toggleDashboardNodeExpanded(m.rootNodes, selected.key) {
		m.rebuildVisibleItems(selected.key)
		return true
	}
	return false
}

func (m *dashboardModel) collapseSelectedItem() bool {
	selected := m.selectedItem()
	if selected == nil {
		return false
	}

	if selected.expandable && selected.expanded {
		if setDashboardNodeExpanded(m.rootNodes, selected.key, false) {
			m.rebuildVisibleItems(selected.key)
			return true
		}
		return false
	}

	if selected.parentKey == "" {
		return false
	}

	if setDashboardNodeExpanded(m.rootNodes, selected.parentKey, false) {
		m.rebuildVisibleItems(selected.parentKey)
		return true
	}
	return false
}

func (m *dashboardModel) rebuildVisibleItems(selectedKey string) {
	if selectedKey == "" {
		if current := m.selectedItem(); current != nil {
			selectedKey = current.key
		}
	}

	m.items = flattenDashboardNodes(m.rootNodes, 0, "")
	listItems := make([]list.Item, len(m.items))
	selectedIndex := 0
	for index, item := range m.items {
		listItems[index] = item
		if item.key == selectedKey {
			selectedIndex = index
		}
	}

	_ = m.list.SetItems(listItems)
	if len(listItems) > 0 {
		m.list.Select(selectedIndex)
	}
	m.syncViewportContent()
}

func (m *dashboardModel) syncViewportContent() {
	selected := m.selectedItem()
	if selected == nil {
		m.viewport.SetContent("No dashboard details are available.")
		return
	}
	m.viewport.SetContent(renderDashboardDetailContent(selected.content, m.viewport.Width))
	m.viewport.GotoTop()
}

func (m dashboardModel) copySelectedValue() (tea.Model, tea.Cmd) {
	text := m.selectedCopyText()
	if text == "" {
		m.statusMessage = "Nothing actionable to copy from this section"
		return m, nil
	}

	return m, copyDashboardTextCmd(text, "Copied selected value to clipboard")
}

func (m dashboardModel) openSelectedURL() (tea.Model, tea.Cmd) {
	links := m.selectedLinkOptions()
	switch len(links) {
	case 0:
		m.statusMessage = "No URL available for the selected section"
		return m, nil
	case 1:
		m.clearLinkPicker()
		return m, openDashboardURLCmd(links[0].url)
	default:
		m.linkOptions = links
		m.linkIndex = 0
		m.statusMessage = "Multiple URLs found. Choose one with ↑/↓ and press enter or o"
		return m, nil
	}
}

func (m dashboardModel) selectedCopyText() string {
	selected := m.selectedItem()
	if selected == nil {
		return ""
	}

	return selectedDashboardCopyText(selected)
}

func (m dashboardModel) selectedLinkOptions() []dashboardLinkOption {
	selected := m.selectedItem()
	if selected == nil {
		return nil
	}

	return selectedDashboardLinkOptions(selected)
}

func (m dashboardModel) linkPickerActive() bool {
	return len(m.linkOptions) > 0
}

func (m *dashboardModel) clearLinkPicker() {
	m.linkOptions = nil
	m.linkIndex = 0
}

func (m dashboardModel) currentLinkOption() *dashboardLinkOption {
	if len(m.linkOptions) == 0 || m.linkIndex < 0 || m.linkIndex >= len(m.linkOptions) {
		return nil
	}

	return &m.linkOptions[m.linkIndex]
}

func (m dashboardModel) copySelectedLink() (tea.Model, tea.Cmd) {
	link := m.currentLinkOption()
	if link == nil {
		m.statusMessage = "No URL selected"
		return m, nil
	}

	m.clearLinkPicker()
	return m, copyDashboardTextCmd(link.url, "Copied selected URL to clipboard")
}

func (m dashboardModel) openSelectedLink() (tea.Model, tea.Cmd) {
	link := m.currentLinkOption()
	if link == nil {
		m.statusMessage = "No URL selected"
		return m, nil
	}

	m.clearLinkPicker()
	return m, openDashboardURLCmd(link.url)
}

func (m dashboardModel) View() string {
	header := renderDashboardHeader(m.catalog)

	listPanelStyle := dashboardPanelStyle(m.focusBorderColor(dashboardFocusSections))
	detailPanelStyle := dashboardPanelStyle(m.focusBorderColor(dashboardFocusDetails))

	sectionsPanel := listPanelStyle.Width(m.listPanelWidth).Render(m.list.View())
	detailsPanel := detailPanelStyle.Width(m.detailPanelWidth).Render(m.viewport.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, sectionsPanel, " ", detailsPanel)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
		"tab: switch panel  ↑/↓: navigate  enter: toggle accordion  c: copy value  o: open URL  j/k or pgup/pgdn: scroll details  q: quit",
	)
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render(m.statusLine())
	parts := []string{header, body, help, status}
	if picker := m.renderLinkPicker(); picker != "" {
		parts = append(parts, picker)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m dashboardModel) focusBorderColor(focus dashboardFocus) lipgloss.Color {
	if m.focus == focus {
		return lipgloss.Color("117")
	}
	return lipgloss.Color("240")
}

func dashboardPanelStyle(borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}

func printDashboardTUI(catalog *dataaccess.DashboardCatalog) error {
	if catalog == nil {
		return fmt.Errorf("dashboard details are required")
	}

	snapshot := renderDashboardSnapshot(catalog)
	utils.LastPrintedString = snapshot

	if !isDashboardInteractive() {
		fmt.Println(snapshot)
		return nil
	}

	program := tea.NewProgram(newDashboardModel(catalog), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to launch dashboard TUI: %w", err)
	}

	return nil
}

func renderDashboardSnapshot(catalog *dataaccess.DashboardCatalog) string {
	model := newDashboardModel(catalog)
	return strings.TrimRight(model.View(), "\n")
}

func isDashboardInteractive() bool {
	return dashboardFileIsTerminal(os.Stdout) && dashboardFileIsTerminal(os.Stdin)
}

func dashboardFileIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}

	fd := file.Fd()
	if fd > uintptr(^uint(0)>>1) {
		return false
	}

	return term.IsTerminal(int(fd))
}

func buildDashboardNodes(catalog *dataaccess.DashboardCatalog) []*dashboardNode {
	if catalog == nil || len(catalog.Features) == 0 {
		return []*dashboardNode{{
			key:         "overview",
			title:       "Overview",
			description: "no metrics dashboards available",
			content:     "No dashboard metadata is available for this instance.",
		}}
	}

	nodes := make([]*dashboardNode, 0, len(catalog.Features))
	for _, feature := range catalog.Features {
		children := make([]*dashboardNode, 0, len(feature.Dashboards))
		for _, dashboard := range feature.Dashboards {
			children = append(children, buildDashboardChildNode(feature, dashboard))
		}

		descriptionParts := []string{fmt.Sprintf("%d dashboard(s)", len(feature.Dashboards))}
		if host := dashboardHost(feature.GrafanaEndpoint); host != "" {
			descriptionParts = append(descriptionParts, host)
		}

		node := &dashboardNode{
			key:         feature.Key,
			title:       dashboardFeatureDisplayName(feature),
			description: strings.Join(descriptionParts, "  |  "),
			content:     buildDashboardFeatureContent(feature),
			expandable:  len(children) > 0,
			expanded:    true,
			children:    children,
		}
		nodes = append(nodes, node)
	}

	return nodes
}

func buildDashboardChildNode(feature dataaccess.DashboardFeatureInfo, dashboard dataaccess.DashboardRef) *dashboardNode {
	lines := []string{
		fmt.Sprintf("%s Dashboard", dashboardFeatureDisplayName(feature)),
		"",
		"Overview:",
		fmt.Sprintf("Name: %s", dashboardDisplayValue(dashboard.Name)),
		fmt.Sprintf("Description: %s", dashboardDisplayValue(dashboard.Description)),
		"",
		"Access:",
		fmt.Sprintf("Dashboard URL: %s", dashboardDisplayValue(dashboard.URL)),
	}

	if feature.GrafanaUIUsername != "" || feature.GrafanaUIPassword != "" {
		lines = append(lines,
			"",
			"Grafana UI:",
			fmt.Sprintf("Username: %s", dashboardDisplayValue(feature.GrafanaUIUsername)),
			fmt.Sprintf("Password: %s", dashboardDisplayValue(feature.GrafanaUIPassword)),
		)
		if feature.GrafanaUILoginScope != "" {
			lines = append(lines, fmt.Sprintf("Scope: %s", dashboardDisplayValue(feature.GrafanaUILoginScope)))
		}
	}

	if feature.ServiceAccountName != "" || feature.ServiceAccountToken != "" {
		lines = append(lines,
			"",
			"Grafana API:",
			fmt.Sprintf("Service account: %s", dashboardDisplayValue(feature.ServiceAccountName)),
			fmt.Sprintf("Service account token: %s", dashboardDisplayValue(feature.ServiceAccountToken)),
		)
	}

	description := strings.TrimSpace(dashboard.Description)
	if description == "" {
		description = dashboardHost(dashboard.URL)
	}

	return &dashboardNode{
		key:         fmt.Sprintf("%s/%s", feature.Key, dashboard.Name),
		title:       dashboard.Name,
		description: description,
		content:     strings.Join(lines, "\n"),
		openURL:     strings.TrimSpace(dashboard.URL),
	}
}

func buildDashboardFeatureContent(feature dataaccess.DashboardFeatureInfo) string {
	lines := []string{
		dashboardFeatureDisplayName(feature),
	}

	if feature.GrafanaUIUsername != "" || feature.GrafanaUIPassword != "" {
		lines = append(lines,
			"",
			"Grafana UI:",
			fmt.Sprintf("Username: %s", dashboardDisplayValue(feature.GrafanaUIUsername)),
			fmt.Sprintf("Password: %s", dashboardDisplayValue(feature.GrafanaUIPassword)),
		)
		if feature.GrafanaUILoginScope != "" {
			lines = append(lines, fmt.Sprintf("Scope: %s", dashboardDisplayValue(feature.GrafanaUILoginScope)))
		}
	}

	if feature.ServiceAccountName != "" || feature.ServiceAccountToken != "" {
		lines = append(lines,
			"",
			"Grafana API:",
			fmt.Sprintf("Service account: %s", dashboardDisplayValue(feature.ServiceAccountName)),
			fmt.Sprintf("Service account token: %s", dashboardDisplayValue(feature.ServiceAccountToken)),
		)
	}

	if len(feature.Dashboards) > 0 {
		lines = append(lines, "", "Published dashboards:")
		for _, dashboard := range feature.Dashboards {
			if dashboard.Description != "" {
				lines = append(lines, fmt.Sprintf("- %s: %s", dashboard.Name, dashboard.Description))
			} else {
				lines = append(lines, fmt.Sprintf("- %s", dashboard.Name))
			}
			if dashboard.URL != "" {
				lines = append(lines, fmt.Sprintf("  %s", dashboard.URL))
			}
		}
	}

	if len(feature.DashboardDefinitions) > 0 {
		lines = append(lines, "", "Dashboard templates:")
		for _, dashboard := range feature.DashboardDefinitions {
			if dashboard.Title != "" {
				lines = append(lines, fmt.Sprintf("- %s/%s: %s", dashboard.Source, dashboard.Name, dashboard.Title))
				continue
			}
			lines = append(lines, fmt.Sprintf("- %s/%s", dashboard.Source, dashboard.Name))
		}
	}

	return strings.Join(lines, "\n")
}

func renderDashboardDetailContent(content string, width int) string {
	if width <= 0 {
		width = defaultDashboardWidth
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	lines := strings.Split(content, "\n")
	rendered := make([]string, 0, len(lines))
	titleRendered := false

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, " ")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			rendered = append(rendered, "")
			continue
		}

		if !titleRendered {
			rendered = append(rendered, titleStyle.Render(trimmed))
			titleRendered = true
			continue
		}

		if strings.HasSuffix(trimmed, ":") && !strings.Contains(strings.TrimSuffix(trimmed, ":"), ": ") {
			rendered = append(rendered, sectionStyle.Render(strings.TrimSuffix(trimmed, ":")))
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			rendered = append(rendered, renderDashboardBullet(trimmed, width, keyStyle, valueStyle)...)
			continue
		}

		if strings.Contains(trimmed, ": ") {
			label, value, _ := strings.Cut(trimmed, ": ")
			rendered = append(rendered, renderDashboardField(label, value, width, keyStyle, valueStyle, urlStyle)...)
			continue
		}

		if dashboardURLPattern.MatchString(trimmed) {
			rendered = append(rendered, renderDashboardWrappedValue(trimmed, width, urlStyle)...)
			continue
		}

		style := textStyle
		if strings.Contains(strings.ToLower(trimmed), "not available") {
			style = mutedStyle
		}
		rendered = append(rendered, renderDashboardWrappedValue(trimmed, width, style)...)
	}

	return strings.Join(rendered, "\n")
}

func renderDashboardField(
	label string,
	value string,
	width int,
	keyStyle lipgloss.Style,
	valueStyle lipgloss.Style,
	urlStyle lipgloss.Style,
) []string {
	prefix := label + ": "
	valueWidth := max(width-lipgloss.Width(prefix), 16)
	valueLines := wrapDashboardLine(value, valueWidth)
	if len(valueLines) == 0 {
		valueLines = []string{""}
	}

	valueRenderer := valueStyle
	if dashboardURLPattern.MatchString(value) {
		valueRenderer = urlStyle
	}

	rendered := make([]string, 0, len(valueLines))
	continuationIndent := strings.Repeat(" ", lipgloss.Width(prefix))
	for index, valueLine := range valueLines {
		if index == 0 {
			rendered = append(rendered, keyStyle.Render(prefix)+valueRenderer.Render(valueLine))
			continue
		}
		rendered = append(rendered, continuationIndent+valueRenderer.Render(valueLine))
	}

	return rendered
}

func renderDashboardBullet(
	line string,
	width int,
	keyStyle lipgloss.Style,
	valueStyle lipgloss.Style,
) []string {
	item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
	if item == "" {
		return []string{line}
	}

	if label, value, ok := strings.Cut(item, ": "); ok {
		prefix := "• " + label + ": "
		valueWidth := max(width-lipgloss.Width(prefix), 16)
		valueLines := wrapDashboardLine(value, valueWidth)
		if len(valueLines) == 0 {
			valueLines = []string{""}
		}

		rendered := make([]string, 0, len(valueLines))
		continuationIndent := strings.Repeat(" ", lipgloss.Width(prefix))
		for index, valueLine := range valueLines {
			if index == 0 {
				rendered = append(rendered, keyStyle.Render("• "+label+": ")+valueStyle.Render(valueLine))
				continue
			}
			rendered = append(rendered, continuationIndent+valueStyle.Render(valueLine))
		}
		return rendered
	}

	wrapped := wrapDashboardLine("• "+item, width)
	rendered := make([]string, 0, len(wrapped))
	for _, wrappedLine := range wrapped {
		rendered = append(rendered, keyStyle.Render(wrappedLine))
	}
	return rendered
}

func renderDashboardWrappedValue(value string, width int, style lipgloss.Style) []string {
	wrapped := wrapDashboardLine(value, width)
	rendered := make([]string, 0, len(wrapped))
	for _, wrappedLine := range wrapped {
		rendered = append(rendered, style.Render(wrappedLine))
	}
	return rendered
}

func renderDashboardHeader(catalog *dataaccess.DashboardCatalog) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(
		fmt.Sprintf("Instance Dashboard · %s", dashboardDisplayValue(catalog.InstanceID)),
	)

	summary := []string{fmt.Sprintf("%d metrics view(s)", len(catalog.Features))}
	for _, feature := range catalog.Features {
		summary = append(summary, dashboardFeatureDisplayName(feature))
	}
	metadata := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(strings.Join(summary, "  |  "))

	checks := []string{
		renderDashboardCheck("Metrics dashboards available", len(catalog.Features) > 0),
	}
	if catalog.PreferredFeatureKey != "" {
		checks = append(checks, renderDashboardCheck(fmt.Sprintf("Preferred view is %s", catalog.PreferredFeatureKey), true))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		metadata,
		strings.Join(checks, "  "),
	)
}

func flattenDashboardNodes(nodes []*dashboardNode, level int, parentKey string) []dashboardItem {
	items := make([]dashboardItem, 0)
	for _, node := range nodes {
		if node == nil {
			continue
		}
		items = append(items, dashboardItem{
			key:         node.key,
			parentKey:   parentKey,
			title:       node.title,
			description: node.description,
			content:     node.content,
			copyText:    node.copyText,
			openURL:     node.openURL,
			level:       level,
			expandable:  node.expandable,
			expanded:    node.expanded,
		})
		if node.expandable && node.expanded {
			items = append(items, flattenDashboardNodes(node.children, level+1, node.key)...)
		}
	}
	return items
}

func toggleDashboardNodeExpanded(nodes []*dashboardNode, key string) bool {
	node := findDashboardNode(nodes, key)
	if node == nil || !node.expandable {
		return false
	}
	node.expanded = !node.expanded
	return true
}

func setDashboardNodeExpanded(nodes []*dashboardNode, key string, expanded bool) bool {
	node := findDashboardNode(nodes, key)
	if node == nil || !node.expandable {
		return false
	}
	node.expanded = expanded
	return true
}

func findDashboardNode(nodes []*dashboardNode, key string) *dashboardNode {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.key == key {
			return node
		}
		if child := findDashboardNode(node.children, key); child != nil {
			return child
		}
	}
	return nil
}

func renderDashboardCheck(label string, ok bool) string {
	icon := "○"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	if ok {
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	}
	return style.Render(fmt.Sprintf("%s %s", icon, label))
}

func dashboardFeatureDisplayName(feature dataaccess.DashboardFeatureInfo) string {
	label := strings.TrimSpace(feature.Label)
	if label == "" {
		label = feature.Key
	}
	if strings.TrimSpace(feature.Key) == "" {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, feature.Key)
}

func dashboardDisplayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "not available"
	}
	return value
}

func dashboardHost(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return parsed.Host
}

func wrapDashboardContent(content string, width int) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapDashboardLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapDashboardLine(line string, width int) []string {
	if width <= 0 || line == "" || lipgloss.Width(line) <= width {
		return []string{line}
	}

	runes := []rune(line)
	wrapped := make([]string, 0, 2)
	for len(runes) > 0 {
		currentWidth := 0
		splitIndex := 0
		lastSoftBreak := 0

		for index, r := range runes {
			runeWidth := lipgloss.Width(string(r))
			if currentWidth+runeWidth > width {
				break
			}
			currentWidth += runeWidth
			splitIndex = index + 1
			if isDashboardSoftBreak(r) {
				lastSoftBreak = splitIndex
			}
		}

		if splitIndex == 0 {
			splitIndex = 1
		}
		if splitIndex < len(runes) && lastSoftBreak > 0 {
			splitIndex = lastSoftBreak
		}

		wrapped = append(wrapped, string(runes[:splitIndex]))
		runes = runes[splitIndex:]
	}

	return wrapped
}

func isDashboardSoftBreak(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("/:?&=_-.,", r)
}

func (m dashboardModel) statusLine() string {
	if m.linkPickerActive() {
		return "Select a URL with ↑/↓. Press enter or o to open, c to copy, esc to close."
	}
	if strings.TrimSpace(m.statusMessage) != "" {
		return m.statusMessage
	}
	return "Use c to copy the selected value. Use o to open a selected URL."
}

func selectedDashboardCopyText(item *dashboardItem) string {
	if item == nil {
		return ""
	}

	if text := strings.TrimSpace(item.copyText); text != "" {
		return text
	}

	return strings.TrimSpace(item.content)
}

func selectedDashboardLinkOptions(item *dashboardItem) []dashboardLinkOption {
	if item == nil {
		return nil
	}

	options := make([]dashboardLinkOption, 0)
	seen := make(map[string]struct{})
	addOption := func(label, rawURL string) {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}
		if _, exists := seen[rawURL]; exists {
			return
		}
		seen[rawURL] = struct{}{}
		label = strings.TrimSpace(label)
		if label == "" {
			label = rawURL
		}
		options = append(options, dashboardLinkOption{label: label, url: rawURL})
	}

	if rawURL := strings.TrimSpace(item.openURL); rawURL != "" {
		addOption(item.title, rawURL)
		return options
	}

	for _, line := range strings.Split(item.content, "\n") {
		matches := dashboardURLPattern.FindAllString(strings.TrimSpace(line), -1)
		for _, rawURL := range matches {
			label := strings.TrimSpace(strings.Replace(line, rawURL, "", 1))
			label = strings.TrimSpace(strings.TrimSuffix(label, ":"))
			addOption(label, rawURL)
		}
	}

	return options
}

func (m dashboardModel) renderLinkPicker() string {
	if !m.linkPickerActive() {
		return ""
	}

	lines := []string{"URL Choices"}
	for index, option := range m.linkOptions {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		if index == m.linkIndex {
			prefix = "▸ "
			style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
		}
		lines = append(lines, style.Render(prefix+option.label))
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render("  "+wrapDashboardContent(option.url, max(m.width-8, 32))))
	}

	return dashboardPanelStyle(lipgloss.Color("117")).Render(strings.Join(lines, "\n"))
}

func copyDashboardTextCmd(text, successMessage string) tea.Cmd {
	return func() tea.Msg {
		if err := copyToClipboard(text); err != nil {
			return dashboardActionResultMsg{err: fmt.Errorf("clipboard copy failed: %w", err)}
		}
		return dashboardActionResultMsg{message: successMessage}
	}
}

func openDashboardURLCmd(rawURL string) tea.Cmd {
	return func() tea.Msg {
		if err := browser.OpenURL(rawURL); err != nil {
			return dashboardActionResultMsg{err: fmt.Errorf("failed to open URL: %w", err)}
		}
		return dashboardActionResultMsg{message: "Opened selected URL in browser"}
	}
}
