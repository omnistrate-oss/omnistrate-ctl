package deploymentcell

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const (
	artifactHelmValuesTemplate         = "HELM_VALUES_TEMPLATE"
	artifactHelmValuesRendered         = "HELM_VALUES_RENDERED"
	artifactKubernetesManifestTemplate = "KUBERNETES_MANIFEST_TEMPLATE"
	artifactKubernetesManifestRendered = "KUBERNETES_MANIFEST_RENDERED"
	artifactKubernetesManifestStatus   = "KUBERNETES_MANIFEST_STATUS"
	artifactGenericCRDDescribeResult   = "GENERIC_CRD_DESCRIBE_RESULT"
	amenityTypeHelm                    = "Helm"
	amenityTypeKubernetesManifest      = "KubernetesManifest"
)

var outputDir string

var debugCmd = &cobra.Command{
	Use:          "debug",
	Short:        "Debug deployment cell amenities",
	Long:         `Debug deployment cell amenity template resolution, per-amenity status, rendered artifacts, and workflow logs. Use --output=json for automation.`,
	RunE:         runDebugDeploymentCell,
	SilenceUsage: true,
	Example: `  omnistrate-ctl deployment-cell debug --id <deployment-cell-id>
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output json
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output-dir ./debug-output`,
}

type deploymentCellDebugData struct {
	DeploymentCellID              string                          `json:"deploymentCellId"`
	CustomHelmExecutionLogsBase64 map[string]string               `json:"customHelmExecutionLogsBase64,omitempty"`
	AmenityStatuses               []deploymentCellAmenityStatus   `json:"amenityStatuses,omitempty"`
	AmenityArtifacts              []deploymentCellAmenityArtifact `json:"amenityArtifacts,omitempty"`
	Template                      *deploymentCellTemplateDebug    `json:"template,omitempty"`
}

type deploymentCellAmenityStatus struct {
	Name                  string  `json:"name"`
	Type                  string  `json:"type"`
	IsManaged             *bool   `json:"isManaged,omitempty"`
	DesiredStatus         string  `json:"desiredStatus"`
	Source                *string `json:"source,omitempty"`
	SourceEnvironmentType *string `json:"sourceEnvironmentType,omitempty"`
	SourceCloudProviderID *string `json:"sourceCloudProviderId,omitempty"`
	Generation            int64   `json:"generation"`
	WorkflowID            *string `json:"workflowId,omitempty"`
	WorkflowRunID         *string `json:"workflowRunId,omitempty"`
	LastError             *string `json:"lastError,omitempty"`
}

type deploymentCellAmenityArtifact struct {
	AmenityName     string  `json:"amenityName"`
	ArtifactKind    string  `json:"artifactKind"`
	Generation      int64   `json:"generation"`
	ContentType     *string `json:"contentType,omitempty"`
	ContentEncoding *string `json:"contentEncoding,omitempty"`
	Sha256          *string `json:"sha256,omitempty"`
	SizeBytes       *int64  `json:"sizeBytes,omitempty"`
	SecretMasked    *bool   `json:"secretMasked,omitempty"`
	PayloadBase64   *string `json:"payloadBase64,omitempty"`
}

type deploymentCellTemplateDebug struct {
	RequestedEnvironmentType         *string `json:"requestedEnvironmentType,omitempty"`
	EffectiveEnvironmentType         *string `json:"effectiveEnvironmentType,omitempty"`
	PerEnvironmentHostClusterEnabled *bool   `json:"perEnvironmentHostClusterEnabled,omitempty"`
}

func init() {
	debugCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	debugCmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", "Optional directory to export Helm logs")
	_ = debugCmd.MarkFlagRequired("id")
}

func runDebugDeploymentCell(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	outputFormat, _ := cmd.Flags().GetString("output")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	debugData, err := fetchDeploymentCellDebugData(context.Background(), token, deploymentCellID)
	if err != nil {
		return err
	}

	if outputDir != "" {
		if err = exportHelmLogs(outputDir, debugData.CustomHelmExecutionLogsBase64); err != nil {
			return err
		}
	}

	if strings.EqualFold(outputFormat, "json") {
		out, marshalErr := json.MarshalIndent(debugData, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Println(string(out))
		return nil
	}

	program := tea.NewProgram(newDeploymentCellDebugModel(debugData), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func fetchDeploymentCellDebugData(ctx context.Context, token string, deploymentCellID string) (deploymentCellDebugData, error) {
	debugResult, err := dataaccess.DebugHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		return deploymentCellDebugData{}, fmt.Errorf("failed to retrieve debug data: %w", err)
	}

	resultMap, err := debugResult.ToMap()
	if err != nil {
		return deploymentCellDebugData{}, err
	}

	raw, err := json.Marshal(resultMap)
	if err != nil {
		return deploymentCellDebugData{}, err
	}

	var debugData deploymentCellDebugData
	if err = json.Unmarshal(raw, &debugData); err != nil {
		return deploymentCellDebugData{}, err
	}
	debugData.DeploymentCellID = deploymentCellID

	sort.Slice(debugData.AmenityStatuses, func(i, j int) bool {
		return debugData.AmenityStatuses[i].Name < debugData.AmenityStatuses[j].Name
	})

	return debugData, nil
}

func exportHelmLogs(dir string, logs map[string]string) error {
	if len(logs) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for serviceName, logContent := range logs {
		safeServiceName := strings.NewReplacer("/", "_", ":", "_").Replace(serviceName)
		filePath := filepath.Join(dir, fmt.Sprintf("helm-logs-%s-%s.txt", safeServiceName, time.Now().Format("20060102-150405")))

		logContentBytes, err := base64.StdEncoding.DecodeString(logContent)
		if err != nil {
			return fmt.Errorf("failed to decode base64 content for service %s: %w", serviceName, err)
		}

		if err = os.WriteFile(filePath, logContentBytes, 0600); err != nil {
			return fmt.Errorf("failed to write logs for service %s: %w", serviceName, err)
		}
	}

	return nil
}

type deploymentCellDebugModel struct {
	data     deploymentCellDebugData
	selected int
	viewMode int
	width    int
	height   int
}

func newDeploymentCellDebugModel(data deploymentCellDebugData) deploymentCellDebugModel {
	return deploymentCellDebugModel{data: data}
}

func (m deploymentCellDebugModel) Init() tea.Cmd {
	return nil
}

func (m deploymentCellDebugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.viewMode = 0
			}
		case "down", "j":
			if m.selected < len(m.data.AmenityStatuses)-1 {
				m.selected++
				m.viewMode = 0
			}
		case "tab":
			m.viewMode = (m.viewMode + 1) % len(m.availableViews())
		}
	}
	return m, nil
}

func (m deploymentCellDebugModel) View() string {
	if len(m.data.AmenityStatuses) == 0 {
		return "No amenity state rows found for this deployment cell.\n\nq: quit\n"
	}

	header := titleStyle.Render("Deployment Cell " + m.data.DeploymentCellID)
	template := m.templateLine()
	left := m.amenityListView()
	right := m.detailView()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := mutedStyle.Render("up/down: select  tab: toggle template/rendered/log/status  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, template, "", body, "", footer)
}

func (m deploymentCellDebugModel) templateLine() string {
	if m.data.Template == nil {
		return mutedStyle.Render("template: unavailable")
	}
	requested := valueOrDefault(m.data.Template.RequestedEnvironmentType, "-")
	effective := valueOrDefault(m.data.Template.EffectiveEnvironmentType, "-")
	perEnv := false
	if m.data.Template.PerEnvironmentHostClusterEnabled != nil {
		perEnv = *m.data.Template.PerEnvironmentHostClusterEnabled
	}
	return mutedStyle.Render(fmt.Sprintf("template: requested=%s effective=%s per-env=%t", requested, effective, perEnv))
}

func (m deploymentCellDebugModel) amenityListView() string {
	var b strings.Builder
	for i, status := range m.data.AmenityStatuses {
		prefix := "  "
		style := rowStyle
		if i == m.selected {
			prefix = "> "
			style = selectedRowStyle
		}
		line := fmt.Sprintf("%s%-28s %-18s %s", prefix, truncate(status.Name, 28), status.Type, status.DesiredStatus)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	return panelStyle.Width(68).Render(b.String())
}

func (m deploymentCellDebugModel) detailView() string {
	status := m.data.AmenityStatuses[m.selected]
	views := m.availableViews()
	if m.viewMode >= len(views) {
		m.viewMode = 0
	}
	view := views[m.viewMode]

	var b strings.Builder
	b.WriteString(titleStyle.Render(status.Name))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("type=%s status=%s generation=%d\n", status.Type, status.DesiredStatus, status.Generation))
	if status.Source != nil || status.SourceEnvironmentType != nil {
		b.WriteString(fmt.Sprintf("source=%s env=%s cloud=%s\n",
			valueOrDefault(status.Source, "-"),
			valueOrDefault(status.SourceEnvironmentType, "-"),
			valueOrDefault(status.SourceCloudProviderID, "-"),
		))
	}
	if status.WorkflowID != nil || status.WorkflowRunID != nil {
		b.WriteString(fmt.Sprintf("workflow=%s run=%s\n",
			valueOrDefault(status.WorkflowID, "-"),
			valueOrDefault(status.WorkflowRunID, "-"),
		))
	}
	if status.LastError != nil && *status.LastError != "" {
		b.WriteString(errorStyle.Render(*status.LastError))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.payloadForView(status, view))

	return panelStyle.Width(100).Render(b.String())
}

func (m deploymentCellDebugModel) availableViews() []string {
	if len(m.data.AmenityStatuses) == 0 {
		return []string{"status"}
	}
	status := m.data.AmenityStatuses[m.selected]
	switch status.Type {
	case amenityTypeHelm:
		return []string{"rendered values", "template values", "helm logs"}
	case amenityTypeKubernetesManifest:
		return []string{"rendered manifest", "template manifest", "cluster status"}
	default:
		return []string{"status"}
	}
}

func (m deploymentCellDebugModel) payloadForView(status deploymentCellAmenityStatus, view string) string {
	switch view {
	case "rendered values":
		return m.artifactPayload(status.Name, artifactHelmValuesRendered)
	case "template values":
		return m.artifactPayload(status.Name, artifactHelmValuesTemplate)
	case "helm logs":
		return decodeBase64String(m.data.CustomHelmExecutionLogsBase64[status.Name])
	case "rendered manifest":
		return m.artifactPayload(status.Name, artifactKubernetesManifestRendered)
	case "template manifest":
		return m.artifactPayload(status.Name, artifactKubernetesManifestTemplate)
	case "cluster status":
		if payload := m.artifactPayload(status.Name, artifactKubernetesManifestStatus); payload != "" {
			return payload
		}
		return m.artifactPayload(status.Name, artifactGenericCRDDescribeResult)
	default:
		return "No detail available."
	}
}

func (m deploymentCellDebugModel) artifactPayload(amenityName string, artifactKind string) string {
	for _, artifact := range m.data.AmenityArtifacts {
		if artifact.AmenityName != amenityName || artifact.ArtifactKind != artifactKind || artifact.PayloadBase64 == nil {
			continue
		}
		return decodeBase64String(*artifact.PayloadBase64)
	}
	return fmt.Sprintf("No %s artifact found.", artifactKind)
}

func decodeBase64String(value string) string {
	if value == "" {
		return "No payload found."
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Sprintf("Failed to decode payload: %v", err)
	}
	return string(decoded)
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func truncate(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	mutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	rowStyle         = lipgloss.NewStyle()
	selectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	panelStyle       = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)
