package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

var debugCmd = &cobra.Command{
	Use:   "debug [instance-id]",
	Short: "Visualize the instance plan DAG",
	Long:  "Visualize the plan DAG for an instance based on its product tier version. Use --output=json for non-interactive output.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDebug,
	Example: `  omnistrate-ctl instance debug <instance-id>
  omnistrate-ctl instance debug <instance-id> --output=json`,
}

type DebugData struct {
	InstanceID        string                        `json:"instanceId"`
	PlanDAG           *PlanDAG                      `json:"planDag,omitempty"`
	ServiceID         string                        `json:"serviceId,omitempty"`
	EnvironmentID     string                        `json:"environmentId,omitempty"`
	Token             string                        `json:"-"`
	ResourceDebugInfo map[string]*ResourceDebugInfo `json:"resourceDebugInfo,omitempty"`
}

// Messages for the loading spinner model
type debugDataMsg struct {
	data DebugData
	err  error
}

type loadingModel struct {
	spinner    spinner.Model
	instanceID string
	status     string
	done       bool
	result     *debugDataMsg
}

func newLoadingModel(instanceID string) loadingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return loadingModel{
		spinner:    s,
		instanceID: instanceID,
		status:     "Fetching instance data...",
	}
}

func (m loadingModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m loadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case debugDataMsg:
		m.done = true
		m.result = &msg
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m loadingModel) View() string {
	return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.status)
}

func fetchDebugData(instanceID, token string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
		if err != nil {
			return debugDataMsg{err: fmt.Errorf("failed to get instance: %w", err)}
		}

		instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
		if err != nil {
			return debugDataMsg{err: fmt.Errorf("failed to describe resource instance: %w", err)}
		}

		planDAG, err := buildPlanDAG(ctx, token, serviceID, instanceData)
		if err != nil {
			planDAG = &PlanDAG{
				Errors: []string{err.Error()},
			}
		}
		return debugDataMsg{
			data: DebugData{
				InstanceID:    instanceID,
				PlanDAG:       planDAG,
				ServiceID:     serviceID,
				EnvironmentID: environmentID,
				Token:         token,
			},
		}
	}
}

func runDebug(cmd *cobra.Command, args []string) error {
	instanceID := args[0]

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("failed to get output flag: %w", err)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if output == "json" {
		return runDebugJSON(instanceID, token)
	}

	// Interactive mode: show spinner while loading
	model := newLoadingModel(instanceID)
	p := tea.NewProgram(model)

	go func() {
		fetchCmd := fetchDebugData(instanceID, token)
		msg := fetchCmd()
		p.Send(msg)
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("loading failed: %w", err)
	}

	m, ok := finalModel.(loadingModel)
	if !ok || m.result == nil {
		return fmt.Errorf("loading interrupted")
	}

	if m.result.err != nil {
		return m.result.err
	}

	return launchDebugTUI(m.result.data)
}

func runDebugJSON(instanceID, token string) error {
	ctx := context.Background()

	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to describe resource instance: %w", err)
	}

	planDAG, err := buildPlanDAG(ctx, token, serviceID, instanceData)
	if err != nil {
		planDAG = &PlanDAG{
			Errors: []string{err.Error()},
		}
	}
	if planDAG != nil {
		attachWorkflowProgress(ctx, token, serviceID, environmentID, instanceID, planDAG)
		// Enrich bootstrap steps with dependency timelines for all resources
		for resourceKey, steps := range planDAG.WorkflowStepsByKey {
			enrichBootstrapSteps(steps, resourceKey, planDAG)
		}
	}

	data := DebugData{
		InstanceID:    instanceID,
		ServiceID:     serviceID,
		EnvironmentID: environmentID,
		PlanDAG:       planDAG,
	}

	// Collect per-resource debug info (helm data, terraform progress/files/logs)
	if planDAG != nil {
		data.ResourceDebugInfo = collectResourceDebugInfo(ctx, token, serviceID, environmentID, instanceID, planDAG, instanceData)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal debug data to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// collectResourceDebugInfo fetches all per-resource debug data for the JSON output path.
// It collects helm data (logs, values) and terraform data (progress, history, files, logs)
// for each resource in the plan DAG. Errors for individual resources or data sources are
// handled gracefully — partial data is returned rather than failing the entire operation.
func collectResourceDebugInfo(ctx context.Context, token, serviceID, environmentID, instanceID string, planDAG *PlanDAG, instanceData *openapiclientfleet.ResourceInstance) map[string]*ResourceDebugInfo {
	result := make(map[string]*ResourceDebugInfo)
	if planDAG == nil || len(planDAG.Nodes) == 0 {
		return result
	}

	// Initialize entries for all visible nodes
	for _, node := range planDAG.Nodes {
		key := node.Key
		if key == "" {
			key = node.ID
		}
		result[key] = &ResourceDebugInfo{
			ResourceID:   node.ID,
			ResourceKey:  key,
			ResourceType: node.Type,
		}
	}

	// Collect helm debug data from the DebugResourceInstance API
	collectHelmDebugInfo(ctx, token, serviceID, environmentID, instanceID, result)

	// Collect terraform debug data from k8s ConfigMaps
	collectTerraformDebugInfo(ctx, token, instanceData, instanceID, planDAG, result)

	// Remove entries that have no debug data
	for key, info := range result {
		if !info.hasData() {
			delete(result, key)
		}
	}

	return result
}

// collectHelmDebugInfo fetches helm debug data (logs, chart values) for all helm resources.
func collectHelmDebugInfo(ctx context.Context, token, serviceID, environmentID, instanceID string, result map[string]*ResourceDebugInfo) {
	debugResult, err := dataaccess.DebugResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil || debugResult.ResourcesDebug == nil {
		return
	}

	for resourceKey, resourceDebugInfo := range *debugResult.ResourcesDebug {
		if resourceKey == "omnistrateobserv" {
			continue
		}

		info, exists := result[resourceKey]
		if !exists {
			continue
		}

		debugDataInterface, ok := resourceDebugInfo.GetDebugDataOk()
		if !ok || debugDataInterface == nil {
			continue
		}

		actualDebugData, ok := (*debugDataInterface).(map[string]interface{})
		if !ok {
			continue
		}

		// Check if it's a helm resource (has chart metadata)
		if _, hasChart := actualDebugData["chartRepoName"]; hasChart {
			info.Helm = parseHelmData(actualDebugData)
		}
	}
}

// collectTerraformDebugInfo fetches terraform debug data (progress, history, files, logs)
// for all terraform resources from k8s ConfigMaps.
func collectTerraformDebugInfo(ctx context.Context, token string, instanceData *openapiclientfleet.ResourceInstance, instanceID string, planDAG *PlanDAG, result map[string]*ResourceDebugInfo) {
	// Check if there are any terraform resources
	hasTerraform := false
	for _, node := range planDAG.Nodes {
		if strings.Contains(strings.ToLower(node.Type), "terraform") {
			hasTerraform = true
			break
		}
	}
	if !hasTerraform {
		return
	}

	// Load terraform configmap index once for all resources
	index, _, err := loadTerraformConfigMapIndexForInstance(ctx, token, instanceData, instanceID)
	if err != nil || index == nil {
		return
	}

	for _, node := range planDAG.Nodes {
		if !strings.Contains(strings.ToLower(node.Type), "terraform") {
			continue
		}

		key := node.Key
		if key == "" {
			key = node.ID
		}
		info, exists := result[key]
		if !exists {
			continue
		}

		// Get terraform files and logs from configmap data
		tfData := index.terraformDataForResource(node.ID)
		if tfData != nil {
			if len(tfData.Files) > 0 {
				info.TerraformFiles = tfData.Files
			}
			if len(tfData.Logs) > 0 {
				info.TerraformLogs = tfData.Logs
			}
		}

		// Get terraform progress, operation history, and plan previews.
		// Plan previews are checked in order: dedicated tf-plan-* CMs first, then state CM,
		// then tfData.Files as a last resort. All lookups are best-effort.
		stateData := extractTerraformStateData(index, instanceID, node.ID)
		if stateData != nil {
			if stateData.Progress != nil {
				info.TerraformProgress = stateData.Progress
			}
			if len(stateData.History) > 0 {
				info.TerraformHistory = stateData.History
			}
			if len(stateData.PlanPreviews) > 0 {
				info.TerraformPlanPreview = stateData.PlanPreviews
			}
			if len(stateData.PreviewErrors) > 0 {
				info.TerraformPlanPreviewError = stateData.PreviewErrors
			}
		}

		// Last resort: if plan previews weren't found via stateData but tfData.Files has them,
		// extract from there as well (belt-and-suspenders).
		if len(info.TerraformPlanPreview) == 0 && len(info.TerraformPlanPreviewError) == 0 && tfData != nil && len(tfData.Files) > 0 {
			previews, previewErrors := findAllPlanPreviews(tfData.Files)
			if len(previews) > 0 {
				info.TerraformPlanPreview = previews
			}
			if len(previewErrors) > 0 {
				info.TerraformPlanPreviewError = previewErrors
			}
		}
	}
}

func init() {
	debugCmd.Flags().StringP("output", "o", "interactive", "Output format (interactive|json)")
	debugCmd.AddCommand(debugHelmLogsCmd)
	debugCmd.AddCommand(debugHelmValuesCmd)
	debugCmd.AddCommand(debugTerraformFilesCmd)
	debugCmd.AddCommand(debugTerraformOutputsCmd)
}
