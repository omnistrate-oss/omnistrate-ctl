package serviceplan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	servicePlanBrowserHeaderHeight       = 1
	servicePlanBrowserTabsHeight         = 3

	servicePlanBrowserFocusLeft servicePlanBrowserFocus = iota
	servicePlanBrowserFocusDetails

	servicePlanBrowserModalDeployments   servicePlanBrowserModalKind = "deployments"
	servicePlanBrowserModalSubscriptions servicePlanBrowserModalKind = "subscriptions"
	servicePlanBrowserModalUsers         servicePlanBrowserModalKind = "users"
	servicePlanBrowserModalCloudAccounts servicePlanBrowserModalKind = "cloud-accounts"

	servicePlanCustomerAccountConfigIDParamKey = "cloud_provider_account_config_id"
	servicePlanCustomerAccountResourcePrefix   = "r-injectedaccountconfig"
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
	DeploymentModel            string
	EnabledFeatures            []string
	Clouds                     []string
	Regions                    []string
	Deployments                []servicePlanDeploymentRow
	Subscriptions              []servicePlanSubscriptionRow
	Users                      []servicePlanUserRow
	CustomerCloudAccounts      []servicePlanCustomerCloudAccountRow
	DeploymentsCount           int
	ActiveSubscriptionsCount   int
	UniqueUsersCount           int
	CustomerCloudAccountsCount int
	Err                        string
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
	RootUserID    string
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

type servicePlanCustomerCloudAccountRow struct {
	InstanceID     string
	CloudProvider  string
	Status         string
	SubscriptionID string
	CustomerEmail  string
	Resource       string
	Region         string
}

type servicePlanBrowserLoader interface {
	LoadEnvironmentDetails(context.Context, string, servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error)
}

type servicePlanBrowserDeploymentLauncher interface {
	LoadDeploymentForm(context.Context, string, servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error)
	LaunchDeployment(context.Context, string, servicePlanDeploymentLaunchRequest) (string, error)
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
	loadingDeploymentForm  bool
	deploymentForm         *servicePlanDeploymentFormState
	loadDeploymentForm     func(servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error)
	launchDeployment       func(servicePlanDeploymentLaunchRequest) (string, error)
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
	isLastChild  bool
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

type servicePlanDeploymentFormLoadedMsg struct {
	form servicePlanDeploymentForm
	err  error
}

type servicePlanDeploymentLaunchedMsg struct {
	instanceID string
	err        error
}

type servicePlanDeploymentForm struct {
	Environment              servicePlanBrowserEnvironment
	Version                  string
	ServiceProviderID        string
	ServiceURLKey            string
	ServiceAPIVersion        string
	ServiceEnvironmentURLKey string
	ServiceModelURLKey       string
	ProductTierURLKey        string
	Resources                []servicePlanDeploymentResource
	CloudProviders           []string
	RegionsByCloud           map[string][]string
	Parameters               []servicePlanDeploymentParameter
	ParametersByResource     map[string][]servicePlanDeploymentParameter
	CustomerCloudAccounts    []servicePlanCustomerCloudAccountRow
	Customers                []servicePlanDeploymentCustomer
	Subscriptions            []servicePlanSubscriptionRow
	RequiresCustomerAccount  bool
}

type servicePlanDeploymentResource struct {
	ID     string
	Name   string
	URLKey string
}

type servicePlanDeploymentParameter struct {
	Key          string
	DisplayName  string
	Description  string
	Type         string
	Required     bool
	IsList       bool
	Custom       bool
	DefaultValue string
	Options      []string
}

type servicePlanDeploymentCustomer struct {
	UserID  string
	Email   string
	Name    string
	OrgName string
	Self    bool
}

type servicePlanDeploymentLaunchRequest struct {
	Form              servicePlanDeploymentForm
	Customer          servicePlanDeploymentCustomer
	ResourceName      string
	CloudProvider     string
	Region            string
	CustomerAccountID string
	SubscriptionID    string
	Params            map[string]any
}

type servicePlanDeploymentFieldKind int

const (
	servicePlanDeploymentFieldParameter servicePlanDeploymentFieldKind = iota
)

type servicePlanDeploymentFormField struct {
	Kind        servicePlanDeploymentFieldKind
	Key         string
	Label       string
	Required    bool
	Type        string
	IsList      bool
	Description string
	Options     []string
	Input       textinput.Model
}

type servicePlanDeploymentFormState struct {
	Form                servicePlanDeploymentForm
	Steps               []servicePlanDeploymentWizardStep
	StepIndex           int
	SelectionCursor     int
	ParamFields         []servicePlanDeploymentFormField
	ParamCursor         int
	SelectedCustomer    servicePlanDeploymentCustomer
	ResourceName        string
	CloudProvider       string
	Region              string
	CustomerAccountID   string
	ParameterResourceID string
	ParamValues         map[string]string
	Launching           bool
	InstanceID          string
	Result              string
	Err                 string
}

type servicePlanDeploymentWizardStep string

const (
	servicePlanDeploymentStepCustomer     servicePlanDeploymentWizardStep = "customer"
	servicePlanDeploymentStepResource     servicePlanDeploymentWizardStep = "resource"
	servicePlanDeploymentStepCloud        servicePlanDeploymentWizardStep = "cloud"
	servicePlanDeploymentStepCloudAccount servicePlanDeploymentWizardStep = "cloud-account"
	servicePlanDeploymentStepRegion       servicePlanDeploymentWizardStep = "region"
	servicePlanDeploymentStepCustomParams servicePlanDeploymentWizardStep = "custom-params"
	servicePlanDeploymentStepSystemParams servicePlanDeploymentWizardStep = "system-params"
	servicePlanDeploymentStepReview       servicePlanDeploymentWizardStep = "review"
	servicePlanDeploymentStepComplete     servicePlanDeploymentWizardStep = "complete"
)

type servicePlanDeploymentWizardOption struct {
	Label       string
	Description string
	Value       string
	Customer    servicePlanDeploymentCustomer
	Account     servicePlanCustomerCloudAccountRow
}

func newServicePlanBrowserDelegate() servicePlanBrowserDelegate {
	delegate := servicePlanBrowserDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	delegate.SetHeight(1)
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
	if isSelected {
		titleStyle = titleStyle.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	}

	title := servicePlanBrowserLeftItemTreePrefix(browserItem) + browserItem.title

	renderedTitle := titleStyle.Render(title)
	if browserItem.hostingBadge.Label != "" {
		renderedTitle = lipgloss.JoinHorizontal(lipgloss.Center, renderedTitle, " ", renderServicePlanHostingBadge(browserItem.hostingBadge))
	}

	fmt.Fprint(writer, renderedTitle)
}

func servicePlanBrowserLeftItemTreePrefix(item servicePlanBrowserLeftItem) string {
	if item.isService {
		if item.expanded {
			return "- "
		}
		return "+ "
	}

	if item.level > 0 {
		if item.isLastChild {
			return "  └─ "
		}
		return "  ├─ "
	}

	return ""
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
		if launcher, ok := loader.(servicePlanBrowserDeploymentLauncher); ok {
			model.loadDeploymentForm = func(env servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error) {
				return launcher.LoadDeploymentForm(ctx, token, env)
			}
			model.launchDeployment = func(request servicePlanDeploymentLaunchRequest) (string, error) {
				return launcher.LaunchDeployment(ctx, token, request)
			}
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
		if !m.hasLoadingDetails() && !m.loadingDeploymentForm {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.syncViewportContent()
		return m, cmd
	case servicePlanBrowserDetailsLoadedMsg:
		delete(m.loadingDetails, msg.cacheKey)
		m.detailCache[msg.cacheKey] = msg.detail
		if m.statusMessage == "Refreshing plan details..." {
			m.statusMessage = "Plan details refreshed"
		}
		m.syncViewportContent()
		return m, nil
	case servicePlanDeploymentFormLoadedMsg:
		if !m.loadingDeploymentForm {
			return m, nil
		}
		m.loadingDeploymentForm = false
		if msg.err != nil {
			m.statusMessage = "Failed to load deployment form: " + msg.err.Error()
			return m, nil
		}
		formState := newServicePlanDeploymentFormState(msg.form, spMax(m.detailPanelWidth-8, 40))
		m.deploymentForm = &formState
		return m, nil
	case servicePlanDeploymentLaunchedMsg:
		if m.deploymentForm == nil {
			return m, nil
		}
		m.deploymentForm.Launching = false
		if msg.err != nil {
			m.deploymentForm.Err = msg.err.Error()
			return m, nil
		}
		m.deploymentForm.Err = ""
		m.deploymentForm.InstanceID = strings.TrimSpace(msg.instanceID)
		m.deploymentForm.Result = "Deployment launched: " + emptyValue(msg.instanceID)
		m.deploymentForm.StepIndex = len(m.deploymentForm.Steps)
		m.deploymentForm.Steps = append(m.deploymentForm.Steps, servicePlanDeploymentStepComplete)
		m.statusMessage = m.deploymentForm.Result
		return m, m.refreshSelectedDetails()
	case tea.KeyMsg:
		if m.deploymentForm != nil {
			return m.updateDeploymentForm(msg)
		}
		if m.loadingDeploymentForm && msg.String() == "esc" {
			m.loadingDeploymentForm = false
			m.statusMessage = ""
			return m, nil
		}
		if m.modal != nil {
			return m.updateModal(msg), nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "d":
			return m, m.openDeploymentForm()
		case "r":
			return m, m.refreshSelectedDetails()
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
			if m.toggleSelectedService() {
				return m, nil
			}
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
	if m.deploymentForm != nil {
		return m.renderDeploymentForm()
	}
	if m.loadingDeploymentForm {
		return m.renderDeploymentFormLoading()
	}
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
		return "Use d to launch a deployment, r to refresh. Use tab or shift+tab to switch environments. Use esc or left to return to the plan list."
	}
	return "Use d to launch a deployment, r to refresh. Plan details update as the selector moves."
}

func (m servicePlanBrowserModel) helpLine() string {
	if m.focus == servicePlanBrowserFocusDetails {
		return "d: deploy  r: refresh  tab/shift+tab: environment  ↑/↓: detail rows  enter: open row  esc/←: focus plans  q: quit"
	}
	return "d: deploy  r: refresh  tab/shift+tab: environment  ↑/↓: navigate plans  enter/→: details/open  ←/→: expand/collapse services  q: quit"
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

func renderServicePlanBrowserHeader(_ servicePlanBrowserCatalog) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Service Plan Browser")
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
		Padding(0, 1).
		Bold(true)
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

	rows := []servicePlanBrowserDetailRow{
		{Label: "Deployment model", Value: emptyValue(detail.DeploymentModel)},
		{Label: "Enabled features", Value: joinOrNone(detail.EnabledFeatures)},
		{Label: "Clouds", Value: joinOrNone(detail.Clouds)},
		{Label: "Regions", Value: joinOrNone(detail.Regions)},
		{Label: "Deployments", Value: fmt.Sprintf("%d", detail.DeploymentsCount), ModalKind: servicePlanBrowserModalDeployments},
		{Label: "Subscriptions", Value: fmt.Sprintf("%d", detail.ActiveSubscriptionsCount), ModalKind: servicePlanBrowserModalSubscriptions},
		{Label: "Users", Value: fmt.Sprintf("%d", detail.UniqueUsersCount), ModalKind: servicePlanBrowserModalUsers},
	}
	if servicePlanEnvironmentRequiresCustomerAccount(*env) || detail.CustomerCloudAccountsCount > 0 {
		rows = append(rows, servicePlanBrowserDetailRow{
			Label:     "Cloud accounts",
			Value:     fmt.Sprintf("%d connected", detail.CustomerCloudAccountsCount),
			ModalKind: servicePlanBrowserModalCloudAccounts,
		})
	}
	return rows
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

func (m *servicePlanBrowserModel) toggleSelectedService() bool {
	selected := m.selectedLeftItem()
	if selected == nil || !selected.expandable {
		return false
	}

	m.expanded[selected.serviceIndex] = !selected.expanded
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
	case servicePlanBrowserModalCloudAccounts:
		for _, account := range detail.CustomerCloudAccounts {
			text := fmt.Sprintf(
				"%s | %s | %s | %s | %s | %s",
				emptyValue(account.InstanceID),
				emptyValue(account.CloudProvider),
				emptyValue(account.Status),
				emptyValue(account.Region),
				emptyValue(account.CustomerEmail),
				emptyValue(account.SubscriptionID),
			)
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
		for index := range planItems {
			planItems[index].isLastChild = index == len(planItems)-1
		}
		if len(planItems) == 0 {
			continue
		}

		items = append(items, servicePlanBrowserLeftItem{
			key:          serviceKey,
			title:        service.Name,
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

func (m *servicePlanBrowserModel) refreshSelectedDetails() tea.Cmd {
	env := m.selectedEnvironment()
	if env == nil {
		m.statusMessage = "No environment selected to refresh"
		return nil
	}
	key := env.cacheKey()
	delete(m.detailCache, key)
	delete(m.loadingDetails, key)
	m.statusMessage = "Refreshing plan details..."
	cmd := m.requestSelectedDetailsLoad()
	m.syncViewportContent()
	return cmd
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

func (m *servicePlanBrowserModel) openDeploymentForm() tea.Cmd {
	selected := m.selectedLeftItem()
	if selected == nil || selected.isService {
		m.statusMessage = "Select a plan before launching a deployment"
		return nil
	}
	env := m.selectedEnvironment()
	if env == nil {
		m.statusMessage = "No environment is selected for this plan"
		return nil
	}
	if m.loadDeploymentForm == nil {
		m.statusMessage = "Deployment launch is not available in this view"
		return nil
	}

	m.loadingDeploymentForm = true
	m.statusMessage = "Loading deployment form..."
	return tea.Batch(m.spinner.Tick, m.loadDeploymentFormCmd(*env))
}

func (m servicePlanBrowserModel) loadDeploymentFormCmd(env servicePlanBrowserEnvironment) tea.Cmd {
	if m.loadDeploymentForm == nil {
		return nil
	}

	return func() tea.Msg {
		form, err := m.loadDeploymentForm(env)
		return servicePlanDeploymentFormLoadedMsg{form: form, err: err}
	}
}

func (m servicePlanBrowserModel) launchDeploymentCmd(request servicePlanDeploymentLaunchRequest) tea.Cmd {
	if m.launchDeployment == nil {
		return nil
	}

	return func() tea.Msg {
		instanceID, err := m.launchDeployment(request)
		return servicePlanDeploymentLaunchedMsg{instanceID: instanceID, err: err}
	}
}

func (m servicePlanBrowserModel) updateDeploymentForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.deploymentForm == nil {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.deploymentForm = nil
		m.statusMessage = ""
		return m, nil
	case "left", "backspace":
		m.deploymentForm.previousStep(spMax(m.detailPanelWidth-8, 40))
		return m, nil
	case "up":
		if m.deploymentForm.currentStepIsParameters() && m.deploymentForm.moveCurrentParameterOption(-1) {
			return m, nil
		}
		m.deploymentForm.moveWizardCursor(-1)
		return m, nil
	case "down":
		if m.deploymentForm.currentStepIsParameters() && m.deploymentForm.moveCurrentParameterOption(1) {
			return m, nil
		}
		m.deploymentForm.moveWizardCursor(1)
		return m, nil
	case "tab":
		if m.deploymentForm.currentStepIsParameters() {
			m.deploymentForm.moveWizardCursor(1)
			return m, nil
		}
	case "shift+tab":
		if m.deploymentForm.currentStepIsParameters() {
			m.deploymentForm.moveWizardCursor(-1)
			return m, nil
		}
	case "right", "enter":
		return m.advanceDeploymentWizard()
	}

	if m.deploymentForm.Launching || !m.deploymentForm.currentStepIsParameters() || len(m.deploymentForm.ParamFields) == 0 {
		return m, nil
	}

	field := &m.deploymentForm.ParamFields[m.deploymentForm.ParamCursor]
	if len(field.Options) > 0 {
		return m, nil
	}
	updated, cmd := field.Input.Update(msg)
	field.Input = updated
	m.deploymentForm.Err = ""
	return m, cmd
}

func (m servicePlanBrowserModel) advanceDeploymentWizard() (tea.Model, tea.Cmd) {
	if m.deploymentForm == nil {
		return m, nil
	}
	if m.deploymentForm.currentStep() == servicePlanDeploymentStepComplete {
		m.deploymentForm = nil
		m.statusMessage = ""
		return m, nil
	}
	if m.deploymentForm.currentStep() == servicePlanDeploymentStepReview {
		request, err := m.deploymentForm.launchRequest()
		if err != nil {
			m.deploymentForm.Err = err.Error()
			return m, nil
		}
		if m.launchDeployment == nil {
			m.deploymentForm.Err = "deployment launch is not available"
			return m, nil
		}
		m.deploymentForm.Err = ""
		m.deploymentForm.Result = ""
		m.deploymentForm.Launching = true
		return m, m.launchDeploymentCmd(request)
	}

	if err := m.deploymentForm.advanceStep(spMax(m.detailPanelWidth-8, 40)); err != nil {
		m.deploymentForm.Err = err.Error()
		return m, nil
	}
	return m, nil
}

func newServicePlanDeploymentFormState(form servicePlanDeploymentForm, width int) servicePlanDeploymentFormState {
	customers := servicePlanDeploymentCustomerOptions(form)
	resourceName := firstString(servicePlanDeploymentResourceOptions(form.Resources))
	cloudProvider := firstString(form.CloudProviders)
	state := servicePlanDeploymentFormState{
		Form:             form,
		SelectedCustomer: firstServicePlanDeploymentCustomer(customers),
		ResourceName:     resourceName,
		CloudProvider:    cloudProvider,
		Region:           firstString(form.RegionsByCloud[cloudProvider]),
		ParamValues:      map[string]string{},
	}
	state.Steps = servicePlanDeploymentWizardSteps(form)
	state.CustomerAccountID = state.firstCloudAccountID()
	state.prepareCurrentStep(width)
	return state
}

func servicePlanDeploymentWizardSteps(form servicePlanDeploymentForm) []servicePlanDeploymentWizardStep {
	steps := make([]servicePlanDeploymentWizardStep, 0, 8)
	if servicePlanEnvironmentIsProduction(form.Environment) {
		steps = append(steps, servicePlanDeploymentStepCustomer)
	}
	steps = append(steps, servicePlanDeploymentStepResource)
	steps = append(steps, servicePlanDeploymentStepCloud)
	if form.RequiresCustomerAccount {
		steps = append(steps, servicePlanDeploymentStepCloudAccount)
	}
	steps = append(steps, servicePlanDeploymentStepRegion)
	if servicePlanDeploymentFormHasParameters(form, true) {
		steps = append(steps, servicePlanDeploymentStepCustomParams)
	}
	if servicePlanDeploymentFormHasParameters(form, false) {
		steps = append(steps, servicePlanDeploymentStepSystemParams)
	}
	steps = append(steps, servicePlanDeploymentStepReview)
	return steps
}

func servicePlanDeploymentFormHasParameters(form servicePlanDeploymentForm, custom bool) bool {
	for _, resource := range form.Resources {
		resourceName := strings.TrimSpace(resource.Name)
		if resourceName == "" {
			resourceName = strings.TrimSpace(resource.ID)
		}
		if len(servicePlanDeploymentParametersByCustomFlag(form, resourceName, custom)) > 0 {
			return true
		}
	}
	return false
}

func (s *servicePlanDeploymentFormState) prepareCurrentStep(width int) {
	s.Err = ""
	s.SelectionCursor = s.selectedOptionIndex()
	if s.currentStepIsParameters() {
		_, s.ParameterResourceID = servicePlanDeploymentParametersForResource(s.Form, s.ResourceName)
		s.ParamFields = s.buildParameterFields(s.currentStep(), width)
		s.ParamCursor = spClamp(s.ParamCursor, spMax(0, len(s.ParamFields)-1))
		s.syncFocus()
		return
	}
	s.ParameterResourceID = ""
	s.ParamFields = nil
	s.ParamCursor = 0
}

func (s servicePlanDeploymentFormState) buildParameterFields(step servicePlanDeploymentWizardStep, width int) []servicePlanDeploymentFormField {
	custom := step == servicePlanDeploymentStepCustomParams
	parameters := servicePlanDeploymentParametersByCustomFlag(s.Form, s.ResourceName, custom)
	fields := make([]servicePlanDeploymentFormField, 0, len(parameters))
	fieldWidth := spMin(spMax(width-28, 24), 64)
	for _, parameter := range parameters {
		if strings.EqualFold(parameter.Key, servicePlanCustomerAccountConfigIDParamKey) {
			continue
		}
		label := strings.TrimSpace(parameter.DisplayName)
		if label == "" {
			label = parameter.Key
		}
		value := parameter.DefaultValue
		if s.ParamValues != nil {
			if existing, ok := s.ParamValues[parameter.Key]; ok {
				value = existing
			}
		}
		input := textinput.New()
		input.CharLimit = 0
		input.Width = fieldWidth
		input.Prompt = ""
		input.SetValue(value)
		fields = append(fields, servicePlanDeploymentFormField{
			Kind:        servicePlanDeploymentFieldParameter,
			Key:         parameter.Key,
			Label:       label,
			Required:    parameter.Required,
			Type:        parameter.Type,
			IsList:      parameter.IsList,
			Description: parameter.Description,
			Options:     parameter.Options,
			Input:       input,
		})
	}
	return fields
}

func (s *servicePlanDeploymentFormState) advanceStep(width int) error {
	if s.Launching {
		return nil
	}
	if s.currentStepIsParameters() {
		if err := s.storeCurrentParameterValues(); err != nil {
			return err
		}
	} else if err := s.applySelectedOption(); err != nil {
		return err
	}

	if s.StepIndex < len(s.Steps)-1 {
		s.StepIndex++
		s.prepareCurrentStep(width)
	}
	return nil
}

func (s *servicePlanDeploymentFormState) previousStep(width int) {
	if s.Launching {
		return
	}
	if s.currentStepIsParameters() {
		_ = s.storeCurrentParameterValues()
	}
	if s.StepIndex > 0 {
		s.StepIndex--
		s.prepareCurrentStep(width)
	}
}

func (s *servicePlanDeploymentFormState) moveWizardCursor(delta int) {
	if s.currentStepIsParameters() {
		if len(s.ParamFields) == 0 {
			s.ParamCursor = 0
			return
		}
		s.ParamCursor = (s.ParamCursor + delta + len(s.ParamFields)) % len(s.ParamFields)
		s.syncFocus()
		return
	}

	options := s.currentOptions()
	if len(options) == 0 {
		s.SelectionCursor = 0
		return
	}
	s.SelectionCursor = (s.SelectionCursor + delta + len(options)) % len(options)
}

func (s *servicePlanDeploymentFormState) moveCurrentParameterOption(delta int) bool {
	if !s.currentStepIsParameters() || len(s.ParamFields) == 0 {
		return false
	}
	field := &s.ParamFields[s.ParamCursor]
	if len(field.Options) == 0 {
		return false
	}

	current := strings.TrimSpace(field.Input.Value())
	index := -1
	for i, option := range field.Options {
		if strings.EqualFold(strings.TrimSpace(option), current) {
			index = i
			break
		}
	}
	next := 0
	if index >= 0 {
		next = (index + delta + len(field.Options)) % len(field.Options)
	} else if delta < 0 {
		next = len(field.Options) - 1
	}
	field.Input.SetValue(field.Options[next])
	s.Err = ""
	return true
}

func (s servicePlanDeploymentFormState) currentStep() servicePlanDeploymentWizardStep {
	if s.StepIndex < 0 || s.StepIndex >= len(s.Steps) {
		return servicePlanDeploymentStepReview
	}
	return s.Steps[s.StepIndex]
}

func (s servicePlanDeploymentFormState) currentStepIsParameters() bool {
	step := s.currentStep()
	return step == servicePlanDeploymentStepCustomParams || step == servicePlanDeploymentStepSystemParams
}

func (s servicePlanDeploymentFormState) currentStepNumber() int {
	return spMin(s.StepIndex+1, len(s.Steps))
}

func (s *servicePlanDeploymentFormState) syncFocus() {
	for i := range s.ParamFields {
		if i == s.ParamCursor {
			s.ParamFields[i].Input.Focus()
		} else {
			s.ParamFields[i].Input.Blur()
		}
	}
}

func (s *servicePlanDeploymentFormState) storeCurrentParameterValues() error {
	if s.ParamValues == nil {
		s.ParamValues = map[string]string{}
	}
	for _, field := range s.ParamFields {
		value := strings.TrimSpace(field.Input.Value())
		if field.Required && value == "" {
			return fmt.Errorf("%s is required", field.Label)
		}
		s.ParamValues[field.Key] = value
	}
	return nil
}

func (s *servicePlanDeploymentFormState) applySelectedOption() error {
	options := s.currentOptions()
	if len(options) == 0 {
		if s.currentStep() == servicePlanDeploymentStepCloudAccount {
			return fmt.Errorf("no connected cloud accounts for %s. Run: omnistrate-ctl account customer create --help", emptyValue(s.CloudProvider))
		}
		return nil
	}
	option := options[spClamp(s.SelectionCursor, len(options)-1)]
	switch s.currentStep() {
	case servicePlanDeploymentStepCustomer:
		s.SelectedCustomer = option.Customer
		s.CustomerAccountID = s.firstCloudAccountID()
	case servicePlanDeploymentStepResource:
		s.ResourceName = option.Value
		s.ParamFields = nil
	case servicePlanDeploymentStepCloud:
		s.CloudProvider = option.Value
		s.Region = firstString(s.Form.RegionsByCloud[s.CloudProvider])
		s.CustomerAccountID = s.firstCloudAccountID()
	case servicePlanDeploymentStepCloudAccount:
		s.CustomerAccountID = option.Value
	case servicePlanDeploymentStepRegion:
		s.Region = option.Value
	}
	s.SelectionCursor = 0
	return nil
}

func (s servicePlanDeploymentFormState) launchRequest() (servicePlanDeploymentLaunchRequest, error) {
	request := servicePlanDeploymentLaunchRequest{
		Form:              s.Form,
		Customer:          s.SelectedCustomer,
		ResourceName:      s.ResourceName,
		CloudProvider:     s.CloudProvider,
		Region:            s.Region,
		CustomerAccountID: s.CustomerAccountID,
		Params:            map[string]any{},
	}

	if s.currentStepIsParameters() {
		copyState := s
		if err := copyState.storeCurrentParameterValues(); err != nil {
			return request, err
		}
	}
	request.SubscriptionID = servicePlanSubscriptionIDForCustomer(s.Form.Subscriptions, s.SelectedCustomer)

	if _, ok := servicePlanDeploymentResourceByName(s.Form, request.ResourceName); !ok {
		return request, fmt.Errorf("resource %q is not available for this plan", request.ResourceName)
	}
	if request.CloudProvider != "" && !servicePlanContainsFold(s.Form.CloudProviders, request.CloudProvider) {
		return request, fmt.Errorf("cloud provider %q is not available for this plan", request.CloudProvider)
	}
	if request.Region != "" {
		regions := s.Form.RegionsByCloud[request.CloudProvider]
		if len(regions) > 0 && !servicePlanContainsFold(regions, request.Region) {
			return request, fmt.Errorf("region %q is not available for cloud provider %q", request.Region, request.CloudProvider)
		}
	}
	if s.Form.RequiresCustomerAccount {
		if request.CustomerAccountID == "" {
			return request, fmt.Errorf("customer cloud account is required for BYOC plans. Run: omnistrate-ctl account customer create --help")
		}
		if !servicePlanCustomerCloudAccountExists(s.filteredCloudAccounts(), request.CustomerAccountID) {
			return request, fmt.Errorf("customer cloud account %q is not connected to this plan", request.CustomerAccountID)
		}
	}

	for _, parameter := range servicePlanDeploymentParametersForSelectedResource(s.Form, request.ResourceName) {
		if strings.EqualFold(parameter.Key, servicePlanCustomerAccountConfigIDParamKey) {
			continue
		}
		value := strings.TrimSpace(s.ParamValues[parameter.Key])
		if value == "" {
			value = strings.TrimSpace(parameter.DefaultValue)
		}
		if value == "" && !parameter.Required {
			continue
		}
		if value == "" {
			return request, fmt.Errorf("%s is required", servicePlanDeploymentParameterLabel(parameter))
		}
		parsed, err := parseServicePlanDeploymentParamValue(value, parameter.Type, parameter.IsList)
		if err != nil {
			return request, fmt.Errorf("%s: %w", servicePlanDeploymentParameterLabel(parameter), err)
		}
		request.Params[parameter.Key] = parsed
	}

	return request, nil
}

func parseServicePlanDeploymentParamValue(value, valueType string, isList bool) (any, error) {
	value = strings.TrimSpace(value)
	if isList {
		if strings.HasPrefix(value, "[") {
			var parsed any
			if err := json.Unmarshal([]byte(value), &parsed); err != nil {
				return nil, err
			}
			return parsed, nil
		}
		if value == "" {
			return []string{}, nil
		}
		parts := strings.Split(value, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values, nil
	}

	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "boolean":
		return strconv.ParseBool(value)
	case "int", "int32", "int64", "integer":
		return strconv.ParseInt(value, 10, 64)
	case "float", "float32", "float64", "double", "number":
		return strconv.ParseFloat(value, 64)
	case "object", "json", "map":
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		return value, nil
	}
}

func (m servicePlanBrowserModel) renderDeploymentFormLoading() string {
	env := m.selectedEnvironment()
	title := "Launch Deployment"
	if plan := m.selectedPlan(); plan != nil && env != nil {
		title = plan.ServiceName + " / " + plan.Name + " / " + env.Name
	}
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(title),
		"",
		m.spinner.View()+" Loading deployment form...",
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("esc: cancel"),
	)
	return servicePlanBrowserPanelStyle(lipgloss.Color("117")).Width(spMax(m.width-4, 80)).Render(content)
}

func (m servicePlanBrowserModel) renderDeploymentForm() string {
	form := m.deploymentForm
	if form == nil {
		return ""
	}

	width := spMax(m.width-4, 80)
	height := spMax(m.height-6, 16)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	selectedLabelStyle := labelStyle.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	lines := []string{
		titleStyle.Render("Launch Deployment"),
		metaStyle.Render(fmt.Sprintf("%s / %s / %s", form.Form.Environment.ServiceName, form.Form.Environment.PlanName, form.Form.Environment.Name)),
		metaStyle.Render(fmt.Sprintf("Step %d/%d: %s", form.currentStepNumber(), len(form.Steps), servicePlanDeploymentStepTitle(form.currentStep()))),
		metaStyle.Render("Version: " + emptyValue(form.Form.Version)),
		"",
	}

	switch {
	case form.currentStep() == servicePlanDeploymentStepComplete:
		lines = append(lines, form.renderDeploymentCompleteLines(valueStyle)...)
	case form.currentStep() == servicePlanDeploymentStepReview:
		lines = append(lines, form.renderDeploymentReviewLines(labelStyle, valueStyle)...)
	case form.currentStepIsParameters():
		lines = append(lines, form.renderDeploymentParameterLines(height, labelStyle, selectedLabelStyle, metaStyle)...)
	default:
		lines = append(lines, form.renderDeploymentOptionLines(height, selectedLabelStyle, valueStyle, metaStyle)...)
	}

	if form.Err != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: "+form.Err))
	}
	if form.Launching {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render("Launching deployment..."))
	}

	help := "enter/→: next  ↑/↓: select  ←: back  esc: close"
	if form.currentStepIsParameters() {
		help = "enter/→: next  tab/shift+tab: fields  ↑/↓: select option or move fields  ←: back  esc: close"
	}
	if form.currentStep() == servicePlanDeploymentStepReview {
		help = "enter: launch  ←: back  esc: close"
	}
	if form.currentStep() == servicePlanDeploymentStepComplete {
		help = "enter/esc: close"
	}
	lines = append(lines, "", metaStyle.Render(help))

	return servicePlanBrowserPanelStyle(lipgloss.Color("117")).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (s servicePlanDeploymentFormState) renderDeploymentOptionLines(height int, selectedStyle, valueStyle, metaStyle lipgloss.Style) []string {
	options := s.currentOptions()
	if len(options) == 0 {
		if s.currentStep() == servicePlanDeploymentStepCloudAccount {
			return []string{
				"No connected cloud accounts match this selection.",
				"",
				metaStyle.Render("Run: omnistrate-ctl account customer create --help"),
			}
		}
		return []string{"No options available."}
	}

	visibleRows := spMax(height-10, 4)
	start := 0
	if s.SelectionCursor >= visibleRows {
		start = s.SelectionCursor - visibleRows + 1
	}
	end := spMin(len(options), start+visibleRows)
	lines := make([]string, 0, visibleRows+1)
	for i := start; i < end; i++ {
		option := options[i]
		prefix := "  "
		style := valueStyle
		if i == s.SelectionCursor {
			prefix = "▸ "
			style = selectedStyle
		}
		lines = append(lines, style.Render(prefix+option.Label))
		if option.Description != "" {
			lines = append(lines, metaStyle.Render("    "+option.Description))
		}
	}
	return lines
}

func (s servicePlanDeploymentFormState) renderDeploymentParameterLines(height int, labelStyle, selectedLabelStyle, metaStyle lipgloss.Style) []string {
	if len(s.ParamFields) == 0 {
		return []string{"No parameters required for this step."}
	}

	visibleFields := spMax(height-10, 4)
	start := 0
	if s.ParamCursor >= visibleFields {
		start = s.ParamCursor - visibleFields + 1
	}
	end := spMin(len(s.ParamFields), start+visibleFields)
	lines := make([]string, 0, visibleFields+1)
	for i := start; i < end; i++ {
		field := s.ParamFields[i]
		prefix := "  "
		style := labelStyle
		if i == s.ParamCursor {
			prefix = "▸ "
			style = selectedLabelStyle
		}
		required := ""
		if field.Required {
			required = " *"
		}
		lines = append(lines, style.Render(prefix+field.Label+required+": ")+field.Input.View())
		if len(field.Options) > 0 {
			lines = append(lines, metaStyle.Render("    "+servicePlanDeploymentParameterOptionsLine(field)))
		} else if field.Description != "" {
			lines = append(lines, metaStyle.Render("    "+field.Description))
		}
	}
	return lines
}

func servicePlanDeploymentParameterOptionsLine(field servicePlanDeploymentFormField) string {
	current := strings.TrimSpace(field.Input.Value())
	parts := make([]string, 0, len(field.Options))
	for _, option := range field.Options {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}
		if strings.EqualFold(option, current) {
			parts = append(parts, "["+option+"]")
			continue
		}
		parts = append(parts, option)
	}
	if current == "" {
		return "options: " + strings.Join(parts, "  ")
	}
	return "selected: " + strings.Join(parts, "  ")
}

func (s servicePlanDeploymentFormState) renderDeploymentReviewLines(labelStyle, valueStyle lipgloss.Style) []string {
	lines := []string{
		labelStyle.Render("Deploy for: ") + valueStyle.Render(servicePlanDeploymentCustomerLabel(s.SelectedCustomer)),
		labelStyle.Render("Resource: ") + valueStyle.Render(emptyValue(s.ResourceName)),
		labelStyle.Render("Cloud: ") + valueStyle.Render(emptyValue(s.CloudProvider)),
		labelStyle.Render("Region: ") + valueStyle.Render(emptyValue(s.Region)),
	}
	if s.Form.RequiresCustomerAccount {
		lines = append(lines, labelStyle.Render("Cloud account: ")+valueStyle.Render(emptyValue(s.CustomerAccountID)))
	}
	if subscriptionID := servicePlanSubscriptionIDForCustomer(s.Form.Subscriptions, s.SelectedCustomer); subscriptionID != "" {
		lines = append(lines, labelStyle.Render("Subscription: ")+valueStyle.Render(subscriptionID))
	} else if !s.SelectedCustomer.Self {
		lines = append(lines, labelStyle.Render("Subscription: ")+valueStyle.Render("will be created on launch"))
	}

	params := servicePlanDeploymentParametersForSelectedResource(s.Form, s.ResourceName)
	if len(params) > 0 {
		lines = append(lines, "", labelStyle.Render("Parameters"))
		for _, parameter := range params {
			if strings.EqualFold(parameter.Key, servicePlanCustomerAccountConfigIDParamKey) {
				continue
			}
			value := strings.TrimSpace(s.ParamValues[parameter.Key])
			if value == "" {
				value = strings.TrimSpace(parameter.DefaultValue)
			}
			lines = append(lines, valueStyle.Render("  "+servicePlanDeploymentParameterLabel(parameter)+": "+emptyValue(value)))
		}
	}
	return lines
}

func (s servicePlanDeploymentFormState) renderDeploymentCompleteLines(valueStyle lipgloss.Style) []string {
	instanceID := emptyValue(s.InstanceID)
	lines := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("Deployment launched: " + instanceID),
		"",
		"Next commands:",
		valueStyle.Render("  omnistrate-ctl instance describe " + instanceID),
		valueStyle.Render("  omnistrate-ctl instance list-endpoints " + instanceID),
		valueStyle.Render("  omnistrate-ctl instance debug " + instanceID),
	}
	return lines
}

func (s servicePlanDeploymentFormState) currentOptions() []servicePlanDeploymentWizardOption {
	switch s.currentStep() {
	case servicePlanDeploymentStepCustomer:
		customers := servicePlanDeploymentCustomerOptions(s.Form)
		options := make([]servicePlanDeploymentWizardOption, 0, len(customers))
		for _, customer := range customers {
			options = append(options, servicePlanDeploymentWizardOption{
				Label:       servicePlanDeploymentCustomerLabel(customer),
				Description: servicePlanDeploymentCustomerDescription(customer),
				Value:       customer.Email,
				Customer:    customer,
			})
		}
		return options
	case servicePlanDeploymentStepResource:
		options := make([]servicePlanDeploymentWizardOption, 0, len(s.Form.Resources))
		for _, resource := range s.Form.Resources {
			label := strings.TrimSpace(resource.Name)
			if label == "" {
				label = resource.ID
			}
			options = append(options, servicePlanDeploymentWizardOption{
				Label: label,
				Value: label,
			})
		}
		return options
	case servicePlanDeploymentStepCloud:
		options := make([]servicePlanDeploymentWizardOption, 0, len(s.Form.CloudProviders))
		for _, cloud := range s.Form.CloudProviders {
			options = append(options, servicePlanDeploymentWizardOption{Label: cloud, Value: cloud})
		}
		return options
	case servicePlanDeploymentStepCloudAccount:
		accounts := s.filteredCloudAccounts()
		options := make([]servicePlanDeploymentWizardOption, 0, len(accounts))
		for _, account := range accounts {
			label := account.InstanceID
			descriptionParts := []string{account.CloudProvider, account.Region, account.Status, account.CustomerEmail}
			options = append(options, servicePlanDeploymentWizardOption{
				Label:       label,
				Description: strings.Join(nonEmptyStrings(descriptionParts), " | "),
				Value:       account.InstanceID,
				Account:     account,
			})
		}
		return options
	case servicePlanDeploymentStepRegion:
		regions := s.Form.RegionsByCloud[s.CloudProvider]
		options := make([]servicePlanDeploymentWizardOption, 0, len(regions))
		for _, region := range regions {
			options = append(options, servicePlanDeploymentWizardOption{Label: region, Value: region})
		}
		return options
	default:
		return nil
	}
}

func (s servicePlanDeploymentFormState) selectedOptionIndex() int {
	options := s.currentOptions()
	selected := ""
	switch s.currentStep() {
	case servicePlanDeploymentStepCustomer:
		selected = s.SelectedCustomer.Email
		if s.SelectedCustomer.Self {
			selected = "self"
		}
	case servicePlanDeploymentStepResource:
		selected = s.ResourceName
	case servicePlanDeploymentStepCloud:
		selected = s.CloudProvider
	case servicePlanDeploymentStepCloudAccount:
		selected = s.CustomerAccountID
	case servicePlanDeploymentStepRegion:
		selected = s.Region
	}
	for i, option := range options {
		if strings.EqualFold(option.Value, selected) ||
			strings.EqualFold(option.Label, selected) ||
			(option.Customer.Self && selected == "self") {
			return i
		}
	}
	return 0
}

func (s servicePlanDeploymentFormState) filteredCloudAccounts() []servicePlanCustomerCloudAccountRow {
	accounts := make([]servicePlanCustomerCloudAccountRow, 0, len(s.Form.CustomerCloudAccounts))
	selectedSubscriptionID := servicePlanSubscriptionIDForCustomer(s.Form.Subscriptions, s.SelectedCustomer)
	for _, account := range s.Form.CustomerCloudAccounts {
		if s.CloudProvider != "" && !strings.EqualFold(account.CloudProvider, s.CloudProvider) {
			continue
		}
		if !s.SelectedCustomer.Self {
			switch {
			case selectedSubscriptionID != "" && account.SubscriptionID != "":
				if !strings.EqualFold(account.SubscriptionID, selectedSubscriptionID) {
					continue
				}
			case s.SelectedCustomer.Email != "":
				if !strings.EqualFold(account.CustomerEmail, s.SelectedCustomer.Email) {
					continue
				}
			default:
				continue
			}
		}
		accounts = append(accounts, account)
	}
	return accounts
}

func (s servicePlanDeploymentFormState) firstCloudAccountID() string {
	for _, account := range s.filteredCloudAccounts() {
		if strings.TrimSpace(account.InstanceID) != "" {
			return strings.TrimSpace(account.InstanceID)
		}
	}
	return ""
}

func servicePlanDeploymentCustomerOptions(form servicePlanDeploymentForm) []servicePlanDeploymentCustomer {
	customers := []servicePlanDeploymentCustomer{{Self: true, Name: "Self"}}
	customers = append(customers, form.Customers...)
	return customers
}

func firstServicePlanDeploymentCustomer(customers []servicePlanDeploymentCustomer) servicePlanDeploymentCustomer {
	if len(customers) == 0 {
		return servicePlanDeploymentCustomer{Self: true, Name: "Self"}
	}
	return customers[0]
}

func servicePlanDeploymentCustomerLabel(customer servicePlanDeploymentCustomer) string {
	if customer.Self {
		return "Self"
	}
	name := strings.TrimSpace(customer.Name)
	email := strings.TrimSpace(customer.Email)
	if name == "" {
		return emptyValue(email)
	}
	if email == "" {
		return name
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

func servicePlanDeploymentCustomerDescription(customer servicePlanDeploymentCustomer) string {
	if customer.Self {
		return "Deploy as the current service provider user"
	}
	parts := nonEmptyStrings([]string{customer.OrgName, customer.UserID})
	return strings.Join(parts, " | ")
}

func servicePlanDeploymentStepTitle(step servicePlanDeploymentWizardStep) string {
	switch step {
	case servicePlanDeploymentStepCustomer:
		return "Deploy on Behalf Of"
	case servicePlanDeploymentStepResource:
		return "Resource"
	case servicePlanDeploymentStepCloud:
		return "Cloud Provider"
	case servicePlanDeploymentStepCloudAccount:
		return "Customer Cloud Account"
	case servicePlanDeploymentStepRegion:
		return "Region"
	case servicePlanDeploymentStepCustomParams:
		return "Customer Parameters"
	case servicePlanDeploymentStepSystemParams:
		return "Advanced Parameters"
	case servicePlanDeploymentStepReview:
		return "Review"
	case servicePlanDeploymentStepComplete:
		return "Complete"
	default:
		return "Deploy"
	}
}

func servicePlanDeploymentResourceOptions(resources []servicePlanDeploymentResource) []string {
	options := make([]string, 0, len(resources))
	for _, resource := range resources {
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			name = strings.TrimSpace(resource.ID)
		}
		if name != "" {
			options = append(options, name)
		}
	}
	return options
}

func servicePlanCustomerCloudAccountExists(accounts []servicePlanCustomerCloudAccountRow, instanceID string) bool {
	for _, account := range accounts {
		if strings.EqualFold(strings.TrimSpace(account.InstanceID), strings.TrimSpace(instanceID)) {
			return true
		}
	}
	return false
}

func servicePlanDeploymentParametersForSelectedResource(form servicePlanDeploymentForm, resourceName string) []servicePlanDeploymentParameter {
	parameters, _ := servicePlanDeploymentParametersForResource(form, resourceName)
	return parameters
}

func servicePlanDeploymentParametersByCustomFlag(form servicePlanDeploymentForm, resourceName string, custom bool) []servicePlanDeploymentParameter {
	parameters := servicePlanDeploymentParametersForSelectedResource(form, resourceName)
	filtered := make([]servicePlanDeploymentParameter, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter.Custom == custom {
			filtered = append(filtered, parameter)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Key < filtered[j].Key
	})
	return filtered
}

func servicePlanDeploymentParameterLabel(parameter servicePlanDeploymentParameter) string {
	label := strings.TrimSpace(parameter.DisplayName)
	if label == "" {
		label = parameter.Key
	}
	return label
}

func servicePlanSubscriptionIDForCustomer(subscriptions []servicePlanSubscriptionRow, customer servicePlanDeploymentCustomer) string {
	if customer.Self {
		return ""
	}
	for _, subscription := range subscriptions {
		if customer.UserID != "" && strings.EqualFold(subscription.RootUserID, customer.UserID) {
			return strings.TrimSpace(subscription.ID)
		}
		if customer.Email != "" && strings.EqualFold(subscription.RootUserEmail, customer.Email) {
			return strings.TrimSpace(subscription.ID)
		}
	}
	return ""
}

func servicePlanDeploymentResourceByName(form servicePlanDeploymentForm, resourceName string) (servicePlanDeploymentResource, bool) {
	for _, resource := range form.Resources {
		if strings.EqualFold(resource.Name, resourceName) ||
			strings.EqualFold(resource.ID, resourceName) ||
			strings.EqualFold(resource.URLKey, resourceName) {
			return resource, true
		}
	}
	return servicePlanDeploymentResource{}, false
}

func servicePlanDeploymentParametersForResource(form servicePlanDeploymentForm, resourceName string) ([]servicePlanDeploymentParameter, string) {
	if form.ParametersByResource == nil {
		return form.Parameters, ""
	}
	resource, ok := servicePlanDeploymentResourceByName(form, resourceName)
	if !ok {
		return form.Parameters, ""
	}
	parameters, ok := form.ParametersByResource[resource.ID]
	if !ok {
		return form.Parameters, resource.ID
	}
	return parameters, resource.ID
}

func servicePlanContainsFold(values []string, value string) bool {
	for _, existing := range values {
		if strings.EqualFold(strings.TrimSpace(existing), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmptyStrings(values []string) []string {
	nonEmpty := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			nonEmpty = append(nonEmpty, value)
		}
	}
	return nonEmpty
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
	customerCloudAccounts := []servicePlanCustomerCloudAccountRow{}
	if servicePlanEnvironmentRequiresCustomerAccount(env) {
		customerCloudAccounts, err = listServicePlanCustomerCloudAccounts(ctx, token, env, activeSubscriptions)
		if err != nil {
			return servicePlanEnvironmentDetails{}, err
		}
	}

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
		DeploymentModel:            servicePlanDeploymentModel(env, productTier),
		EnabledFeatures:            productTierEnabledFeatures(productTier),
		Clouds:                     clouds,
		Regions:                    regions,
		Deployments:                servicePlanDeploymentRows(deployments),
		Subscriptions:              servicePlanSubscriptionRows(activeSubscriptions),
		Users:                      servicePlanUserRows(uniqueUsers),
		CustomerCloudAccounts:      customerCloudAccounts,
		DeploymentsCount:           len(deployments),
		ActiveSubscriptionsCount:   len(activeSubscriptions),
		UniqueUsersCount:           len(uniqueUsers),
		CustomerCloudAccountsCount: len(customerCloudAccounts),
	}
	return detail, nil
}

func (productionServicePlanBrowserLoader) LoadDeploymentForm(ctx context.Context, token string, env servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error) {
	version, err := dataaccess.FindPreferredVersion(ctx, token, env.ServiceID, env.PlanID)
	if err != nil {
		version, err = dataaccess.FindLatestVersion(ctx, token, env.ServiceID, env.PlanID)
		if err != nil {
			return servicePlanDeploymentForm{}, err
		}
	}

	offeringResult, err := dataaccess.DescribeServiceOffering(ctx, token, env.ServiceID, env.PlanID, version)
	if err != nil {
		return servicePlanDeploymentForm{}, err
	}
	offering, err := selectServicePlanOffering(offeringResult, env)
	if err != nil {
		return servicePlanDeploymentForm{}, err
	}

	resources := servicePlanDeploymentResources(offering.ResourceParameters)
	if len(resources) == 0 {
		return servicePlanDeploymentForm{}, fmt.Errorf("no deployable resources found for %s / %s", env.ServiceName, env.PlanName)
	}

	parametersByResource := map[string][]servicePlanDeploymentParameter{}
	for _, resource := range resources {
		parametersResult, err := dataaccess.ListInputParameters(ctx, token, env.ServiceID, resource.ID, env.PlanID, version)
		if err != nil {
			return servicePlanDeploymentForm{}, err
		}
		parametersByResource[resource.ID] = servicePlanDeploymentParameters(parametersResult.GetInputParameters())
	}

	pageSize := int64(100)
	exclude := true
	includeInactive := false
	subscriptions, err := dataaccess.ListAllSubscriptions(ctx, token, env.ServiceID, env.ID, &dataaccess.ListSubscriptionsOptions{
		ProductTierId:   &env.PlanID,
		IncludeInactive: &includeInactive,
		ExcludePricing:  &exclude,
		PageSize:        &pageSize,
	})
	if err != nil {
		return servicePlanDeploymentForm{}, err
	}
	activeSubscriptions := activeServicePlanSubscriptions(subscriptions)

	customers := []servicePlanDeploymentCustomer{}
	if servicePlanEnvironmentIsProduction(env) {
		customers, err = listServicePlanDeploymentCustomers(ctx, token)
		if err != nil {
			return servicePlanDeploymentForm{}, err
		}
	}

	customerCloudAccounts := []servicePlanCustomerCloudAccountRow{}
	requiresCustomerAccount := servicePlanEnvironmentRequiresCustomerAccount(env) || servicePlanOfferingRequiresCustomerAccount(offering)
	if requiresCustomerAccount {
		customerCloudAccounts, err = listServicePlanCustomerCloudAccounts(ctx, token, env, activeSubscriptions)
		if err != nil {
			return servicePlanDeploymentForm{}, err
		}
	}

	cloudProviders, regionsByCloud := servicePlanOfferingCloudsAndRegions(offering)
	form := servicePlanDeploymentForm{
		Environment:              env,
		Version:                  version,
		ServiceProviderID:        offeringResult.ConsumptionDescribeServiceOfferingResult.ServiceProviderId,
		ServiceURLKey:            offeringResult.ConsumptionDescribeServiceOfferingResult.ServiceURLKey,
		ServiceAPIVersion:        offering.ServiceAPIVersion,
		ServiceEnvironmentURLKey: offering.ServiceEnvironmentURLKey,
		ServiceModelURLKey:       offering.ServiceModelURLKey,
		ProductTierURLKey:        offering.ProductTierURLKey,
		Resources:                resources,
		CloudProviders:           cloudProviders,
		RegionsByCloud:           regionsByCloud,
		Parameters:               parametersByResource[resources[0].ID],
		ParametersByResource:     parametersByResource,
		CustomerCloudAccounts:    customerCloudAccounts,
		Customers:                customers,
		Subscriptions:            servicePlanSubscriptionRows(activeSubscriptions),
		RequiresCustomerAccount:  requiresCustomerAccount,
	}
	return form, nil
}

func (productionServicePlanBrowserLoader) LaunchDeployment(ctx context.Context, token string, request servicePlanDeploymentLaunchRequest) (string, error) {
	resource, ok := servicePlanDeploymentResourceByName(request.Form, request.ResourceName)
	if !ok {
		return "", fmt.Errorf("resource %q is not available for this plan", request.ResourceName)
	}

	params := request.Params
	if params == nil {
		params = map[string]any{}
	}
	if request.Form.RequiresCustomerAccount {
		customerAccountID := strings.TrimSpace(request.CustomerAccountID)
		if customerAccountID == "" {
			return "", fmt.Errorf("customer cloud account is required for BYOC plans")
		}
		params[servicePlanCustomerAccountConfigIDParamKey] = customerAccountID
	}

	subscriptionID := strings.TrimSpace(request.SubscriptionID)
	if !request.Customer.Self {
		if subscriptionID == "" {
			createResp, err := dataaccess.CreateSubscriptionOnBehalf(ctx, token, request.Form.Environment.ServiceID, request.Form.Environment.ID, &dataaccess.CreateSubscriptionOnBehalfOptions{
				ProductTierID:            request.Form.Environment.PlanID,
				OnBehalfOfCustomerUserID: request.Customer.UserID,
				OnBehalfOfCustomerEmail:  request.Customer.Email,
			})
			if err != nil {
				return "", fmt.Errorf("failed to create subscription for %s: %w", servicePlanDeploymentCustomerLabel(request.Customer), err)
			}
			subscriptionID = strings.TrimSpace(createResp.GetId())
			if subscriptionID == "" {
				return "", fmt.Errorf("subscription creation for %s returned an empty subscription ID", servicePlanDeploymentCustomerLabel(request.Customer))
			}
		}
	}

	createRequest := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		ProductTierVersion: &request.Form.Version,
		CloudProvider:      &request.CloudProvider,
		Region:             &request.Region,
		RequestParams:      params,
	}
	if subscriptionID != "" {
		createRequest.SubscriptionId = &subscriptionID
	}
	instance, err := dataaccess.CreateResourceInstance(
		ctx,
		token,
		request.Form.ServiceProviderID,
		request.Form.ServiceURLKey,
		request.Form.ServiceAPIVersion,
		request.Form.ServiceEnvironmentURLKey,
		request.Form.ServiceModelURLKey,
		request.Form.ProductTierURLKey,
		resource.URLKey,
		createRequest,
	)
	if err != nil {
		return "", err
	}
	if instance == nil || strings.TrimSpace(instance.GetId()) == "" {
		return "", fmt.Errorf("deployment create response did not include an instance ID")
	}
	return instance.GetId(), nil
}

func selectServicePlanOffering(result *openapiclientfleet.InventoryDescribeServiceOfferingResult, env servicePlanBrowserEnvironment) (*openapiclientfleet.ServiceOffering, error) {
	if result == nil || result.ConsumptionDescribeServiceOfferingResult == nil {
		return nil, fmt.Errorf("service offering response is empty for %s / %s", env.ServiceName, env.PlanName)
	}

	var planMatch *openapiclientfleet.ServiceOffering
	for i := range result.ConsumptionDescribeServiceOfferingResult.Offerings {
		offering := &result.ConsumptionDescribeServiceOfferingResult.Offerings[i]
		if !strings.EqualFold(offering.ProductTierID, env.PlanID) {
			continue
		}
		if planMatch == nil {
			planMatch = offering
		}
		if strings.EqualFold(offering.ServiceEnvironmentID, env.ID) {
			return offering, nil
		}
	}
	if planMatch != nil {
		return planMatch, nil
	}
	return nil, fmt.Errorf("service offering not found for %s / %s", env.ServiceName, env.PlanName)
}

func servicePlanDeploymentResources(resources []openapiclientfleet.ResourceEntity) []servicePlanDeploymentResource {
	result := make([]servicePlanDeploymentResource, 0, len(resources))
	for _, resource := range resources {
		if resource.IsDeprecated {
			continue
		}
		result = append(result, servicePlanDeploymentResource{
			ID:     strings.TrimSpace(resource.ResourceId),
			Name:   strings.TrimSpace(resource.Name),
			URLKey: strings.TrimSpace(resource.UrlKey),
		})
	}
	return result
}

func servicePlanDeploymentParameters(parameters []openapiclient.DescribeInputParameterResult) []servicePlanDeploymentParameter {
	result := make([]servicePlanDeploymentParameter, 0, len(parameters))
	for _, parameter := range parameters {
		defaultValue := ""
		if value, ok := parameter.GetDefaultValueOk(); ok && value != nil {
			defaultValue = *value
		}
		result = append(result, servicePlanDeploymentParameter{
			Key:          strings.TrimSpace(parameter.GetKey()),
			DisplayName:  strings.TrimSpace(parameter.GetName()),
			Description:  strings.TrimSpace(parameter.GetDescription()),
			Type:         strings.TrimSpace(parameter.GetType()),
			Required:     parameter.GetRequired(),
			IsList:       parameter.GetIsList(),
			Custom:       servicePlanInputParameterIsCustom(parameter),
			DefaultValue: defaultValue,
			Options:      parameter.GetOptions(),
		})
	}
	return result
}

func servicePlanInputParameterIsCustom(parameter openapiclient.DescribeInputParameterResult) bool {
	value, ok := parameter.AdditionalProperties["custom"]
	if !ok {
		return false
	}
	custom, ok := value.(bool)
	return ok && custom
}

func servicePlanOfferingCloudsAndRegions(offering *openapiclientfleet.ServiceOffering) ([]string, map[string][]string) {
	regionsByCloud := map[string][]string{}
	if offering == nil {
		return nil, regionsByCloud
	}

	add := func(cloud string, regions []string) {
		cloud = strings.ToLower(strings.TrimSpace(cloud))
		if cloud == "" {
			return
		}
		normalizedRegions := make([]string, 0, len(regions))
		for _, region := range regions {
			region = strings.TrimSpace(region)
			if region != "" {
				normalizedRegions = append(normalizedRegions, region)
			}
		}
		if len(normalizedRegions) == 0 {
			return
		}
		sort.Strings(normalizedRegions)
		regionsByCloud[cloud] = normalizedRegions
	}

	add("aws", offering.AwsRegions)
	add("azure", offering.AzureRegions)
	add("gcp", offering.GcpRegions)
	add("oci", offering.OciRegions)
	add("nebius", offering.NebiusRegions)
	add("private", offering.PrivateRegions)
	add("on-prem", offering.OnPremPlatforms)

	cloudSet := map[string]bool{}
	for _, cloud := range offering.CloudProviders {
		cloud = strings.ToLower(strings.TrimSpace(cloud))
		if cloud != "" {
			cloudSet[cloud] = true
		}
	}
	for cloud := range regionsByCloud {
		cloudSet[cloud] = true
	}
	return sortedKeys(cloudSet), regionsByCloud
}

func listServicePlanCustomerCloudAccounts(
	ctx context.Context,
	token string,
	env servicePlanBrowserEnvironment,
	subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult,
) ([]servicePlanCustomerCloudAccountRow, error) {
	searchResult, err := dataaccess.SearchInventory(ctx, token, "resourceinstance:i")
	if err != nil {
		return nil, err
	}

	emailsBySubscription := servicePlanEmailsBySubscription(subscriptions)
	rows := make([]servicePlanCustomerCloudAccountRow, 0)
	for _, record := range searchResult.ResourceInstanceResults {
		resourceID := strings.TrimSpace(utils.FromPtr(record.ResourceId))
		if !strings.HasPrefix(resourceID, servicePlanCustomerAccountResourcePrefix) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.ServiceId), strings.TrimSpace(env.ServiceID)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.ServiceEnvironmentId), strings.TrimSpace(env.ID)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.ProductTierId), strings.TrimSpace(env.PlanID)) {
			continue
		}

		subscriptionID := strings.TrimSpace(utils.FromPtr(record.SubscriptionId))
		rows = append(rows, servicePlanCustomerCloudAccountRow{
			InstanceID:     strings.TrimSpace(record.Id),
			CloudProvider:  strings.TrimSpace(record.CloudProvider),
			Status:         strings.TrimSpace(record.Status),
			SubscriptionID: subscriptionID,
			CustomerEmail:  emailsBySubscription[subscriptionID],
			Resource:       strings.TrimSpace(record.ResourceName),
			Region:         strings.TrimSpace(record.RegionCode),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].InstanceID < rows[j].InstanceID
	})
	return rows, nil
}

func listServicePlanDeploymentCustomers(ctx context.Context, token string) ([]servicePlanDeploymentCustomer, error) {
	customers := make([]servicePlanDeploymentCustomer, 0)
	nextPageToken := ""
	for {
		result, err := dataaccess.ListCustomerUsers(ctx, token, dataaccess.CustomerUserListOptions{
			NextPageToken: nextPageToken,
			PageSize:      100,
			ExcludeStats:  true,
		})
		if err != nil {
			return nil, err
		}
		for _, user := range result.GetUsers() {
			email := strings.TrimSpace(utils.FromPtr(user.Email))
			userID := strings.TrimSpace(utils.FromPtr(user.UserId))
			if email == "" && userID == "" {
				continue
			}
			customers = append(customers, servicePlanDeploymentCustomer{
				UserID:  userID,
				Email:   email,
				Name:    strings.TrimSpace(utils.FromPtr(user.UserName)),
				OrgName: strings.TrimSpace(utils.FromPtr(user.OrgName)),
			})
		}
		nextPageToken = result.GetNextPageToken()
		if nextPageToken == "" {
			break
		}
	}
	sort.Slice(customers, func(i, j int) bool {
		return strings.ToLower(customers[i].Email) < strings.ToLower(customers[j].Email)
	})
	return customers, nil
}

func servicePlanEmailsBySubscription(subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult) map[string]string {
	emails := map[string]string{}
	for _, subscription := range subscriptions {
		id := strings.TrimSpace(subscription.Id)
		if id != "" && strings.TrimSpace(subscription.RootUserEmail) != "" {
			emails[id] = strings.TrimSpace(subscription.RootUserEmail)
		}
	}
	return emails
}

func servicePlanEnvironmentRequiresCustomerAccount(env servicePlanBrowserEnvironment) bool {
	return servicePlanHostingBadgeForValues(env.TenancyType, env.DeploymentType).Label == "BYOC"
}

func servicePlanEnvironmentIsProduction(env servicePlanBrowserEnvironment) bool {
	normalized := strings.ToLower(strings.TrimSpace(env.Name))
	return normalized == "prod" || normalized == "production"
}

func servicePlanOfferingRequiresCustomerAccount(offering *openapiclientfleet.ServiceOffering) bool {
	if offering == nil {
		return false
	}
	return servicePlanHostingBadgeForValues(offering.ServiceModelType, offering.ProductTierType).Label == "BYOC"
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
			RootUserID:    subscription.RootUserId,
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
