package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	InstanceID    string   `json:"instanceId"`
	PlanDAG       *PlanDAG `json:"planDag,omitempty"`
	ServiceID     string   `json:"-"`
	EnvironmentID string   `json:"-"`
	Token         string   `json:"-"`
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

		instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID, true)
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

	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID, true)
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
	}

	data := DebugData{
		InstanceID: instanceID,
		PlanDAG:    planDAG,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal debug data to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func init() {
	debugCmd.Flags().StringP("output", "o", "interactive", "Output format (interactive|json)")
	debugCmd.AddCommand(debugHelmLogsCmd)
	debugCmd.AddCommand(debugHelmValuesCmd)
	debugCmd.AddCommand(debugTerraformFilesCmd)
	debugCmd.AddCommand(debugTerraformOutputsCmd)
}
