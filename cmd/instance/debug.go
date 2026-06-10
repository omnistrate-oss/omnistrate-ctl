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
	ProductTierID     string                        `json:"productTierId,omitempty"`
	TierVersion       string                        `json:"tierVersion,omitempty"`
	Token             string                        `json:"-"`
	ResultParams      map[string]interface{}        `json:"-"`
	InputParams       map[string]interface{}        `json:"-"`
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
		// Extract result_params (resolved output values) from consumption result
		var resultParams map[string]interface{}
		consumptionResult := instanceData.GetConsumptionResourceInstanceResult()
		if rp := consumptionResult.GetResultParams(); rp != nil {
			if rpMap, ok := rp.(map[string]interface{}); ok {
				resultParams = rpMap
			}
		}

		// Extract input_params (resolved input values) from instance
		var inputParams map[string]interface{}
		if ip := instanceData.GetInputParams(); ip != nil {
			if ipMap, ok := ip.(map[string]interface{}); ok {
				inputParams = ipMap
			}
		}

		return debugDataMsg{
			data: DebugData{
				InstanceID:    instanceID,
				PlanDAG:       planDAG,
				ServiceID:     serviceID,
				EnvironmentID: environmentID,
				ProductTierID: instanceData.ProductTierId,
				TierVersion:   instanceData.TierVersion,
				Token:         token,
				ResultParams:  resultParams,
				InputParams:   inputParams,
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

	// Extract result_params (resolved output values) from consumption result
	var resultParams map[string]interface{}
	consumptionResult := instanceData.GetConsumptionResourceInstanceResult()
	if rp := consumptionResult.GetResultParams(); rp != nil {
		if rpMap, ok := rp.(map[string]interface{}); ok {
			resultParams = rpMap
		}
	}

	// Extract input_params (resolved input values) from instance
	var inputParams map[string]interface{}
	if ip := instanceData.GetInputParams(); ip != nil {
		if ipMap, ok := ip.(map[string]interface{}); ok {
			inputParams = ipMap
		}
	}

	data := DebugData{
		InstanceID:    instanceID,
		ServiceID:     serviceID,
		EnvironmentID: environmentID,
		ProductTierID: instanceData.ProductTierId,
		TierVersion:   instanceData.TierVersion,
		PlanDAG:       planDAG,
		ResultParams:  resultParams,
		InputParams:   inputParams,
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
// It collects helm data (logs, values), terraform data (progress, history, files, logs),
// operator data (input/output parameters), and compose data (input/output parameters)
// for each resource in the plan DAG.
// Errors for individual resources or data sources are handled gracefully — partial data
// is returned rather than failing the entire operation.
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

	// Extract result_params for resolving output parameter values
	var resultParams map[string]interface{}
	consumptionResult := instanceData.GetConsumptionResourceInstanceResult()
	if rp := consumptionResult.GetResultParams(); rp != nil {
		if rpMap, ok := rp.(map[string]interface{}); ok {
			resultParams = rpMap
		}
	}

	// Extract input_params for resolving input parameter values
	var inputParams map[string]interface{}
	if ip := instanceData.GetInputParams(); ip != nil {
		if ipMap, ok := ip.(map[string]interface{}); ok {
			inputParams = ipMap
		}
	}

	// Collect helm debug data from the DebugResourceInstance API
	collectHelmDebugInfo(ctx, token, serviceID, environmentID, instanceID, planDAG, instanceData, inputParams, resultParams, result)

	// Collect terraform debug data from k8s ConfigMaps
	collectTerraformDebugInfo(ctx, token, instanceData, instanceID, planDAG, result)

	// Collect operator debug data (input/output parameters) for non-helm, non-terraform resources
	collectOperatorDebugInfo(ctx, token, serviceID, planDAG, instanceData, inputParams, resultParams, result)

	// Collect compose debug data (input/output parameters) for compose resources
	collectComposeDebugInfo(ctx, token, serviceID, planDAG, instanceData, inputParams, resultParams, result)

	// Remove entries that have no debug data
	for key, info := range result {
		if !info.hasData() {
			delete(result, key)
		}
	}

	return result
}

// collectHelmDebugInfo fetches helm debug data (logs, chart values) and input/output parameters for all helm resources.
func collectHelmDebugInfo(ctx context.Context, token, serviceID, environmentID, instanceID string, planDAG *PlanDAG, instanceData *openapiclientfleet.ResourceInstance, inputParams map[string]interface{}, resultParams map[string]interface{}, result map[string]*ResourceDebugInfo) {
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

			// Find the node ID for this resource to fetch input/output params
			var nodeID string
			if planDAG != nil {
				for _, node := range planDAG.Nodes {
					nodeKey := node.Key
					if nodeKey == "" {
						nodeKey = node.ID
					}
					if nodeKey == resourceKey {
						nodeID = node.ID
						break
					}
				}
			}

			if nodeID != "" && instanceData != nil {
				// Fetch input parameters
				fetchedInputParams, _ := fetchInputParams(
					ctx, token, serviceID, nodeID,
					instanceData.ProductTierId, instanceData.TierVersion,
					inputParams,
				)
				info.Helm.InputParams = fetchedInputParams

				// Fetch output parameters
				outputParams, _ := fetchOutputParams(
					ctx, token, serviceID, nodeID,
					instanceData.ProductTierId, instanceData.TierVersion,
					resultParams,
				)
				info.Helm.OutputParams = outputParams
			}
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
		// Plan previews: dedicated tf-plan-* CMs first, then state CM fallback.
		// All lookups are best-effort.
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
			if len(stateData.PlanPreviewDiffs) > 0 {
				info.TerraformPlanPreviewDiff = stateData.PlanPreviewDiffs
			}
			if len(stateData.PreviewErrors) > 0 {
				info.TerraformPlanPreviewError = stateData.PreviewErrors
			}
		}
	}
}

// collectOperatorDebugInfo fetches operator debug data (input/output parameters, CRD outputs)
// for operator-type resources.
func collectOperatorDebugInfo(ctx context.Context, token, serviceID string, planDAG *PlanDAG, instanceData *openapiclientfleet.ResourceInstance, inputParams map[string]interface{}, resultParams map[string]interface{}, result map[string]*ResourceDebugInfo) {
	for _, node := range planDAG.Nodes {
		lower := strings.ToLower(node.Type)
		if !strings.Contains(lower, "operator") {
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

		opData := &OperatorData{}

		// Fetch all input parameters
		fetchedInputParams, inputErr := fetchInputParams(
			ctx, token, serviceID, node.ID,
			instanceData.ProductTierId, instanceData.TierVersion,
			inputParams,
		)
		if inputErr == nil {
			opData.InputParams = fetchedInputParams
		}

		// Fetch exported output parameters
		outputParams, listErr := fetchOutputParams(
			ctx, token, serviceID, node.ID,
			instanceData.ProductTierId, instanceData.TierVersion,
			resultParams,
		)
		if listErr == nil {
			opData.OutputParams = outputParams
		}

		// Fetch CRD output parameters from DescribeResource (operatorCRDConfiguration.outputParameters)
		resourceResult, descErr := dataaccess.DescribeResource(
			ctx, token, serviceID, node.ID,
			&instanceData.ProductTierId, &instanceData.TierVersion,
		)
		if descErr == nil && resourceResult != nil {
			crdConfig, ok := resourceResult.GetOperatorCRDConfigurationOk()
			if ok && crdConfig != nil {
				crdOutputParams := crdConfig.GetOutputParameters()
				for k, v := range crdOutputParams {
					crdParam := OperatorCRDOutputParam{
						Key:   k,
						Value: v,
					}
					if resultParams != nil {
						if rv, ok := resultParams[k]; ok {
							crdParam.ResolvedValue = fmt.Sprintf("%v", rv)
						}
					}
					opData.CRDOutputParams = append(opData.CRDOutputParams, crdParam)
				}
			}
		}

		if len(opData.InputParams) > 0 || len(opData.OutputParams) > 0 || len(opData.CRDOutputParams) > 0 {
			info.Operator = opData
		}
	}
}

// collectComposeDebugInfo fetches compose debug data (input/output parameters)
// for compose-type resources.
func collectComposeDebugInfo(ctx context.Context, token, serviceID string, planDAG *PlanDAG, instanceData *openapiclientfleet.ResourceInstance, inputParams map[string]interface{}, resultParams map[string]interface{}, result map[string]*ResourceDebugInfo) {
	collectComposeDebugInfoWithFetchers(ctx, token, serviceID, planDAG, instanceData, inputParams, resultParams, result, fetchInputParams, fetchOutputParams)
}

type inputParamsFetcher func(context.Context, string, string, string, string, string, map[string]interface{}) ([]OperatorInputParam, error)
type outputParamsFetcher func(context.Context, string, string, string, string, string, map[string]interface{}) ([]OperatorOutputParam, error)

func collectComposeDebugInfoWithFetchers(ctx context.Context, token, serviceID string, planDAG *PlanDAG, instanceData *openapiclientfleet.ResourceInstance, inputParams map[string]interface{}, resultParams map[string]interface{}, result map[string]*ResourceDebugInfo, fetchInput inputParamsFetcher, fetchOutput outputParamsFetcher) {
	for _, node := range planDAG.Nodes {
		lower := strings.ToLower(node.Type)
		if !strings.Contains(lower, "compose") {
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

		cData := &ComposeData{}

		// Fetch all input parameters
		fetchedInputParams, inputErr := fetchInput(
			ctx, token, serviceID, node.ID,
			instanceData.ProductTierId, instanceData.TierVersion,
			inputParams,
		)
		if inputErr == nil {
			cData.InputParams = fetchedInputParams
		}

		// Fetch exported output parameters
		outputParams, outputErr := fetchOutput(
			ctx, token, serviceID, node.ID,
			instanceData.ProductTierId, instanceData.TierVersion,
			resultParams,
		)
		if outputErr == nil {
			cData.OutputParams = outputParams
		}

		if len(cData.InputParams) > 0 || len(cData.OutputParams) > 0 {
			info.Compose = cData
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
