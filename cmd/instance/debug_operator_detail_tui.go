package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

const (
	opTabInputVars     = 0
	opTabOutputVars    = 1
	opTabCRDOutputVars = 2
	opTabWfErrors      = 3
	opNumTabs          = 4
)

var opTabNames = []string{"Input Parameters", "Output Parameters", "Operator CRD Outputs", "Workflow Events"}

func init() {
	if len(opTabNames) != opNumTabs {
		panic(fmt.Sprintf("opTabNames length %d does not match opNumTabs %d", len(opTabNames), opNumTabs))
	}
}

// operatorDataMsg is sent when operator debug data has been fetched.
type operatorDataMsg struct {
	operatorData *OperatorData
	err          error
}

type operatorDetailModel struct {
	node      PlanDAGNode
	debugData DebugData
	activeTab int
	width     int
	height    int

	loading bool
	loadErr error
	spinner spinner.Model

	operatorData *OperatorData

	// Input variables tree
	inputTree   []outputNode
	inputCursor int
	inputScroll int

	// Output variables tree
	outputTree   []outputNode
	outputCursor int
	outputScroll int

	// CRD output parameters tree
	crdOutputTree   []outputNode
	crdOutputCursor int
	crdOutputScroll int

	// Workflow Events tab
	wfErrors *workflowErrorsState

	clipboardMsg string
}

func newOperatorDetailModel(node PlanDAGNode, data DebugData) operatorDetailModel {
	return operatorDetailModel{
		node:      node,
		debugData: data,
		activeTab: opTabInputVars,
		loading:   true,
		spinner:   newResourceDetailSpinner(),
		wfErrors:  &workflowErrorsState{},
	}
}

func (m operatorDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchOperatorData())
}

func (m operatorDetailModel) fetchOperatorData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		opData := &OperatorData{}

		// Fetch all input parameters
		inputParams, err := fetchInputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.InputParams,
		)
		if err == nil {
			opData.InputParams = inputParams
		}

		// Fetch exported output parameters
		outputParams, listErr := fetchOutputParams(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			m.debugData.ProductTierID, m.debugData.TierVersion,
			m.debugData.ResultParams,
		)
		if listErr == nil {
			opData.OutputParams = outputParams
		}

		// Fetch CRD output parameters from DescribeResource (operatorCRDConfiguration.outputParameters)
		resourceResult, descErr := dataaccess.DescribeResource(
			ctx, m.debugData.Token,
			m.debugData.ServiceID, m.node.ID,
			&m.debugData.ProductTierID, &m.debugData.TierVersion,
		)
		if descErr == nil && resourceResult != nil {
			crdConfig, ok := resourceResult.GetOperatorCRDConfigurationOk()
			if ok && crdConfig != nil {
				outputParams := crdConfig.GetOutputParameters()
				for k, v := range outputParams {
					crdParam := OperatorCRDOutputParam{
						Key:   k,
						Value: v,
					}
					if m.debugData.ResultParams != nil {
						if rv, ok := m.debugData.ResultParams[k]; ok {
							crdParam.ResolvedValue = fmt.Sprintf("%v", rv)
						}
					}
					opData.CRDOutputParams = append(opData.CRDOutputParams, crdParam)
				}
			}
		}

		if err != nil && listErr != nil && descErr != nil {
			return operatorDataMsg{err: fmt.Errorf("failed to fetch operator data: %w", err)}
		}

		return operatorDataMsg{operatorData: opData}
	}
}

func (m operatorDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.wfErrors.refreshing || isWorkflowInProgress(m.getWfEvents()) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case operatorDataMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err
			return m, nil
		}
		m.operatorData = msg.operatorData
		if m.operatorData != nil {
			m.inputTree = buildOperatorParamTree(m.operatorData.InputParams)
			m.outputTree = buildOperatorOutputParamTree(m.operatorData.OutputParams)
			m.crdOutputTree = buildOperatorCRDOutputParamTree(m.operatorData.CRDOutputParams)
		}
		return m, scheduleResourceWorkflowRefreshIfNeeded(m.debugData, m.node)

	case wfEventsRefreshTickMsg:
		return m, handleResourceWorkflowRefreshTick(m.debugData, m.node, m.wfErrors)
	case wfEventsRefreshMsg:
		return m, handleResourceWorkflowRefresh(m.debugData, m.node, m.wfErrors, msg)
	case wfCountdownTickMsg:
		return m, handleResourceWorkflowCountdown(m.debugData, m.node)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalText = ""
				m.wfErrors.modalTitle = ""
				m.wfErrors.modalScroll = 0
				return m, nil
			}
			return m, func() tea.Msg { return backToDagMsg{} }
		case "tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab + 1) % opNumTabs
			return m, nil
		case "shift+tab":
			if m.wfErrors.modalText != "" {
				return m, nil
			}
			m.activeTab = (m.activeTab - 1 + opNumTabs) % opNumTabs
			return m, nil
		case "up", "k":
			if m.wfErrors.modalText != "" {
				if m.wfErrors.modalScroll > 0 {
					m.wfErrors.modalScroll--
				}
				return m, nil
			}
			switch m.activeTab {
			case opTabInputVars:
				m.inputCursor, m.inputScroll = moveResourceDetailTreeUp(m.inputTree, m.inputCursor, m.inputScroll, m.opVisibleRows())
			case opTabOutputVars:
				m.outputCursor, m.outputScroll = moveResourceDetailTreeUp(m.outputTree, m.outputCursor, m.outputScroll, m.opVisibleRows())
			case opTabCRDOutputVars:
				m.crdOutputCursor, m.crdOutputScroll = moveResourceDetailTreeUp(m.crdOutputTree, m.crdOutputCursor, m.crdOutputScroll, m.opVisibleRows())
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor > 0 {
					m.wfErrors.cursor--
				}
				_ = items
			}
		case "down", "j":
			if m.wfErrors.modalText != "" {
				m.wfErrors.modalScroll++
				maxScroll := wfEventModalMaxScroll(m.wfErrors, m.width, m.height)
				if m.wfErrors.modalScroll > maxScroll {
					m.wfErrors.modalScroll = maxScroll
				}
				return m, nil
			}
			switch m.activeTab {
			case opTabInputVars:
				m.inputCursor, m.inputScroll = moveResourceDetailTreeDown(m.inputTree, m.inputCursor, m.inputScroll, m.opVisibleRows())
			case opTabOutputVars:
				m.outputCursor, m.outputScroll = moveResourceDetailTreeDown(m.outputTree, m.outputCursor, m.outputScroll, m.opVisibleRows())
			case opTabCRDOutputVars:
				m.crdOutputCursor, m.crdOutputScroll = moveResourceDetailTreeDown(m.crdOutputTree, m.crdOutputCursor, m.crdOutputScroll, m.opVisibleRows())
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items)-1 {
					m.wfErrors.cursor++
				}
				_ = items
			}
		case "enter":
			switch m.activeTab {
			case opTabInputVars:
				toggleResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case opTabOutputVars:
				toggleResourceDetailTreeNode(m.outputTree, m.outputCursor)
			case opTabCRDOutputVars:
				toggleResourceDetailTreeNode(m.crdOutputTree, m.crdOutputCursor)
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				if m.wfErrors.cursor < len(items) {
					item := items[m.wfErrors.cursor]
					if item.event != nil {
						m.wfErrors.modalText = formatEventDetail(item.event)
						m.wfErrors.modalTitle = extractEventAction(item.event.Message)
						m.wfErrors.modalScroll = 0
					}
				}
			}
		case "right", "l":
			switch m.activeTab {
			case opTabInputVars:
				expandResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case opTabOutputVars:
				expandResourceDetailTreeNode(m.outputTree, m.outputCursor)
			case opTabCRDOutputVars:
				expandResourceDetailTreeNode(m.crdOutputTree, m.crdOutputCursor)
			}
		case "left", "h":
			switch m.activeTab {
			case opTabInputVars:
				collapseResourceDetailTreeNode(m.inputTree, m.inputCursor)
			case opTabOutputVars:
				collapseResourceDetailTreeNode(m.outputTree, m.outputCursor)
			case opTabCRDOutputVars:
				collapseResourceDetailTreeNode(m.crdOutputTree, m.crdOutputCursor)
			}
		case "pgup":
			switch m.activeTab {
			case opTabWfErrors:
				m.wfErrors.cursor -= m.opVisibleRows()
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "pgdown":
			switch m.activeTab {
			case opTabWfErrors:
				items := flattenWfEventItems(m.getWfEvents())
				m.wfErrors.cursor += m.opVisibleRows()
				if m.wfErrors.cursor >= len(items) {
					m.wfErrors.cursor = len(items) - 1
				}
				if m.wfErrors.cursor < 0 {
					m.wfErrors.cursor = 0
				}
			}
		case "y":
			content := m.opCopyableContent()
			if content != "" {
				return m, copyToClipboardCmd(content)
			}
		}

	case clipboardResultMsg:
		if msg.err != nil {
			m.clipboardMsg = fmt.Sprintf("✗ %v", msg.err)
		} else {
			m.clipboardMsg = "✓ Copied to clipboard"
		}
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearClipboardMsg{} })
	case clearClipboardMsg:
		m.clipboardMsg = ""
	}
	return m, nil
}

func (m operatorDetailModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.wfErrors.modalText != "" {
		return renderWfEventModal(m.wfErrors, m.width, m.height)
	}

	header := m.renderOpHeader()
	tabs := m.renderOpTabsWithBody()
	footer := m.renderOpFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, footer)
}

func (m operatorDetailModel) renderOpHeader() string {
	return renderResourceDetailHeader(m.width, m.node)
}

func (m operatorDetailModel) renderOpTabsWithBody() string {
	return renderResourceDetailTabsWithBody(m.width, m.opBodyHeight(), opTabNames, m.activeTab, m.getOpTabContent())
}

func (m operatorDetailModel) getOpTabContent() string {
	switch m.activeTab {
	case opTabInputVars:
		return m.renderInputVarsTab()
	case opTabOutputVars:
		return m.renderOutputVarsTab()
	case opTabCRDOutputVars:
		return m.renderCRDOutputVarsTab()
	case opTabWfErrors:
		return m.renderOpWfErrorsTab()
	}
	return ""
}

func (m operatorDetailModel) renderInputVarsTab() string {
	return m.renderParamTreeTab("Input Parameters", m.inputTree, m.inputCursor, m.inputScroll)
}

func (m operatorDetailModel) renderOutputVarsTab() string {
	return m.renderParamTreeTab("Output Parameters", m.outputTree, m.outputCursor, m.outputScroll)
}

func (m operatorDetailModel) renderCRDOutputVarsTab() string {
	return m.renderParamTreeTab("Operator CRD Outputs", m.crdOutputTree, m.crdOutputCursor, m.crdOutputScroll)
}

func (m operatorDetailModel) renderParamTreeTab(title string, tree []outputNode, cursor, scroll int) string {
	return renderResourceDetailParamTreeTab(title, tree, cursor, scroll, m.opVisibleRows(), m.opContentWidth(), m.loading, m.spinner.View(), m.loadErr, nil, true)
}

func (m operatorDetailModel) renderOpWfErrorsTab() string {
	return renderResourceWorkflowEventsTab(m.debugData, m.node, m.wfErrors, m.opBodyHeight(), m.opContentWidth(), m.spinner.View())
}

func (m operatorDetailModel) renderOpFooter() string {
	var text string
	switch m.activeTab {
	case opTabInputVars, opTabOutputVars, opTabCRDOutputVars:
		if (m.activeTab == opTabInputVars && len(m.inputTree) > 0) ||
			(m.activeTab == opTabOutputVars && len(m.outputTree) > 0) ||
			(m.activeTab == opTabCRDOutputVars && len(m.crdOutputTree) > 0) {
			text = "↑↓: navigate  ←→/enter: expand/collapse  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
		} else {
			text = "tab/shift+tab: switch tabs  esc: back  q: quit"
		}
	case opTabWfErrors:
		text = "↑↓/pgup/pgdn: scroll  y: copy  tab/shift+tab: switch tabs  esc: back  q: quit"
	default:
		text = "tab/shift+tab: switch tabs  esc: back  q: quit"
	}
	return renderResourceDetailFooter(m.width, m.clipboardMsg, text)
}

func (m operatorDetailModel) opCopyableContent() string {
	switch m.activeTab {
	case opTabInputVars:
		if m.operatorData != nil && len(m.operatorData.InputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.InputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabOutputVars:
		if m.operatorData != nil && len(m.operatorData.OutputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.OutputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabCRDOutputVars:
		if m.operatorData != nil && len(m.operatorData.CRDOutputParams) > 0 {
			raw, err := json.Marshal(m.operatorData.CRDOutputParams)
			if err == nil {
				return string(raw)
			}
		}
	case opTabWfErrors:
		return workflowEventsCopyText(m.getWfEvents())
	}
	return ""
}

func (m operatorDetailModel) opBodyHeight() int {
	return resourceDetailBodyHeight(m.height)
}

func (m operatorDetailModel) opVisibleRows() int {
	return resourceDetailVisibleRows(m.height)
}

func (m operatorDetailModel) opContentWidth() int {
	return resourceDetailContentWidth(m.width)
}

func (m operatorDetailModel) getWfEvents() *ResourceWorkflowSteps {
	return getResourceWorkflowEvents(m.debugData, m.node)
}

// buildOperatorParamTree converts input parameters to a navigable tree of outputNodes.
func buildOperatorParamTree(params []OperatorInputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		displayName := p.Key
		if p.DisplayName != "" && p.DisplayName != p.Key {
			displayName = fmt.Sprintf("%s (%s)", p.Key, p.DisplayName)
		}

		value := p.ResolvedValue
		if value == "" {
			value = p.DefaultValue
		}

		node := outputNode{
			key:      displayName,
			value:    value,
			nodeType: "string",
			depth:    0,
		}
		roots = append(roots, node)
	}
	return roots
}

// buildOperatorOutputParamTree converts output parameters to a navigable tree of outputNodes.
func buildOperatorOutputParamTree(params []OperatorOutputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		displayName := p.Key
		if p.DisplayName != "" && p.DisplayName != p.Key {
			displayName = fmt.Sprintf("%s (%s)", p.Key, p.DisplayName)
		}

		value := p.ResolvedValue
		if value == "" {
			value = p.Value
		}

		node := outputNode{
			key:      displayName,
			value:    value,
			nodeType: "string",
			depth:    0,
		}
		roots = append(roots, node)
	}
	return roots
}

// buildOperatorCRDOutputParamTree converts CRD output parameters to a navigable tree of outputNodes.
func buildOperatorCRDOutputParamTree(params []OperatorCRDOutputParam) []outputNode {
	if len(params) == 0 {
		return nil
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	var roots []outputNode
	for _, p := range params {
		value := p.ResolvedValue
		if value == "" {
			value = p.Value
		}

		node := outputNode{
			key:      p.Key,
			value:    value,
			nodeType: "string",
			depth:    0,
		}
		roots = append(roots, node)
	}
	return roots
}
