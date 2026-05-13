package serviceplan

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"golang.org/x/term"
)

const (
	defaultServicePlanBrowserWidth       = 120
	defaultServicePlanBrowserHeight      = 32
	servicePlanBrowserListMinWidth       = 34
	servicePlanBrowserListPreferredWidth = 42
	servicePlanBrowserHelpHeight         = 2
	servicePlanBrowserHeaderHeight       = 5
	servicePlanBrowserTabsHeight         = 3

	servicePlanBrowserFocusLeft servicePlanBrowserFocus = iota
	servicePlanBrowserFocusDetails

	servicePlanBrowserModalDeployments   servicePlanBrowserModalKind = "deployments"
	servicePlanBrowserModalSubscriptions servicePlanBrowserModalKind = "subscriptions"
	servicePlanBrowserModalUsers         servicePlanBrowserModalKind = "users"
)

type servicePlanBrowserFocus int

type servicePlanBrowserModalKind string

type servicePlanBrowserCatalog struct {
	Services []servicePlanBrowserService
}

type servicePlanBrowserService struct {
	ID    string
	Name  string
	Plans []servicePlanBrowserPlan
}

type servicePlanBrowserPlan struct {
	Name         string
	ServiceID    string
	ServiceName  string
	Environments []servicePlanBrowserEnvironment
}

type servicePlanHostingBadge struct {
	Label string
	Color lipgloss.Color
}

type servicePlanBrowserEnvironment struct {
	ID             string
	Name           string
	PlanID         string
	PlanName       string
	ServiceID      string
	ServiceName    string
	DeploymentType string
	TenancyType    string
}

type servicePlanEnvironmentDetails struct {
	DeploymentModel          string
	EnabledFeatures          []string
	Clouds                   []string
	Regions                  []string
	Deployments              []servicePlanDeploymentRow
	Subscriptions            []servicePlanSubscriptionRow
	Users                    []servicePlanUserRow
	DeploymentsCount         int
	ActiveSubscriptionsCount int
	UniqueUsersCount         int
	Err                      string
}

type servicePlanDeploymentRow struct {
	ID           string
	Status       string
	Cloud        string
	Region       string
	Subscription string
	Owner        string
}

type servicePlanSubscriptionRow struct {
	ID            string
	Status        string
	RootUserEmail string
	RootUserName  string
	InstanceCount int64
}

type servicePlanUserRow struct {
	ID      string
	Email   string
	Name    string
	Status  string
	OrgName string
}

type servicePlanBrowserLoader interface {
	LoadEnvironmentDetails(context.Context, string, servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error)
}

type productionServicePlanBrowserLoader struct{}

type servicePlanBrowserModel struct {
	catalog                servicePlanBrowserCatalog
	loadEnvironmentDetails func(servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error)
	expanded               map[int]bool
	detailCache            map[string]servicePlanEnvironmentDetails
	loadingDetails         map[string]bool
	environmentTabs        []string
	items                  []servicePlanBrowserLeftItem
	list                   list.Model
	viewport               viewport.Model
	spinner                spinner.Model
	focus                  servicePlanBrowserFocus
	detailCursor           int
	detailViewportTop      bool
	activeTab              int
	serviceIndex           int
	planIndex              int
	width                  int
	height                 int
	listPanelWidth         int
	detailPanelWidth       int
	statusMessage          string
	modal                  *servicePlanBrowserModal
}

type servicePlanBrowserLeftItem struct {
	key          string
	parentKey    string
	title        string
	description  string
	level        int
	expandable   bool
	expanded     bool
	isService    bool
	hostingBadge servicePlanHostingBadge
	serviceIndex int
	planIndex    int
}

func (i servicePlanBrowserLeftItem) Title() string       { return i.title }
func (i servicePlanBrowserLeftItem) Description() string { return i.description }
func (i servicePlanBrowserLeftItem) FilterValue() string {
	return i.title + " " + i.description
}

type servicePlanBrowserDelegate struct {
	list.DefaultDelegate
}

type servicePlanBrowserDetailRow struct {
	Label     string
	Value     string
	ModalKind servicePlanBrowserModalKind
}

type servicePlanBrowserModal struct {
	Kind   servicePlanBrowserModalKind
	Title  string
	Rows   []servicePlanBrowserModalRow
	Filter string
	Cursor int
}

type servicePlanBrowserModalRow struct {
	Text   string
	Search string
}

type servicePlanBrowserDetailsLoadedMsg struct {
	cacheKey string
	detail   servicePlanEnvironmentDetails
}

func newServicePlanBrowserDelegate() servicePlanBrowserDelegate {
	delegate := servicePlanBrowserDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	delegate.SetHeight(2)
	delegate.SetSpacing(0)
	return delegate
}

func (d servicePlanBrowserDelegate) Render(writer io.Writer, model list.Model, index int, item list.Item) {
	browserItem, ok := item.(servicePlanBrowserLeftItem)
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

	indent := strings.Repeat("  ", browserItem.level)
	titlePrefix := ""
	switch {
	case browserItem.expandable && browserItem.expanded:
		titlePrefix = "▾ "
	case browserItem.expandable:
		titlePrefix = "▸ "
	case browserItem.level > 0:
		titlePrefix = "• "
	}

	title := indent + titlePrefix + browserItem.title
	description := browserItem.description
	if description != "" {
		description = indent + "  " + description
	}

	renderedTitle := titleStyle.Render(title)
	if browserItem.hostingBadge.Label != "" {
		renderedTitle = lipgloss.JoinHorizontal(lipgloss.Center, renderedTitle, " ", renderServicePlanHostingBadge(browserItem.hostingBadge))
	}

	fmt.Fprintf(writer, "%s\n%s", renderedTitle, descriptionStyle.Render(description))
}

func buildServicePlanBrowserCatalog(services []openapiclient.DescribeServiceResult, filterMaps []map[string]string) servicePlanBrowserCatalog {
	catalog := servicePlanBrowserCatalog{Services: make([]servicePlanBrowserService, 0, len(services))}

	for _, service := range services {
		browserService := servicePlanBrowserService{
			ID:    service.Id,
			Name:  service.Name,
			Plans: make([]servicePlanBrowserPlan, 0),
		}
		planIndexes := map[string]int{}

		for _, env := range service.ServiceEnvironments {
			for _, plan := range env.ServicePlans {
				formatted := formatServicePlan(service.Id, service.Name, env.Name, plan, false)
				match, err := utils.MatchesFilters(formatted, filterMaps)
				if err != nil || !match {
					continue
				}

				planName := strings.TrimSpace(plan.Name)
				if planName == "" {
					planName = plan.ProductTierID
				}
				planKey := strings.ToLower(planName)
				planIndex, ok := planIndexes[planKey]
				if !ok {
					planIndex = len(browserService.Plans)
					planIndexes[planKey] = planIndex
					browserService.Plans = append(browserService.Plans, servicePlanBrowserPlan{
						Name:        planName,
						ServiceID:   service.Id,
						ServiceName: service.Name,
					})
				}

				browserService.Plans[planIndex].Environments = append(browserService.Plans[planIndex].Environments, servicePlanBrowserEnvironment{
					ID:             env.Id,
					Name:           env.Name,
					PlanID:         plan.ProductTierID,
					PlanName:       planName,
					ServiceID:      service.Id,
					ServiceName:    service.Name,
					DeploymentType: plan.TierType,
					TenancyType:    plan.ModelType,
				})
			}
		}

		if len(browserService.Plans) > 0 {
			catalog.Services = append(catalog.Services, browserService)
		}
	}

	return catalog
}

func runServicePlanBrowser(ctx context.Context, token string, catalog servicePlanBrowserCatalog) error {
	model := newServicePlanBrowserModel(ctx, token, catalog, productionServicePlanBrowserLoader{})

	snapshot := renderServicePlanBrowserSnapshot(model)
	utils.LastPrintedString = snapshot

	if !isServicePlanBrowserInteractive() {
		fmt.Println(snapshot)
		return nil
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to launch service plan browser: %w", err)
	}

	return nil
}

func newServicePlanBrowserModel(ctx context.Context, token string, catalog servicePlanBrowserCatalog, loader servicePlanBrowserLoader) servicePlanBrowserModel {
	delegate := newServicePlanBrowserDelegate()
	planList := list.New(nil, delegate, 0, 0)
	planList.Title = "Service Plans"
	planList.SetShowHelp(false)
	planList.SetShowFilter(false)
	planList.SetShowStatusBar(false)
	planList.SetFilteringEnabled(false)
	planList.DisableQuitKeybindings()
	planList.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Padding(0, 1)

	detailSpinner := spinner.New()
	detailSpinner.Spinner = spinner.Dot
	detailSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	model := servicePlanBrowserModel{
		catalog:         catalog,
		expanded:        map[int]bool{},
		detailCache:     map[string]servicePlanEnvironmentDetails{},
		loadingDetails:  map[string]bool{},
		environmentTabs: servicePlanBrowserEnvironmentTabs(catalog),
		list:            planList,
		viewport:        viewport.New(0, 0),
		spinner:         detailSpinner,
		focus:           servicePlanBrowserFocusLeft,
	}
	if loader != nil {
		model.loadEnvironmentDetails = func(env servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error) {
			return loader.LoadEnvironmentDetails(ctx, token, env)
		}
	}
	model.setSize(defaultServicePlanBrowserWidth, defaultServicePlanBrowserHeight)
	model.ensureActiveEnvironmentExpanded()
	model.rebuildVisibleItems(model.firstPlanKey())
	if model.requestSelectedDetailsLoad() != nil {
		model.syncViewportContent()
	}
	return model
}

func (m servicePlanBrowserModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.selectedDetailsLoadCmd())
}

func (m servicePlanBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case spinner.TickMsg:
		if !m.hasLoadingDetails() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.syncViewportContent()
		return m, cmd
	case servicePlanBrowserDetailsLoadedMsg:
		delete(m.loadingDetails, msg.cacheKey)
		m.detailCache[msg.cacheKey] = msg.detail
		m.syncViewportContent()
		return m, nil
	case tea.KeyMsg:
		if m.modal != nil {
			return m.updateModal(msg), nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			return m, m.moveEnvironmentTab(1)
		case "shift+tab":
			return m, m.moveEnvironmentTab(-1)
		}

		if m.focus == servicePlanBrowserFocusDetails {
			return m.updateDetails(msg)
		}

		switch msg.String() {
		case "enter":
			_, loadCmd := m.enterDetailsPane()
			return m, loadCmd
		case " ", "right":
			if m.expandSelectedService() {
				return m, nil
			}
			if entered, loadCmd := m.enterDetailsPane(); entered {
				return m, loadCmd
			}
		case "left":
			if m.collapseSelectedItem() {
				return m, nil
			}
		}
	}

	previousKey := m.selectedLeftItemKey()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.selectedLeftItemKey() != previousKey {
		m.syncSelectionFromList()
		loadCmd := m.requestSelectedDetailsLoad()
		m.syncViewportContent()
		cmd = tea.Batch(cmd, loadCmd)
	}
	return m, cmd
}

func (m servicePlanBrowserModel) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "left":
		m.focus = servicePlanBrowserFocusLeft
		m.syncViewportContent()
	case "enter":
		m.openSelectedModal()
	case "down":
		m.moveDetailCursor(1)
	case "up":
		m.moveDetailCursor(-1)
	case "pgdown", "f", " ":
		m.detailViewportTop = false
		m.viewport.PageDown()
	case "pgup", "b":
		m.detailViewportTop = false
		m.viewport.PageUp()
	case "g", "home":
		m.detailCursor = -1
		m.detailViewportTop = true
		m.syncViewportContent()
	case "G", "end":
		m.detailViewportTop = false
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m servicePlanBrowserModel) updateModal(msg tea.KeyMsg) servicePlanBrowserModel {
	switch msg.String() {
	case "ctrl+c", "q":
		return m
	case "esc":
		m.modal = nil
		return m
	case "up", "k":
		m.modal.Cursor = spClamp(m.modal.Cursor-1, spMax(0, len(m.modal.filteredRows())-1))
		return m
	case "down", "j":
		m.modal.Cursor = spClamp(m.modal.Cursor+1, spMax(0, len(m.modal.filteredRows())-1))
		return m
	case "backspace", "ctrl+h":
		if m.modal.Filter != "" {
			runes := []rune(m.modal.Filter)
			m.modal.Filter = string(runes[:len(runes)-1])
			m.modal.Cursor = 0
		}
		return m
	}

	if len(msg.Runes) > 0 {
		m.modal.Filter += string(msg.Runes)
		m.modal.Cursor = 0
	}

	return m
}

func (m servicePlanBrowserModel) View() string {
	if m.modal != nil {
		return m.renderModal()
	}

	header := renderServicePlanBrowserHeader(m.catalog)
	tabs := m.renderEnvironmentTabs(m.width)

	listPanelStyle := servicePlanBrowserPanelStyle(m.focusBorderColor(servicePlanBrowserFocusLeft))
	detailPanelStyle := servicePlanBrowserPanelStyle(m.focusBorderColor(servicePlanBrowserFocusDetails))
	plansPanel := listPanelStyle.Width(m.listPanelWidth).Render(m.list.View())
	detailsPanel := detailPanelStyle.Width(m.detailPanelWidth).Render(m.viewport.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, plansPanel, " ", detailsPanel)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.helpLine())
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render(m.statusLine())
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, help, status)
}

func renderServicePlanBrowserSnapshot(model servicePlanBrowserModel) string {
	return strings.TrimRight(model.View(), "\n")
}

func (m *servicePlanBrowserModel) setSize(width, height int) {
	if width <= 0 {
		width = defaultServicePlanBrowserWidth
	}
	if height <= 0 {
		height = defaultServicePlanBrowserHeight
	}

	m.width = width
	m.height = height

	listPanelWidth := spMin(spMax(width/3, servicePlanBrowserListMinWidth), servicePlanBrowserListPreferredWidth)
	bodyHeight := spMax(height-servicePlanBrowserHeaderHeight-servicePlanBrowserTabsHeight-servicePlanBrowserHelpHeight, 12)
	detailPanelWidth := spMax(width-listPanelWidth-1, 40)
	panelFrameWidth := servicePlanBrowserPanelStyle(lipgloss.Color("240")).GetHorizontalFrameSize()

	m.listPanelWidth = listPanelWidth
	m.detailPanelWidth = detailPanelWidth
	m.list.SetWidth(spMax(listPanelWidth-panelFrameWidth, 10))
	m.list.SetHeight(bodyHeight)
	m.viewport.Width = spMax(detailPanelWidth-panelFrameWidth, 10)
	m.viewport.Height = bodyHeight
	m.syncViewportContent()
}

func servicePlanBrowserPanelStyle(borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}

func (m servicePlanBrowserModel) focusBorderColor(focus servicePlanBrowserFocus) lipgloss.Color {
	if m.focus == focus {
		return lipgloss.Color("117")
	}
	return lipgloss.Color("240")
}

func (m servicePlanBrowserModel) statusLine() string {
	if strings.TrimSpace(m.statusMessage) != "" {
		return m.statusMessage
	}
	if m.focus == servicePlanBrowserFocusDetails {
		return "Use tab or shift+tab to switch environments. Use esc or left to return to the plan list."
	}
	return "Use tab or shift+tab to switch environments. Plan details update as the selector moves."
}

func (m servicePlanBrowserModel) helpLine() string {
	if m.focus == servicePlanBrowserFocusDetails {
		return "tab/shift+tab: environment  ↑/↓: detail rows  enter: open row  esc/←: focus plans  q: quit"
	}
	return "tab/shift+tab: environment  ↑/↓: navigate plans  enter/→: details/open  ←/→: expand/collapse services  q: quit"
}

func (m *servicePlanBrowserModel) rebuildVisibleItems(selectedKey string) {
	if selectedKey == "" {
		selectedKey = m.selectedLeftItemKey()
	}

	m.items = m.leftItems()
	listItems := make([]list.Item, len(m.items))
	if selectedKey == "" || !servicePlanBrowserItemsContainKey(m.items, selectedKey) {
		selectedKey = firstServicePlanBrowserPlanKey(m.items)
	}
	if selectedKey == "" && len(m.items) > 0 {
		selectedKey = m.items[0].key
	}

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
	m.list.Title = "Service Plans"
	if env := m.activeEnvironmentName(); env != "" {
		m.list.Title = "Service Plans · " + env
	}
	m.syncSelectionFromList()
	m.syncViewportContent()
}

func (m servicePlanBrowserModel) firstPlanKey() string {
	return firstServicePlanBrowserPlanKey(m.leftItems())
}

func servicePlanBrowserItemsContainKey(items []servicePlanBrowserLeftItem, key string) bool {
	for _, item := range items {
		if item.key == key {
			return true
		}
	}
	return false
}

func firstServicePlanBrowserPlanKey(items []servicePlanBrowserLeftItem) string {
	for _, item := range items {
		if !item.isService {
			return item.key
		}
	}
	return ""
}

func servicePlanBrowserEnvironmentTabs(catalog servicePlanBrowserCatalog) []string {
	seen := map[string]bool{}
	tabs := make([]string, 0)
	for _, service := range catalog.Services {
		for _, plan := range service.Plans {
			for _, env := range plan.Environments {
				name := strings.TrimSpace(env.Name)
				key := servicePlanEnvironmentKey(name)
				if name == "" {
					name = "-"
				}
				if seen[key] {
					continue
				}
				seen[key] = true
				tabs = append(tabs, name)
			}
		}
	}
	return tabs
}

func servicePlanEnvironmentKey(environment string) string {
	environment = strings.ToLower(strings.TrimSpace(environment))
	if environment == "" {
		return "-"
	}
	return environment
}

func (m servicePlanBrowserModel) activeEnvironmentName() string {
	if len(m.environmentTabs) == 0 {
		return ""
	}
	activeTab := spClamp(m.activeTab, len(m.environmentTabs)-1)
	return m.environmentTabs[activeTab]
}

func (m servicePlanBrowserModel) environmentMatchesActive(env servicePlanBrowserEnvironment) bool {
	if len(m.environmentTabs) == 0 {
		return true
	}
	return servicePlanEnvironmentKey(env.Name) == servicePlanEnvironmentKey(m.activeEnvironmentName())
}

func (m servicePlanBrowserModel) planForActiveEnvironment(plan servicePlanBrowserPlan) (servicePlanBrowserPlan, bool) {
	filtered := plan
	filtered.Environments = make([]servicePlanBrowserEnvironment, 0, len(plan.Environments))
	for _, env := range plan.Environments {
		if m.environmentMatchesActive(env) {
			filtered.Environments = append(filtered.Environments, env)
		}
	}
	return filtered, len(filtered.Environments) > 0
}

func (m servicePlanBrowserModel) servicePlansForActiveEnvironment(service servicePlanBrowserService) []servicePlanBrowserPlan {
	plans := make([]servicePlanBrowserPlan, 0, len(service.Plans))
	for _, plan := range service.Plans {
		filtered, ok := m.planForActiveEnvironment(plan)
		if ok {
			plans = append(plans, filtered)
		}
	}
	return plans
}

func (m *servicePlanBrowserModel) ensureActiveEnvironmentExpanded() {
	for serviceIndex, service := range m.catalog.Services {
		if len(m.servicePlansForActiveEnvironment(service)) == 0 {
			continue
		}
		if m.expanded[serviceIndex] {
			return
		}
	}
	for serviceIndex, service := range m.catalog.Services {
		if len(m.servicePlansForActiveEnvironment(service)) == 0 {
			continue
		}
		m.expanded[serviceIndex] = true
		return
	}
}

func (m servicePlanBrowserModel) selectedLeftItemKey() string {
	item := m.selectedLeftItem()
	if item == nil {
		return ""
	}
	return item.key
}

func (m servicePlanBrowserModel) selectedLeftItem() *servicePlanBrowserLeftItem {
	index := m.list.Index()
	if index < 0 || index >= len(m.items) {
		return nil
	}
	return &m.items[index]
}

func (m *servicePlanBrowserModel) syncSelectionFromList() {
	item := m.selectedLeftItem()
	if item == nil {
		return
	}

	if item.isService {
		m.serviceIndex = item.serviceIndex
		return
	}

	changed := m.serviceIndex != item.serviceIndex || m.planIndex != item.planIndex
	m.serviceIndex = item.serviceIndex
	m.planIndex = item.planIndex
	if changed {
		m.detailCursor = 0
		m.detailViewportTop = false
	}
}

func (m *servicePlanBrowserModel) syncViewportContent() {
	selected := m.selectedLeftItem()
	if selected == nil {
		m.viewport.SetContent("No service plans found.")
		return
	}

	if selected.isService {
		m.viewport.SetContent(m.renderServiceContent(*selected, m.viewport.Width))
		m.viewport.GotoTop()
		return
	}

	content, cursorLine := m.renderPlanContentWithCursorLine(m.viewport.Width)
	m.viewport.SetContent(content)
	if m.focus == servicePlanBrowserFocusDetails {
		if m.detailViewportTop {
			m.viewport.GotoTop()
		} else {
			m.ensureViewportLineVisible(cursorLine, len(strings.Split(content, "\n")))
		}
	} else {
		m.viewport.GotoTop()
	}
}

func (m servicePlanBrowserModel) renderServiceContent(item servicePlanBrowserLeftItem, width int) string {
	service := m.catalog.Services[item.serviceIndex]
	plans := m.servicePlansForActiveEnvironment(service)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	lines := []string{
		titleStyle.Render(service.Name),
		"",
		sectionStyle.Render("Overview"),
	}
	lines = append(lines, renderServicePlanField("Service ID", emptyValue(service.ID), width, keyStyle, valueStyle)...)
	lines = append(lines, renderServicePlanField("Environment", emptyValue(m.activeEnvironmentName()), width, keyStyle, valueStyle)...)
	lines = append(lines, renderServicePlanField("Plans", fmt.Sprintf("%d", len(plans)), width, keyStyle, valueStyle)...)
	lines = append(lines, "", sectionStyle.Render("Plans in environment"))
	for _, plan := range plans {
		lines = append(lines, renderServicePlanBullet(plan.Name, width, keyStyle, valueStyle)...)
	}

	return strings.Join(lines, "\n")
}

func (m servicePlanBrowserModel) renderPlanContentWithCursorLine(width int) (string, int) {
	plan := m.selectedPlan()
	if plan == nil {
		return "No plan selected.", -1
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedKeyStyle := keyStyle.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	selectedValueStyle := valueStyle.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))

	lines := []string{
		titleStyle.Render(plan.ServiceName + " / " + plan.Name),
		"",
		sectionStyle.Render("Overview"),
	}
	lines = append(lines, renderServicePlanField("Plan name", plan.Name, width, keyStyle, valueStyle)...)
	lines = append(lines, renderServicePlanField("Service ID", emptyValue(plan.ServiceID), width, keyStyle, valueStyle)...)
	lines = append(lines, renderServicePlanField("Environment", emptyValue(m.activeEnvironmentName()), width, keyStyle, valueStyle)...)

	env := m.selectedEnvironment()
	if env == nil {
		lines = append(lines, "", sectionStyle.Render("Environment"))
		lines = append(lines, renderServicePlanField("Status", "No environment selected", width, keyStyle, valueStyle)...)
		return strings.Join(lines, "\n"), -1
	}

	lines = append(lines, "", sectionStyle.Render("Environment"))
	lines = append(lines, renderServicePlanField("Name", env.Name, width, keyStyle, valueStyle)...)
	lines = append(lines, renderServicePlanField("Plan ID", emptyValue(env.PlanID), width, keyStyle, valueStyle)...)

	lines = append(lines, "", sectionStyle.Render("Details"))
	rows := m.detailRows()
	cursorLine := -1
	for index, row := range rows {
		value := row.Value
		if row.ModalKind != "" {
			value += " (enter)"
		}
		rowKeyStyle := keyStyle
		rowValueStyle := valueStyle
		label := row.Label
		if m.focus == servicePlanBrowserFocusDetails && index == m.detailCursor {
			rowKeyStyle = selectedKeyStyle
			rowValueStyle = selectedValueStyle
			label = "▸ " + label
		}
		if index == m.detailCursor {
			cursorLine = servicePlanRenderedLineCount(lines)
		}
		lines = append(lines, renderServicePlanField(label, value, width, rowKeyStyle, rowValueStyle)...)
	}

	return strings.Join(lines, "\n"), cursorLine
}

func servicePlanRenderedLineCount(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	return len(strings.Split(strings.Join(lines, "\n"), "\n"))
}

func renderServicePlanField(label, value string, width int, keyStyle, valueStyle lipgloss.Style) []string {
	prefix := label + ": "
	valueWidth := spMax(width-lipgloss.Width(prefix), 16)
	valueLines := wrapServicePlanLine(value, valueWidth)
	if len(valueLines) == 0 {
		valueLines = []string{""}
	}

	rendered := make([]string, 0, len(valueLines))
	continuationIndent := strings.Repeat(" ", lipgloss.Width(prefix))
	for index, valueLine := range valueLines {
		if index == 0 {
			rendered = append(rendered, keyStyle.Render(prefix)+valueStyle.Render(valueLine))
			continue
		}
		rendered = append(rendered, continuationIndent+valueStyle.Render(valueLine))
	}

	return rendered
}

func renderServicePlanBullet(line string, width int, keyStyle, valueStyle lipgloss.Style) []string {
	item := strings.TrimSpace(line)
	if item == "" {
		return []string{line}
	}

	if label, value, ok := strings.Cut(item, ": "); ok {
		prefix := "• " + label + ": "
		valueWidth := spMax(width-lipgloss.Width(prefix), 16)
		valueLines := wrapServicePlanLine(value, valueWidth)
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

	wrapped := wrapServicePlanLine("• "+item, width)
	rendered := make([]string, 0, len(wrapped))
	for _, wrappedLine := range wrapped {
		rendered = append(rendered, keyStyle.Render(wrappedLine))
	}
	return rendered
}

func wrapServicePlanLine(line string, width int) []string {
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
			if isServicePlanSoftBreak(r) {
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

func isServicePlanSoftBreak(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("/:?&=_-.,", r)
}

func renderServicePlanBrowserHeader(catalog servicePlanBrowserCatalog) string {
	serviceCount := len(catalog.Services)
	planCount := 0
	envCount := 0
	for _, service := range catalog.Services {
		planCount += len(service.Plans)
		for _, plan := range service.Plans {
			envCount += len(plan.Environments)
		}
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Service Plan Browser")
	metadata := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
		fmt.Sprintf("%d service(s)  |  %d plan name(s)  |  %d environment variant(s)", serviceCount, planCount, envCount),
	)
	checks := []string{
		renderServicePlanBrowserCheck("Service catalog loaded", serviceCount > 0),
		renderServicePlanBrowserCheck("Plan details available", planCount > 0),
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, metadata, strings.Join(checks, "  "))
}

func renderServicePlanBrowserCheck(label string, ok bool) string {
	icon := "○"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	if ok {
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	}
	return style.Render(fmt.Sprintf("%s %s", icon, label))
}

func (m servicePlanBrowserModel) renderEnvironmentTabs(width int) string {
	if len(m.environmentTabs) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No environments")
	}

	highlightColor := lipgloss.Color("62")
	inactiveTabBorder := servicePlanTabBorderWithBottom("┴", "─", "┴")
	activeTabBorder := servicePlanTabBorderWithBottom("┘", " ", "└")

	inactiveTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(highlightColor).
		Padding(0, 1)
	activeTabStyle := lipgloss.NewStyle().
		Border(activeTabBorder, true).
		BorderForeground(highlightColor).
		Padding(0, 1).
		Bold(true)

	renderedTabs := make([]string, 0, len(m.environmentTabs))
	for i, name := range m.environmentTabs {
		envColor := servicePlanEnvironmentColor(name)
		style := inactiveTabStyle.Foreground(envColor).Faint(true)
		if i == m.activeTab {
			style = activeTabStyle.Foreground(envColor)
		}

		border, _, _, _, _ := style.GetBorder()
		if i == 0 {
			if i == m.activeTab {
				border.BottomLeft = "│"
			} else {
				border.BottomLeft = "├"
			}
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(emptyValue(name)))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	rowWidth := lipgloss.Width(row)
	gapWidth := width - rowWidth - 2
	if gapWidth > 0 {
		gapBorder := lipgloss.Border{
			Bottom:      "─",
			BottomLeft:  "┴",
			BottomRight: "┐",
		}
		gapStyle := lipgloss.NewStyle().
			Border(gapBorder, false, false, true, false).
			BorderForeground(highlightColor)
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gapStyle.Render(strings.Repeat(" ", gapWidth)))
	}

	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

func servicePlanTabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func servicePlanEnvironmentColor(environment string) lipgloss.Color {
	normalized := strings.ToLower(strings.TrimSpace(environment))
	switch normalized {
	case "prod", "production":
		return lipgloss.Color("160")
	case "stage", "staging":
		return lipgloss.Color("214")
	case "dev", "development":
		return lipgloss.Color("82")
	case "qa", "test", "testing":
		return lipgloss.Color("39")
	default:
		return lipgloss.Color("62")
	}
}

func (m servicePlanBrowserModel) detailRows() []servicePlanBrowserDetailRow {
	env := m.selectedEnvironment()
	if env == nil {
		return []servicePlanBrowserDetailRow{{Label: "Status", Value: "No environment selected"}}
	}

	detail, ok := m.detailCache[env.cacheKey()]
	if !ok {
		if m.loadingDetails[env.cacheKey()] {
			return []servicePlanBrowserDetailRow{{Label: "Status", Value: m.spinner.View() + " Loading details"}}
		}
		return []servicePlanBrowserDetailRow{{Label: "Status", Value: "Details not loaded"}}
	}
	if detail.Err != "" {
		return []servicePlanBrowserDetailRow{{Label: "Error", Value: detail.Err}}
	}

	return []servicePlanBrowserDetailRow{
		{Label: "Deployment model", Value: emptyValue(detail.DeploymentModel)},
		{Label: "Enabled features", Value: joinOrNone(detail.EnabledFeatures)},
		{Label: "Clouds", Value: joinOrNone(detail.Clouds)},
		{Label: "Regions", Value: joinOrNone(detail.Regions)},
		{Label: "Deployments", Value: fmt.Sprintf("%d", detail.DeploymentsCount), ModalKind: servicePlanBrowserModalDeployments},
		{Label: "Subscriptions", Value: fmt.Sprintf("%d", detail.ActiveSubscriptionsCount), ModalKind: servicePlanBrowserModalSubscriptions},
		{Label: "Users", Value: fmt.Sprintf("%d", detail.UniqueUsersCount), ModalKind: servicePlanBrowserModalUsers},
	}
}

func (m servicePlanBrowserModel) renderModal() string {
	rows := m.modal.filteredRows()
	width := spMax(m.width, 80)
	height := spMax(m.height, 24)
	contentHeight := spMax(height-6, 8)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Render(" " + m.modal.Title + " "),
		"Filter: " + m.modal.Filter,
		"",
	}

	if len(rows) == 0 {
		lines = append(lines, "No matching rows")
	} else {
		start := 0
		if m.modal.Cursor >= contentHeight {
			start = m.modal.Cursor - contentHeight + 1
		}
		end := spMin(len(rows), start+contentHeight)
		for i := start; i < end; i++ {
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			if i == m.modal.Cursor {
				prefix = "▸ "
				style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
			}
			lines = append(lines, style.Render(prefix+rows[i].Text))
		}
	}

	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("esc: close  type: filter  ↑/↓: navigate"))

	return servicePlanBrowserPanelStyle(lipgloss.Color("117")).
		Width(width - 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *servicePlanBrowserModel) moveDetailCursor(delta int) {
	rows := m.detailRows()
	if len(rows) == 0 {
		m.detailCursor = -1
		m.detailViewportTop = true
		m.syncViewportContent()
		return
	}

	switch {
	case m.detailCursor <= 0 && delta < 0:
		m.detailCursor = -1
		m.detailViewportTop = true
	case m.detailCursor < 0 && delta <= 0:
		m.detailCursor = -1
		m.detailViewportTop = true
	case m.detailCursor < 0 && delta > 0:
		m.detailCursor = 0
		m.detailViewportTop = false
	default:
		m.detailCursor = spClamp(m.detailCursor+delta, len(rows)-1)
		m.detailViewportTop = false
	}
	m.syncViewportContent()
}

func (m *servicePlanBrowserModel) ensureViewportLineVisible(line, totalLines int) {
	if line < 0 {
		return
	}

	visibleRows := m.viewport.Height
	if visibleRows < 1 {
		visibleRows = 1
	}

	maxScroll := totalLines - visibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}

	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}
	if m.viewport.YOffset > maxScroll {
		m.viewport.YOffset = maxScroll
	}
	if line < m.viewport.YOffset {
		m.viewport.YOffset = line
	}
	if line >= m.viewport.YOffset+visibleRows {
		m.viewport.YOffset = line - visibleRows + 1
	}
	if m.viewport.YOffset > maxScroll {
		m.viewport.YOffset = maxScroll
	}
}

func (m *servicePlanBrowserModel) moveEnvironmentTab(delta int) tea.Cmd {
	if len(m.environmentTabs) == 0 {
		return nil
	}

	selectedKey := m.selectedLeftItemKey()
	m.activeTab = (m.activeTab + delta + len(m.environmentTabs)) % len(m.environmentTabs)
	m.detailCursor = -1
	m.detailViewportTop = true
	m.ensureActiveEnvironmentExpanded()
	m.rebuildVisibleItems(selectedKey)
	loadCmd := m.requestSelectedDetailsLoad()
	m.syncViewportContent()
	return loadCmd
}

func (m *servicePlanBrowserModel) enterDetailsPane() (bool, tea.Cmd) {
	selected := m.selectedLeftItem()
	if selected == nil || selected.isService {
		return false, nil
	}

	m.focus = servicePlanBrowserFocusDetails
	m.syncSelectionFromList()
	loadCmd := m.requestSelectedDetailsLoad()
	m.syncViewportContent()
	return true, loadCmd
}

func (m *servicePlanBrowserModel) expandSelectedService() bool {
	selected := m.selectedLeftItem()
	if selected == nil || !selected.expandable {
		return false
	}
	if selected.expanded {
		return false
	}

	m.expanded[selected.serviceIndex] = true
	m.rebuildVisibleItems(selected.key)
	return true
}

func (m *servicePlanBrowserModel) collapseSelectedItem() bool {
	selected := m.selectedLeftItem()
	if selected == nil {
		return false
	}

	if selected.expandable && selected.expanded {
		m.expanded[selected.serviceIndex] = false
		m.rebuildVisibleItems(selected.key)
		return true
	}

	if selected.parentKey == "" {
		return false
	}

	m.expanded[selected.serviceIndex] = false
	m.rebuildVisibleItems(selected.parentKey)
	return true
}

func (m *servicePlanBrowserModel) openSelectedModal() {
	rows := m.detailRows()
	if len(rows) == 0 || m.detailCursor < 0 || m.detailCursor >= len(rows) {
		return
	}
	row := rows[m.detailCursor]
	if row.ModalKind == "" {
		m.statusMessage = "Selected detail row does not have expandable rows"
		return
	}

	detail, ok := m.selectedDetails()
	if !ok {
		return
	}

	modalRows := make([]servicePlanBrowserModalRow, 0)
	title := row.Label
	switch row.ModalKind {
	case servicePlanBrowserModalDeployments:
		for _, deployment := range detail.Deployments {
			text := fmt.Sprintf("%s | %s | %s | %s | %s", emptyValue(deployment.ID), emptyValue(deployment.Status), emptyValue(deployment.Cloud), emptyValue(deployment.Region), emptyValue(deployment.Owner))
			modalRows = append(modalRows, servicePlanBrowserModalRow{Text: text, Search: strings.ToLower(text)})
		}
	case servicePlanBrowserModalSubscriptions:
		for _, subscription := range detail.Subscriptions {
			text := fmt.Sprintf("%s | %s | %s | %s | instances=%d", emptyValue(subscription.ID), emptyValue(subscription.Status), emptyValue(subscription.RootUserEmail), emptyValue(subscription.RootUserName), subscription.InstanceCount)
			modalRows = append(modalRows, servicePlanBrowserModalRow{Text: text, Search: strings.ToLower(text)})
		}
	case servicePlanBrowserModalUsers:
		for _, user := range detail.Users {
			text := fmt.Sprintf("%s | %s | %s | %s | %s", emptyValue(user.ID), emptyValue(user.Email), emptyValue(user.Name), emptyValue(user.Status), emptyValue(user.OrgName))
			modalRows = append(modalRows, servicePlanBrowserModalRow{Text: text, Search: strings.ToLower(text)})
		}
	}

	m.modal = &servicePlanBrowserModal{
		Kind:  row.ModalKind,
		Title: title,
		Rows:  modalRows,
	}
}

func (m servicePlanBrowserModel) leftItems() []servicePlanBrowserLeftItem {
	items := make([]servicePlanBrowserLeftItem, 0)
	for serviceIndex, service := range m.catalog.Services {
		planItems := make([]servicePlanBrowserLeftItem, 0, len(service.Plans))
		serviceKey := fmt.Sprintf("service:%d", serviceIndex)
		for planIndex, plan := range service.Plans {
			filteredPlan, ok := m.planForActiveEnvironment(plan)
			if !ok {
				continue
			}
			planItems = append(planItems, servicePlanBrowserLeftItem{
				key:          fmt.Sprintf("%s/plan:%d", serviceKey, planIndex),
				parentKey:    serviceKey,
				title:        plan.Name,
				level:        1,
				hostingBadge: servicePlanHostingBadgeForPlan(filteredPlan),
				serviceIndex: serviceIndex,
				planIndex:    planIndex,
			})
		}
		if len(planItems) == 0 {
			continue
		}

		items = append(items, servicePlanBrowserLeftItem{
			key:          serviceKey,
			title:        service.Name,
			description:  fmt.Sprintf("%d plan name(s)", len(planItems)),
			expandable:   true,
			expanded:     m.expanded[serviceIndex],
			isService:    true,
			serviceIndex: serviceIndex,
		})
		if !m.expanded[serviceIndex] {
			continue
		}
		items = append(items, planItems...)
	}
	return items
}

func servicePlanHostingBadgeForPlan(plan servicePlanBrowserPlan) servicePlanHostingBadge {
	for _, env := range plan.Environments {
		if badge := servicePlanHostingBadgeForValues(env.TenancyType, env.DeploymentType); badge.Label != "" {
			return badge
		}
	}
	return servicePlanHostingBadgeForValues("", "")
}

func servicePlanHostingBadgeForValues(modelType, tierType string) servicePlanHostingBadge {
	values := []string{modelType, tierType}
	normalizedValues := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		value = strings.ReplaceAll(value, "-", "_")
		value = strings.ReplaceAll(value, " ", "_")
		if value != "" {
			normalizedValues = append(normalizedValues, value)
		}
	}

	for _, value := range normalizedValues {
		switch value {
		case "CUSTOMER_CLOUD", "BYOA", "BYOC":
			return servicePlanHostingBadge{Label: "BYOC", Color: lipgloss.Color("166")}
		}
	}
	for _, value := range normalizedValues {
		if value == "OMNISTRATE_HOSTED" {
			return servicePlanHostingBadge{Label: "Omnistrate Hosted", Color: lipgloss.Color("29")}
		}
	}
	for _, value := range normalizedValues {
		switch value {
		case "CUSTOMER_HOSTED", "HOSTED", "DEDICATED", "SHARED", "OMNISTRATE_DEDICATED_TENANCY", "OMNISTRATE_MULTI_TENANCY":
			return servicePlanHostingBadge{Label: "Hosted", Color: lipgloss.Color("33")}
		}
	}
	return servicePlanHostingBadge{Label: "Hosted", Color: lipgloss.Color("33")}
}

func renderServicePlanHostingBadge(badge servicePlanHostingBadge) string {
	if badge.Label == "" {
		return ""
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(badge.Color).
		Padding(0, 1).
		Render(badge.Label)
}

func (m servicePlanBrowserModel) selectedPlan() *servicePlanBrowserPlan {
	if m.serviceIndex < 0 || m.serviceIndex >= len(m.catalog.Services) {
		return nil
	}
	plans := m.catalog.Services[m.serviceIndex].Plans
	if m.planIndex < 0 || m.planIndex >= len(plans) {
		return nil
	}
	return &plans[m.planIndex]
}

func (m servicePlanBrowserModel) selectedEnvironment() *servicePlanBrowserEnvironment {
	plan := m.selectedPlan()
	if plan == nil || len(plan.Environments) == 0 {
		return nil
	}
	for i := range plan.Environments {
		if m.environmentMatchesActive(plan.Environments[i]) {
			return &plan.Environments[i]
		}
	}
	return nil
}

func (m servicePlanBrowserModel) selectedDetails() (servicePlanEnvironmentDetails, bool) {
	env := m.selectedEnvironment()
	if env == nil {
		return servicePlanEnvironmentDetails{}, false
	}
	detail, ok := m.detailCache[env.cacheKey()]
	return detail, ok
}

func (m *servicePlanBrowserModel) requestSelectedDetailsLoad() tea.Cmd {
	env := m.selectedEnvironment()
	if env == nil {
		return nil
	}
	key := env.cacheKey()
	if _, ok := m.detailCache[key]; ok {
		return nil
	}
	if m.loadingDetails[key] {
		return nil
	}
	if m.loadEnvironmentDetails == nil {
		return nil
	}

	m.loadingDetails[key] = true
	return tea.Batch(m.spinner.Tick, m.loadDetailsCmd(*env))
}

func (m servicePlanBrowserModel) selectedDetailsLoadCmd() tea.Cmd {
	env := m.selectedEnvironment()
	if env == nil {
		return nil
	}
	if !m.loadingDetails[env.cacheKey()] {
		return nil
	}
	return m.loadDetailsCmd(*env)
}

func (m servicePlanBrowserModel) loadDetailsCmd(env servicePlanBrowserEnvironment) tea.Cmd {
	if m.loadEnvironmentDetails == nil {
		return nil
	}

	return func() tea.Msg {
		detail, err := m.loadEnvironmentDetails(env)
		if err != nil {
			detail = servicePlanEnvironmentDetails{Err: err.Error()}
		}
		return servicePlanBrowserDetailsLoadedMsg{
			cacheKey: env.cacheKey(),
			detail:   detail,
		}
	}
}

func (m servicePlanBrowserModel) hasLoadingDetails() bool {
	for _, loading := range m.loadingDetails {
		if loading {
			return true
		}
	}
	return false
}

func (e servicePlanBrowserEnvironment) cacheKey() string {
	return e.ServiceID + "/" + e.ID + "/" + e.PlanID
}

func (m servicePlanBrowserModal) filteredRows() []servicePlanBrowserModalRow {
	return filterServicePlanModalRows(m.Rows, m.Filter)
}

func filterServicePlanModalRows(rows []servicePlanBrowserModalRow, filter string) []servicePlanBrowserModalRow {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return rows
	}

	filtered := make([]servicePlanBrowserModalRow, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(row.Search, filter) || strings.Contains(strings.ToLower(row.Text), filter) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (productionServicePlanBrowserLoader) LoadEnvironmentDetails(ctx context.Context, token string, env servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error) {
	pageSize := int64(100)
	exclude := true

	productTier, err := dataaccess.DescribeProductTier(ctx, token, env.ServiceID, env.PlanID)
	if err != nil {
		return servicePlanEnvironmentDetails{}, err
	}

	deployments, err := dataaccess.ListAllResourceInstances(ctx, token, env.ServiceID, env.ID, &dataaccess.ListResourceInstanceOptions{
		ProductTierId:           &env.PlanID,
		PageSize:                &pageSize,
		ExcludeNetworkTopology:  &exclude,
		ExcludeHAStatus:         &exclude,
		ExcludeIntegrations:     &exclude,
		ExcludeMaintenanceTasks: &exclude,
	})
	if err != nil {
		return servicePlanEnvironmentDetails{}, err
	}

	includeInactive := false
	subscriptions, err := dataaccess.ListAllSubscriptions(ctx, token, env.ServiceID, env.ID, &dataaccess.ListSubscriptionsOptions{
		ProductTierId:   &env.PlanID,
		IncludeInactive: &includeInactive,
		ExcludePricing:  &exclude,
		PageSize:        &pageSize,
	})
	if err != nil {
		return servicePlanEnvironmentDetails{}, err
	}
	activeSubscriptions := activeServicePlanSubscriptions(subscriptions)

	allUsers := make([]openapiclientfleet.User, 0)
	for _, subscription := range activeSubscriptions {
		if strings.TrimSpace(subscription.Id) == "" {
			continue
		}
		subscriptionID := subscription.Id
		users, err := dataaccess.ListAllUsers(ctx, token, env.ServiceID, env.ID, &dataaccess.ListUsersOptions{
			SubscriptionId: &subscriptionID,
			ExcludeStats:   &exclude,
			PageSize:       &pageSize,
		})
		if err != nil {
			return servicePlanEnvironmentDetails{}, err
		}
		allUsers = append(allUsers, users...)
	}
	uniqueUsers := dedupeServicePlanUsers(allUsers)

	clouds, regions := productTierCloudsAndRegions(productTier)
	detail := servicePlanEnvironmentDetails{
		DeploymentModel:          servicePlanDeploymentModel(env, productTier),
		EnabledFeatures:          productTierEnabledFeatures(productTier),
		Clouds:                   clouds,
		Regions:                  regions,
		Deployments:              servicePlanDeploymentRows(deployments),
		Subscriptions:            servicePlanSubscriptionRows(activeSubscriptions),
		Users:                    servicePlanUserRows(uniqueUsers),
		DeploymentsCount:         len(deployments),
		ActiveSubscriptionsCount: len(activeSubscriptions),
		UniqueUsersCount:         len(uniqueUsers),
	}
	return detail, nil
}

func activeServicePlanSubscriptions(subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult) []openapiclientfleet.FleetDescribeSubscriptionResult {
	active := make([]openapiclientfleet.FleetDescribeSubscriptionResult, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		status := strings.ToLower(strings.TrimSpace(subscription.Status))
		switch status {
		case "inactive", "suspended", "cancelled", "canceled", "terminated", "deleted":
			continue
		default:
			active = append(active, subscription)
		}
	}
	return active
}

func dedupeServicePlanUsers(users []openapiclientfleet.User) []openapiclientfleet.User {
	seen := map[string]bool{}
	unique := make([]openapiclientfleet.User, 0, len(users))
	for _, user := range users {
		key := strings.TrimSpace(user.UserId)
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(user.Email))
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, user)
	}
	return unique
}

func productTierCloudsAndRegions(productTier *openapiclient.DescribeProductTierResult) ([]string, []string) {
	if productTier == nil {
		return nil, nil
	}

	cloudSet := map[string]bool{}
	regionSet := map[string]bool{}
	add := func(cloud string, regions []string) {
		if len(regions) == 0 {
			return
		}
		cloudSet[cloud] = true
		for _, region := range regions {
			if strings.TrimSpace(region) != "" {
				regionSet[region] = true
			}
		}
	}

	add("aws", productTier.AwsRegions)
	add("azure", productTier.AzureRegions)
	add("gcp", productTier.GcpRegions)
	add("oci", productTier.OciRegions)
	add("nebius", productTier.NebiusRegions)
	add("private", productTier.PrivateRegions)
	add("on-prem", productTier.OnPremPlatforms)

	if productTier.CloudProvidersConfigReadiness != nil {
		for cloud, regions := range *productTier.CloudProvidersConfigReadiness {
			if strings.TrimSpace(cloud) != "" {
				cloudSet[cloud] = true
			}
			for region := range regions {
				if strings.TrimSpace(region) != "" {
					regionSet[region] = true
				}
			}
		}
	}

	return sortedKeys(cloudSet), sortedKeys(regionSet)
}

func productTierEnabledFeatures(productTier *openapiclient.DescribeProductTierResult) []string {
	if productTier == nil {
		return nil
	}

	featureSet := map[string]bool{}
	for _, feature := range productTier.EnabledFeatures {
		name := strings.TrimSpace(feature.GetFeature())
		if name == "" {
			continue
		}
		scope := strings.TrimSpace(feature.GetScope())
		if scope != "" {
			name = name + " (" + scope + ")"
		}
		featureSet[name] = true
	}

	if productTier.Features != nil {
		for name, enabled := range *productTier.Features {
			if enabled && strings.TrimSpace(name) != "" {
				featureSet[name] = true
			}
		}
	}

	return sortedKeys(featureSet)
}

func servicePlanDeploymentModel(env servicePlanBrowserEnvironment, productTier *openapiclient.DescribeProductTierResult) string {
	parts := make([]string, 0, 2)
	if env.DeploymentType != "" {
		parts = append(parts, env.DeploymentType)
	} else if productTier != nil && productTier.TierType != "" {
		parts = append(parts, productTier.TierType)
	}
	if env.TenancyType != "" {
		parts = append(parts, env.TenancyType)
	}
	return strings.Join(parts, " / ")
}

func servicePlanDeploymentRows(instances []openapiclientfleet.ResourceInstance) []servicePlanDeploymentRow {
	rows := make([]servicePlanDeploymentRow, 0, len(instances))
	for _, instance := range instances {
		result := instance.ConsumptionResourceInstanceResult
		status := result.GetStatus()
		if status == "" {
			status = instance.TierVersionStatus
		}
		cloud := instance.CloudProvider
		if cloud == "" {
			cloud = result.GetCloudProvider()
		}
		rows = append(rows, servicePlanDeploymentRow{
			ID:           result.GetId(),
			Status:       status,
			Cloud:        cloud,
			Region:       result.GetRegion(),
			Subscription: instance.SubscriptionId,
			Owner:        instance.SubscriptionOwnerName,
		})
	}
	return rows
}

func servicePlanSubscriptionRows(subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult) []servicePlanSubscriptionRow {
	rows := make([]servicePlanSubscriptionRow, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		rows = append(rows, servicePlanSubscriptionRow{
			ID:            subscription.Id,
			Status:        subscription.Status,
			RootUserEmail: subscription.RootUserEmail,
			RootUserName:  subscription.RootUserName,
			InstanceCount: subscription.InstanceCount,
		})
	}
	return rows
}

func servicePlanUserRows(users []openapiclientfleet.User) []servicePlanUserRow {
	rows := make([]servicePlanUserRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, servicePlanUserRow{
			ID:      user.UserId,
			Email:   user.Email,
			Name:    user.UserName,
			Status:  user.Status,
			OrgName: user.OrgName,
		})
	}
	return rows
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func emptyValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func isServicePlanBrowserInteractive() bool {
	return servicePlanBrowserFileIsTerminal(os.Stdout) && servicePlanBrowserFileIsTerminal(os.Stdin)
}

func servicePlanBrowserFileIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}

	fd := file.Fd()
	if fd > uintptr(^uint(0)>>1) {
		return false
	}

	return term.IsTerminal(int(fd))
}

func spMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func spMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func spClamp(value, maxValue int) int {
	if maxValue < 0 {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
