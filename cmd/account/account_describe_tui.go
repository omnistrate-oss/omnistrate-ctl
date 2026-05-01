package account

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/browser"
	"golang.org/x/term"
)

const (
	defaultAccountDescribeWidth      = 120
	defaultAccountDescribeHeight     = 32
	accountDescribeListMinWidth      = 34
	accountDescribeListPreferredWide = 38
	accountDescribeHelpHeight        = 2
	accountDescribeHeaderHeight      = 5
)

type accountDescribeFocus int

const (
	accountDescribeFocusSections accountDescribeFocus = iota
	accountDescribeFocusDetails
)

type accountProvider string

const (
	accountProviderAWS     accountProvider = "aws"
	accountProviderGCP     accountProvider = "gcp"
	accountProviderAzure   accountProvider = "azure"
	accountProviderNebius  accountProvider = "nebius"
	accountProviderOCI     accountProvider = "oci"
	accountProviderUnknown accountProvider = "unknown"
)

type accountDescribeNode struct {
	key         string
	title       string
	description string
	status      string
	content     string
	copyText    string
	openURL     string
	expandable  bool
	expanded    bool
	children    []*accountDescribeNode
}

type accountDescribeItem struct {
	key         string
	parentKey   string
	title       string
	description string
	status      string
	content     string
	copyText    string
	openURL     string
	level       int
	expandable  bool
	expanded    bool
}

func (i accountDescribeItem) Title() string       { return i.title }
func (i accountDescribeItem) Description() string { return i.description }
func (i accountDescribeItem) FilterValue() string {
	return i.title + " " + i.description + " " + i.status
}

type accountDescribeDelegate struct {
	list.DefaultDelegate
}

func newAccountDescribeDelegate() accountDescribeDelegate {
	delegate := accountDescribeDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	delegate.SetHeight(2)
	delegate.SetSpacing(0)
	return delegate
}

func (d accountDescribeDelegate) Render(writer io.Writer, model list.Model, index int, item list.Item) {
	describeItem, ok := item.(accountDescribeItem)
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

	indent := strings.Repeat("  ", describeItem.level)
	titlePrefix := ""
	switch {
	case describeItem.expandable && describeItem.expanded:
		titlePrefix = "▾ "
	case describeItem.expandable:
		titlePrefix = "▸ "
	case describeItem.level > 0:
		titlePrefix = "• "
	}

	title := indent + titlePrefix + describeItem.title
	if describeItem.status != "" {
		title = fmt.Sprintf("%s  %s", title, renderStatusBadge(describeItem.status))
	}

	description := describeItem.description
	if description != "" {
		description = indent + "  " + description
	}

	fmt.Fprintf(writer, "%s\n%s", titleStyle.Render(title), descriptionStyle.Render(description))
}

type accountDescribeModel struct {
	account          *openapiclient.DescribeAccountConfigResult
	provider         accountProvider
	rootNodes        []*accountDescribeNode
	items            []accountDescribeItem
	list             list.Model
	viewport         viewport.Model
	focus            accountDescribeFocus
	width            int
	height           int
	listPanelWidth   int
	detailPanelWidth int
	statusMessage    string
	linkOptions      []accountDescribeLinkOption
	linkIndex        int
}

type accountDescribeActionResultMsg struct {
	message string
	err     error
}

type accountDescribeLinkOption struct {
	label string
	url   string
}

var accountDescribeURLPattern = regexp.MustCompile(`https?://[^\s]+`)

func newAccountDescribeModel(account *openapiclient.DescribeAccountConfigResult) accountDescribeModel {
	provider := detectAccountProvider(account)
	rootNodes := buildAccountDescribeNodes(account)

	delegate := newAccountDescribeDelegate()
	sectionList := list.New(nil, delegate, 0, 0)
	sectionList.Title = "Sections"
	sectionList.SetShowHelp(false)
	sectionList.SetShowFilter(false)
	sectionList.SetShowStatusBar(false)
	sectionList.SetFilteringEnabled(false)
	sectionList.DisableQuitKeybindings()
	sectionList.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")).Padding(0, 1)

	detailViewport := viewport.New(0, 0)

	model := accountDescribeModel{
		account:   account,
		provider:  provider,
		rootNodes: rootNodes,
		list:      sectionList,
		viewport:  detailViewport,
		focus:     accountDescribeFocusSections,
	}
	model.setSize(defaultAccountDescribeWidth, defaultAccountDescribeHeight)
	model.rebuildVisibleItems("")
	return model
}

func (m accountDescribeModel) Init() tea.Cmd {
	return nil
}

func (m accountDescribeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case accountDescribeActionResultMsg:
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
			if m.focus == accountDescribeFocusSections {
				m.focus = accountDescribeFocusDetails
			} else {
				m.focus = accountDescribeFocusSections
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

		if m.focus == accountDescribeFocusDetails {
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

func (m accountDescribeModel) handleViewportKey(msg tea.KeyMsg) accountDescribeModel {
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

func (m accountDescribeModel) handleLinkPickerKey(msg tea.KeyMsg) accountDescribeModel {
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

func (m *accountDescribeModel) setSize(width, height int) {
	if width <= 0 {
		width = defaultAccountDescribeWidth
	}
	if height <= 0 {
		height = defaultAccountDescribeHeight
	}

	m.width = width
	m.height = height

	listPanelWidth := min(max(width/3, accountDescribeListMinWidth), accountDescribeListPreferredWide)
	bodyHeight := max(height-accountDescribeHeaderHeight-accountDescribeHelpHeight, 12)
	detailPanelWidth := max(width-listPanelWidth-1, 40)
	panelFrameWidth := accountDescribePanelStyle(lipgloss.Color("240")).GetHorizontalFrameSize()

	m.listPanelWidth = listPanelWidth
	m.detailPanelWidth = detailPanelWidth
	m.list.SetWidth(max(listPanelWidth-panelFrameWidth, 10))
	m.list.SetHeight(bodyHeight)
	m.viewport.Width = max(detailPanelWidth-panelFrameWidth, 10)
	m.viewport.Height = bodyHeight
	m.syncViewportContent()
}

func (m *accountDescribeModel) selectedItem() *accountDescribeItem {
	index := m.list.Index()
	if index < 0 || index >= len(m.items) {
		return nil
	}
	return &m.items[index]
}

func (m *accountDescribeModel) toggleSelectedExpandable() bool {
	selected := m.selectedItem()
	if selected == nil || !selected.expandable {
		return false
	}
	if toggleNodeExpanded(m.rootNodes, selected.key) {
		m.rebuildVisibleItems(selected.key)
		return true
	}
	return false
}

func (m *accountDescribeModel) collapseSelectedItem() bool {
	selected := m.selectedItem()
	if selected == nil {
		return false
	}

	if selected.expandable && selected.expanded {
		if setNodeExpanded(m.rootNodes, selected.key, false) {
			m.rebuildVisibleItems(selected.key)
			return true
		}
		return false
	}

	if selected.parentKey == "" {
		return false
	}

	if setNodeExpanded(m.rootNodes, selected.parentKey, false) {
		m.rebuildVisibleItems(selected.parentKey)
		return true
	}
	return false
}

func (m *accountDescribeModel) rebuildVisibleItems(selectedKey string) {
	if selectedKey == "" {
		if current := m.selectedItem(); current != nil {
			selectedKey = current.key
		}
	}

	m.items = flattenAccountDescribeNodes(m.rootNodes, 0, "")
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

func (m *accountDescribeModel) syncViewportContent() {
	selected := m.selectedItem()
	if selected == nil {
		m.viewport.SetContent("No account details are available.")
		return
	}
	m.viewport.SetContent(wrapAccountDescribeContent(selected.content, m.viewport.Width))
	m.viewport.GotoTop()
}

func (m accountDescribeModel) copySelectedValue() (tea.Model, tea.Cmd) {
	text := m.selectedCopyText()
	if text == "" {
		m.statusMessage = "Nothing actionable to copy from this section"
		return m, nil
	}

	return m, copyToClipboardCmd(text, "Copied selected value to clipboard")
}

func (m accountDescribeModel) openSelectedURL() (tea.Model, tea.Cmd) {
	links := m.selectedLinkOptions()
	switch len(links) {
	case 0:
		m.statusMessage = "No URL available for the selected section"
		return m, nil
	case 1:
		m.clearLinkPicker()
		return m, openURLCmd(links[0].url)
	default:
		m.linkOptions = links
		m.linkIndex = 0
		m.statusMessage = "Multiple URLs found. Choose one with ↑/↓ and press enter or o"
		return m, nil
	}
}

func (m accountDescribeModel) selectedCopyText() string {
	selected := m.selectedItem()
	if selected == nil {
		return ""
	}

	return selectedCopyText(selected)
}

func (m accountDescribeModel) selectedLinkOptions() []accountDescribeLinkOption {
	selected := m.selectedItem()
	if selected == nil {
		return nil
	}

	return selectedLinkOptions(selected)
}

func (m accountDescribeModel) linkPickerActive() bool {
	return len(m.linkOptions) > 0
}

func (m *accountDescribeModel) clearLinkPicker() {
	m.linkOptions = nil
	m.linkIndex = 0
}

func (m accountDescribeModel) currentLinkOption() *accountDescribeLinkOption {
	if len(m.linkOptions) == 0 || m.linkIndex < 0 || m.linkIndex >= len(m.linkOptions) {
		return nil
	}

	return &m.linkOptions[m.linkIndex]
}

func (m accountDescribeModel) copySelectedLink() (tea.Model, tea.Cmd) {
	link := m.currentLinkOption()
	if link == nil {
		m.statusMessage = "No URL selected"
		return m, nil
	}

	m.clearLinkPicker()
	return m, copyToClipboardCmd(link.url, "Copied selected URL to clipboard")
}

func (m accountDescribeModel) openSelectedLink() (tea.Model, tea.Cmd) {
	link := m.currentLinkOption()
	if link == nil {
		m.statusMessage = "No URL selected"
		return m, nil
	}

	m.clearLinkPicker()
	return m, openURLCmd(link.url)
}

func (m accountDescribeModel) View() string {
	header := renderAccountDescribeHeader(m.account, m.provider)

	listPanelStyle := accountDescribePanelStyle(m.focusBorderColor(accountDescribeFocusSections))
	detailPanelStyle := accountDescribePanelStyle(m.focusBorderColor(accountDescribeFocusDetails))

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

func (m accountDescribeModel) focusBorderColor(focus accountDescribeFocus) lipgloss.Color {
	if m.focus == focus {
		return lipgloss.Color("117")
	}
	return lipgloss.Color("240")
}

func accountDescribePanelStyle(borderColor lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}

func printAccountDescribeTUI(account *openapiclient.DescribeAccountConfigResult) error {
	if account == nil {
		return fmt.Errorf("account details are required")
	}

	snapshot := renderAccountDescribeSnapshot(account)
	utils.LastPrintedString = snapshot

	if !isAccountDescribeInteractive() {
		fmt.Println(snapshot)
		return nil
	}

	program := tea.NewProgram(newAccountDescribeModel(account), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to launch account TUI: %w", err)
	}

	return nil
}

func renderAccountDescribeSnapshot(account *openapiclient.DescribeAccountConfigResult) string {
	model := newAccountDescribeModel(account)
	return strings.TrimRight(model.View(), "\n")
}

func isAccountDescribeInteractive() bool {
	return fileIsTerminal(os.Stdout) && fileIsTerminal(os.Stdin)
}

func fileIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}

	fd := file.Fd()
	if fd > uintptr(^uint(0)>>1) {
		return false
	}

	return term.IsTerminal(int(fd))
}

func buildAccountDescribeNodes(account *openapiclient.DescribeAccountConfigResult) []*accountDescribeNode {
	provider := detectAccountProvider(account)
	nodes := []*accountDescribeNode{
		buildOverviewNode(account, provider),
	}

	if len(account.ByoaInstanceIDs) > 0 {
		nodes = append(nodes, buildBYOANode(account))
	}

	switch provider {
	case accountProviderAWS:
		nodes = append(nodes, buildAWSIdentityNode(account))
		nodes = append(nodes, buildActionGroupNode(
			account.Status,
			commandNode("aws-bootstrap", "Bootstrap", "CloudFormation template", utils.FromPtr(account.AwsCloudFormationTemplateURL)),
			commandNode("aws-bootstrap-no-lb", "Bootstrap (No LB)", "CloudFormation template without LoadBalancer policy", utils.FromPtr(account.AwsCloudFormationNoLBTemplateURL)),
		)...)
	case accountProviderGCP:
		nodes = append(nodes, buildGCPIdentityNode(account))
		nodes = append(nodes, buildActionGroupNode(
			account.Status,
			commandNode("gcp-bootstrap", "Bootstrap", "Bootstrap shell command", utils.FromPtr(account.GcpBootstrapShellCommand)),
			commandNode("gcp-disconnect", "Disconnect", "Disconnect shell command", utils.FromPtr(account.GcpDisconnectShellCommand)),
			commandNode("gcp-offboard", "Offboard", "Offboard shell command", utils.FromPtr(account.GcpOffboardShellCommand)),
		)...)
	case accountProviderAzure:
		nodes = append(nodes, buildAzureIdentityNode(account))
		nodes = append(nodes, buildActionGroupNode(
			account.Status,
			commandNode("azure-bootstrap", "Bootstrap", "Bootstrap shell command", utils.FromPtr(account.AzureBootstrapShellCommand)),
			commandNode("azure-disconnect", "Disconnect", "Disconnect shell command", utils.FromPtr(account.AzureDisconnectShellCommand)),
			commandNode("azure-offboard", "Offboard", "Offboard shell command", utils.FromPtr(account.AzureOffboardShellCommand)),
		)...)
	case accountProviderOCI:
		nodes = append(nodes, buildOCIIdentityNode(account))
		nodes = append(nodes, buildActionGroupNode(
			account.Status,
			commandNode("oci-bootstrap", "Bootstrap", "Bootstrap shell command", utils.FromPtr(account.OciBootstrapShellCommand)),
			commandNode("oci-disconnect", "Disconnect", "Disconnect shell command", utils.FromPtr(account.OciDisconnectShellCommand)),
			commandNode("oci-offboard", "Offboard", "Offboard shell command", utils.FromPtr(account.OciOffboardShellCommand)),
		)...)
	case accountProviderNebius:
		nodes = append(nodes, buildNebiusBindingsNode(account))
	}

	return nodes
}

func buildOverviewNode(account *openapiclient.DescribeAccountConfigResult, provider accountProvider) *accountDescribeNode {
	identityLabel, identityValue := primaryIdentifier(account, provider)
	providerConfiguredLabel, providerConfigured := providerConfiguredCheck(account, provider)
	checks := []string{
		renderCheck("Account status is READY", account.Status == "READY"),
	}
	if providerConfiguredLabel != "" {
		checks = append(checks, renderCheck(providerConfiguredLabel, providerConfigured))
	}

	lines := []string{
		"Overview",
		"",
	}
	lines = append(lines, checks...)
	lines = append(lines, "",
		fmt.Sprintf("Provider: %s", providerDisplayName(provider)),
		fmt.Sprintf("Name: %s", displayValue(account.Name)),
		fmt.Sprintf("ID: %s", displayValue(account.Id)),
		fmt.Sprintf("Cloud Provider ID: %s", displayValue(account.CloudProviderId)),
		fmt.Sprintf("Description: %s", displayValue(account.Description)),
		fmt.Sprintf("%s: %s", identityLabel, displayValue(identityValue)),
		fmt.Sprintf("Status: %s", displayValue(account.Status)),
		fmt.Sprintf("Status Message: %s", displayValue(account.StatusMessage)),
	)
	if len(account.ByoaInstanceIDs) > 0 {
		lines = append(lines, fmt.Sprintf("Linked BYOA Instances: %d", len(account.ByoaInstanceIDs)))
	}
	if provider == accountProviderNebius {
		lines = append(lines, fmt.Sprintf("Bindings: %d", len(account.NebiusBindings)))
	}

	return &accountDescribeNode{
		key:         "overview",
		title:       "Overview",
		description: displayValue(identityValue),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildBYOANode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	lines := []string{"Linked BYOA Instances", ""}
	for _, instanceID := range account.ByoaInstanceIDs {
		lines = append(lines, fmt.Sprintf("- %s", instanceID))
	}
	return &accountDescribeNode{
		key:         "byoa-instances",
		title:       "BYOA Instances",
		description: fmt.Sprintf("%d linked instance(s)", len(account.ByoaInstanceIDs)),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildAWSIdentityNode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	lines := []string{
		"AWS Identity",
		"",
		fmt.Sprintf("AWS Account ID: %s", displayValue(utils.FromPtr(account.AwsAccountID))),
		fmt.Sprintf("Bootstrap Role ARN: %s", displayValue(utils.FromPtr(account.AwsBootstrapRoleARN))),
		fmt.Sprintf("CloudFormation Template URL: %s", displayValue(utils.FromPtr(account.AwsCloudFormationTemplateURL))),
		fmt.Sprintf("CloudFormation No-LB Template URL: %s", displayValue(utils.FromPtr(account.AwsCloudFormationNoLBTemplateURL))),
	}
	return &accountDescribeNode{
		key:         "identity",
		title:       "Identity",
		description: displayValue(utils.FromPtr(account.AwsAccountID)),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildGCPIdentityNode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	lines := []string{
		"GCP Identity",
		"",
		fmt.Sprintf("Project ID: %s", displayValue(utils.FromPtr(account.GcpProjectID))),
		fmt.Sprintf("Project Number: %s", displayValue(utils.FromPtr(account.GcpProjectNumber))),
		fmt.Sprintf("Service Account Email: %s", displayValue(utils.FromPtr(account.GcpServiceAccountEmail))),
	}
	return &accountDescribeNode{
		key:         "identity",
		title:       "Identity",
		description: displayValue(utils.FromPtr(account.GcpProjectID)),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildAzureIdentityNode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	lines := []string{
		"Azure Identity",
		"",
		fmt.Sprintf("Subscription ID: %s", displayValue(utils.FromPtr(account.AzureSubscriptionID))),
		fmt.Sprintf("Tenant ID: %s", displayValue(utils.FromPtr(account.AzureTenantID))),
	}
	return &accountDescribeNode{
		key:         "identity",
		title:       "Identity",
		description: displayValue(utils.FromPtr(account.AzureSubscriptionID)),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildOCIIdentityNode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	lines := []string{
		"OCI Identity",
		"",
		fmt.Sprintf("Tenancy OCID: %s", displayValue(utils.FromPtr(account.OciTenancyID))),
		fmt.Sprintf("Domain OCID: %s", displayValue(utils.FromPtr(account.OciDomainID))),
	}
	return &accountDescribeNode{
		key:         "identity",
		title:       "Identity",
		description: displayValue(utils.FromPtr(account.OciTenancyID)),
		status:      account.Status,
		content:     strings.Join(lines, "\n"),
	}
}

func buildNebiusBindingsNode(account *openapiclient.DescribeAccountConfigResult) *accountDescribeNode {
	if len(account.NebiusBindings) == 0 {
		return &accountDescribeNode{
			key:         "nebius-bindings",
			title:       "Bindings",
			description: "no bindings configured",
			status:      account.Status,
			expandable:  true,
			expanded:    true,
			content: strings.Join([]string{
				"Nebius Bindings",
				"",
				renderCheck("At least one Nebius binding is configured", false),
				"",
				"No Nebius bindings are configured.",
			}, "\n"),
		}
	}

	children := make([]*accountDescribeNode, 0, len(account.NebiusBindings))
	summaryLines := []string{
		"Nebius Bindings",
		"",
		renderCheck("At least one Nebius binding is configured", true),
		"",
	}

	for index, binding := range account.NebiusBindings {
		status := nebiusBindingStatus(account, binding)
		statusMessage := nebiusBindingStatusMessage(account, binding)
		keyState := utils.FromPtr(binding.KeyState)
		title := nebiusBindingAccordionTitle(binding, index)
		description := nebiusBindingAccordionDescription(binding)

		children = append(children, &accountDescribeNode{
			key:         fmt.Sprintf("nebius-binding-%d", index),
			title:       title,
			description: description,
			status:      status,
			content: strings.Join([]string{
				fmt.Sprintf("Binding %d", index+1),
				"",
				renderCheck("Binding status is READY", status == "READY"),
				renderCheck("Public key ID matches the configured private key", utils.FromPtr(binding.PublicKeyIDMatches)),
				renderCheck("Service account ownership validated", utils.FromPtr(binding.ServiceAccountKeyValidated)),
				renderCheck("Nebius key state is ACTIVE", keyState == "ACTIVE"),
				"",
				fmt.Sprintf("Title: %s", displayValue(title)),
				fmt.Sprintf("Project ID: %s", displayValue(binding.ProjectID)),
				fmt.Sprintf("Region: %s", displayValue(binding.Region)),
				fmt.Sprintf("Service Account ID: %s", displayValue(binding.ServiceAccountID)),
				fmt.Sprintf("Public Key ID: %s", displayValue(binding.PublicKeyID)),
				fmt.Sprintf("Derived Public Key Fingerprint: %s", displayValue(utils.FromPtr(binding.DerivedPublicKeyFingerprint))),
				fmt.Sprintf("Nebius Key Fingerprint: %s", displayValue(utils.FromPtr(binding.KeyFingerprint))),
				fmt.Sprintf("Nebius Key State: %s", displayValue(keyState)),
				fmt.Sprintf("Key Expires At: %s", formatTime(binding.KeyExpiresAt)),
				fmt.Sprintf("Binding Status: %s", displayValue(status)),
				fmt.Sprintf("Status Message: %s", displayValue(statusMessage)),
			}, "\n"),
		})

		summaryLines = append(summaryLines, fmt.Sprintf("- %s  (%s)", title, displayValue(status)))
	}

	return &accountDescribeNode{
		key:         "nebius-bindings",
		title:       "Bindings",
		description: fmt.Sprintf("%d binding(s)", len(account.NebiusBindings)),
		status:      account.Status,
		content:     strings.Join(summaryLines, "\n"),
		expandable:  true,
		expanded:    true,
		children:    children,
	}
}

func buildActionGroupNode(status string, children ...*accountDescribeNode) []*accountDescribeNode {
	filtered := make([]*accountDescribeNode, 0, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}
		filtered = append(filtered, child)
	}
	if len(filtered) == 0 {
		return nil
	}

	return []*accountDescribeNode{{
		key:         "actions",
		title:       "Actions",
		description: fmt.Sprintf("%d action(s)", len(filtered)),
		status:      status,
		content: strings.Join([]string{
			"Actions",
			"",
			"Select one of the actions in the left pane to view the full command or URL.",
		}, "\n"),
		expandable: true,
		expanded:   true,
		children:   filtered,
	}}
}

func commandNode(keySuffix, title, description, value string) *accountDescribeNode {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &accountDescribeNode{
		key:         keySuffix,
		title:       title,
		description: description,
		copyText:    value,
		openURL:     extractSingleURL(value),
		content: strings.Join([]string{
			title,
			"",
			value,
		}, "\n"),
	}
}

func renderAccountDescribeHeader(account *openapiclient.DescribeAccountConfigResult, provider accountProvider) string {
	providerName := providerDisplayName(provider)
	identityLabel, identityValue := primaryIdentifier(account, provider)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(
		fmt.Sprintf("%s Account · %s", providerName, displayValue(account.Name)),
	)

	summary := []string{
		displayValue(account.Id),
		fmt.Sprintf("%s %s", identityLabel, displayValue(identityValue)),
		fmt.Sprintf("status %s", displayValue(account.Status)),
	}
	if provider == accountProviderNebius {
		summary = append(summary, fmt.Sprintf("%d binding(s)", len(account.NebiusBindings)))
	}
	metadata := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(strings.Join(summary, "  |  "))

	providerConfiguredLabel, providerConfigured := providerConfiguredCheck(account, provider)
	checks := []string{renderCheck("Account READY", account.Status == "READY")}
	if providerConfiguredLabel != "" {
		checks = append(checks, renderCheck(providerConfiguredLabel, providerConfigured))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		metadata,
		strings.Join(checks, "  "),
	)
}

func flattenAccountDescribeNodes(nodes []*accountDescribeNode, level int, parentKey string) []accountDescribeItem {
	items := make([]accountDescribeItem, 0)
	for _, node := range nodes {
		if node == nil {
			continue
		}
		items = append(items, accountDescribeItem{
			key:         node.key,
			parentKey:   parentKey,
			title:       node.title,
			description: node.description,
			status:      node.status,
			content:     node.content,
			copyText:    node.copyText,
			openURL:     node.openURL,
			level:       level,
			expandable:  node.expandable,
			expanded:    node.expanded,
		})
		if node.expandable && node.expanded {
			items = append(items, flattenAccountDescribeNodes(node.children, level+1, node.key)...)
		}
	}
	return items
}

func toggleNodeExpanded(nodes []*accountDescribeNode, key string) bool {
	node := findNode(nodes, key)
	if node == nil || !node.expandable {
		return false
	}
	node.expanded = !node.expanded
	return true
}

func setNodeExpanded(nodes []*accountDescribeNode, key string, expanded bool) bool {
	node := findNode(nodes, key)
	if node == nil || !node.expandable {
		return false
	}
	node.expanded = expanded
	return true
}

func findNode(nodes []*accountDescribeNode, key string) *accountDescribeNode {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.key == key {
			return node
		}
		if child := findNode(node.children, key); child != nil {
			return child
		}
	}
	return nil
}

func renderStatusBadge(status string) string {
	status = strings.ToUpper(strings.TrimSpace(status))
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	switch {
	case strings.Contains(status, "READY") || strings.Contains(status, "SUCCESS"):
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	case strings.Contains(status, "ERROR") || strings.Contains(status, "FAIL"):
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	case strings.Contains(status, "PENDING") || strings.Contains(status, "CREATING") || strings.Contains(status, "UPDATING"):
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	}
	if status == "" {
		status = "UNKNOWN"
	}
	return style.Render(status)
}

func renderCheck(label string, ok bool) string {
	icon := "○"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	if ok {
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	}
	return style.Render(fmt.Sprintf("%s %s", icon, label))
}

func detectAccountProvider(account *openapiclient.DescribeAccountConfigResult) accountProvider {
	switch {
	case account == nil:
		return accountProviderUnknown
	case account.NebiusTenantID != nil || len(account.NebiusBindings) > 0:
		return accountProviderNebius
	case account.AwsAccountID != nil || account.AwsBootstrapRoleARN != nil || account.AwsCloudFormationTemplateURL != nil:
		return accountProviderAWS
	case account.GcpProjectID != nil || account.GcpProjectNumber != nil || account.GcpServiceAccountEmail != nil:
		return accountProviderGCP
	case account.AzureSubscriptionID != nil || account.AzureTenantID != nil:
		return accountProviderAzure
	case account.OciTenancyID != nil || account.OciDomainID != nil:
		return accountProviderOCI
	default:
		return accountProviderUnknown
	}
}

func providerDisplayName(provider accountProvider) string {
	switch provider {
	case accountProviderAWS:
		return "AWS"
	case accountProviderGCP:
		return "GCP"
	case accountProviderAzure:
		return "Azure"
	case accountProviderNebius:
		return "Nebius"
	case accountProviderOCI:
		return "OCI"
	default:
		return "Cloud"
	}
}

func primaryIdentifier(account *openapiclient.DescribeAccountConfigResult, provider accountProvider) (label string, value string) {
	switch provider {
	case accountProviderAWS:
		return "account", utils.FromPtr(account.AwsAccountID)
	case accountProviderGCP:
		return "project", utils.FromPtr(account.GcpProjectID)
	case accountProviderAzure:
		return "subscription", utils.FromPtr(account.AzureSubscriptionID)
	case accountProviderNebius:
		return "tenant", utils.FromPtr(account.NebiusTenantID)
	case accountProviderOCI:
		return "tenancy", utils.FromPtr(account.OciTenancyID)
	default:
		return "id", account.Id
	}
}

func providerConfiguredCheck(account *openapiclient.DescribeAccountConfigResult, provider accountProvider) (label string, ok bool) {
	switch provider {
	case accountProviderAWS:
		return "AWS identity configured", utils.FromPtr(account.AwsAccountID) != "" && utils.FromPtr(account.AwsBootstrapRoleARN) != ""
	case accountProviderGCP:
		return "GCP identity configured", utils.FromPtr(account.GcpProjectID) != "" && utils.FromPtr(account.GcpProjectNumber) != ""
	case accountProviderAzure:
		return "Azure identity configured", utils.FromPtr(account.AzureSubscriptionID) != "" && utils.FromPtr(account.AzureTenantID) != ""
	case accountProviderNebius:
		return "Bindings configured", len(account.NebiusBindings) > 0
	case accountProviderOCI:
		return "OCI identity configured", utils.FromPtr(account.OciTenancyID) != "" && utils.FromPtr(account.OciDomainID) != ""
	default:
		return "", false
	}
}

func nebiusBindingAccordionTitle(binding openapiclient.NebiusAccountBindingResult, index int) string {
	if region := strings.TrimSpace(binding.Region); region != "" {
		return region
	}
	if projectID := strings.TrimSpace(binding.ProjectID); projectID != "" {
		return projectID
	}
	if serviceAccountID := strings.TrimSpace(binding.ServiceAccountID); serviceAccountID != "" {
		return serviceAccountID
	}
	return fmt.Sprintf("Binding %d", index+1)
}

func nebiusBindingAccordionDescription(binding openapiclient.NebiusAccountBindingResult) string {
	if strings.TrimSpace(binding.Region) != "" {
		return firstNonEmpty(binding.ProjectID, binding.ServiceAccountID, binding.PublicKeyID)
	}
	return firstNonEmpty(binding.ServiceAccountID, binding.PublicKeyID)
}

func nebiusBindingStatus(account *openapiclient.DescribeAccountConfigResult, binding openapiclient.NebiusAccountBindingResult) string {
	status := utils.FromPtr(binding.Status)
	if status == "" {
		status = account.Status
	}
	return status
}

func nebiusBindingStatusMessage(account *openapiclient.DescribeAccountConfigResult, binding openapiclient.NebiusAccountBindingResult) string {
	statusMessage := utils.FromPtr(binding.StatusMessage)
	if statusMessage == "" {
		statusMessage = account.StatusMessage
	}
	return statusMessage
}

func displayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func formatTime(ts *time.Time) string {
	if ts == nil {
		return "unknown"
	}
	return ts.UTC().Format(time.RFC3339)
}

func wrapAccountDescribeContent(content string, width int) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapAccountDescribeLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapAccountDescribeLine(line string, width int) []string {
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
			if isAccountDescribeSoftBreak(r) {
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

func isAccountDescribeSoftBreak(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("/:?&=_-.,", r)
}

func (m accountDescribeModel) statusLine() string {
	if m.linkPickerActive() {
		return "Select a URL with ↑/↓. Press enter or o to open, c to copy, esc to close."
	}
	if strings.TrimSpace(m.statusMessage) != "" {
		return m.statusMessage
	}
	return "Use c to copy the selected value. Use o to open a selected URL."
}

func selectedCopyText(item *accountDescribeItem) string {
	if item == nil {
		return ""
	}

	if text := strings.TrimSpace(item.copyText); text != "" {
		return text
	}

	return strings.TrimSpace(item.content)
}

func selectedOpenURL(item *accountDescribeItem) string {
	if item == nil {
		return ""
	}

	if url := strings.TrimSpace(item.openURL); url != "" {
		return url
	}

	return extractSingleURL(firstNonEmpty(item.copyText, item.content))
}

func selectedLinkOptions(item *accountDescribeItem) []accountDescribeLinkOption {
	if item == nil {
		return nil
	}

	options := make([]accountDescribeLinkOption, 0)
	seen := make(map[string]struct{})
	addOption := func(label, url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		if _, exists := seen[url]; exists {
			return
		}
		seen[url] = struct{}{}
		label = strings.TrimSpace(label)
		if label == "" {
			label = url
		}
		options = append(options, accountDescribeLinkOption{label: label, url: url})
	}

	if url := strings.TrimSpace(item.openURL); url != "" {
		addOption(item.title, url)
		return options
	}

	if url := extractSingleURL(item.copyText); url != "" {
		addOption(item.title, url)
	}

	for _, line := range strings.Split(item.content, "\n") {
		matches := accountDescribeURLPattern.FindAllString(strings.TrimSpace(line), -1)
		for _, url := range matches {
			label := strings.TrimSpace(strings.Replace(line, url, "", 1))
			label = strings.TrimSpace(strings.TrimSuffix(label, ":"))
			addOption(label, url)
		}
	}

	return options
}

func extractSingleURL(text string) string {
	matches := accountDescribeURLPattern.FindAllString(strings.TrimSpace(text), -1)
	if len(matches) != 1 {
		return ""
	}
	return matches[0]
}

func (m accountDescribeModel) renderLinkPicker() string {
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
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render("  "+wrapAccountDescribeContent(option.url, max(m.width-8, 32))))
	}

	return accountDescribePanelStyle(lipgloss.Color("117")).Render(strings.Join(lines, "\n"))
}

func copyToClipboardCmd(text, successMessage string) tea.Cmd {
	return func() tea.Msg {
		if err := copyToClipboard(text); err != nil {
			return accountDescribeActionResultMsg{err: fmt.Errorf("clipboard copy failed: %w", err)}
		}
		return accountDescribeActionResultMsg{message: successMessage}
	}
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := browser.OpenURL(url); err != nil {
			return accountDescribeActionResultMsg{err: fmt.Errorf("failed to open URL: %w", err)}
		}
		return accountDescribeActionResultMsg{message: "Opened selected URL in browser"}
	}
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip.exe")
	default:
		return fmt.Errorf("clipboard is not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
