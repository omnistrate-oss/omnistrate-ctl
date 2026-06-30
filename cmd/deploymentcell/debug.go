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
	"gopkg.in/yaml.v3"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const (
	artifactHelmValuesRendered         = "HELM_VALUES_RENDERED"
	artifactKubernetesManifestRendered = "KUBERNETES_MANIFEST_RENDERED"
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

// deploymentCellDebugModel is the amenity list screen. Selecting an amenity
// (enter) opens a full-screen, scrollable detail view; esc returns here.
type deploymentCellDebugModel struct {
	data     deploymentCellDebugData
	selected int
	scroll   int
	width    int
	height   int
	inDetail bool
	detail   amenityDetailModel
}

func newDeploymentCellDebugModel(data deploymentCellDebugData) deploymentCellDebugModel {
	return deploymentCellDebugModel{data: data}
}

// backToListMsg signals that the detail screen wants to return to the list.
type backToListMsg struct{}

func (m deploymentCellDebugModel) Init() tea.Cmd {
	return nil
}

func (m deploymentCellDebugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(backToListMsg); ok {
		m.inDetail = false
		return m, nil
	}

	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}

	if m.inDetail {
		detail, cmd := m.detail.Update(msg)
		m.detail = detail
		return m, cmd
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		return m.updateList(key)
	}
	return m, nil
}

func (m deploymentCellDebugModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
		(&m).normalizeListViewport()
	case "down", "j":
		if m.selected < len(m.data.AmenityStatuses)-1 {
			m.selected++
		}
		(&m).normalizeListViewport()
	case "enter":
		if len(m.data.AmenityStatuses) > 0 {
			status := m.data.AmenityStatuses[m.selected]
			m.detail = newAmenityDetailModel(m.data, status, m.width, m.height)
			m.inDetail = true
		}
	}
	return m, nil
}

// normalizeListViewport keeps the selected amenity within the visible window.
func (m *deploymentCellDebugModel) normalizeListViewport() {
	visible := m.listBodyHeight()
	if m.selected < m.scroll {
		m.scroll = m.selected
	} else if m.selected >= m.scroll+visible {
		m.scroll = m.selected - visible + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// listBodyHeight is the number of amenity rows visible at once: terminal height
// minus the title + template header (2), the footer (1) and the list border (2).
func (m deploymentCellDebugModel) listBodyHeight() int {
	h := m.height - 5
	if h < 1 {
		h = 1
	}
	return h
}

func (m deploymentCellDebugModel) View() string {
	if m.inDetail {
		return m.detail.View()
	}
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if len(m.data.AmenityStatuses) == 0 {
		return "No amenity state rows found for this deployment cell.\n\nq: quit\n"
	}

	header := titleStyle.Render("Deployment Cell " + m.data.DeploymentCellID)
	template := m.templateLine()
	list := m.amenityListView()
	footer := mutedStyle.Render("↑↓: select   enter: open   q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, template, list, footer)
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
	bodyH := m.listBodyHeight()

	scroll := m.scroll
	if maxScroll := len(m.data.AmenityStatuses) - bodyH; scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + bodyH
	if end > len(m.data.AmenityStatuses) {
		end = len(m.data.AmenityStatuses)
	}

	width := m.width - 2
	if width < 20 {
		width = 20
	}

	const tagText = "managed"
	const tagCols = len(tagText) + 1 // tag plus a leading space

	var b strings.Builder
	for i := scroll; i < end; i++ {
		status := m.data.AmenityStatuses[i]
		prefix := "  "
		style := rowStyle
		selected := i == m.selected
		if selected {
			prefix = "> "
			style = selectedRowStyle
		}
		line := fmt.Sprintf("%s%-32s %-20s %s", prefix, truncate(status.Name, 32), truncate(status.Type, 20), status.DesiredStatus)

		if isManaged(status) {
			// Reserve space on the right for a green "managed" tag.
			left := padOrTruncate(line, width-tagCols) + " "
			if selected {
				// Whole row is highlighted; keep the tag inline.
				b.WriteString(style.Render(left + tagText))
			} else {
				b.WriteString(style.Render(left) + managedTagStyle.Render(tagText))
			}
		} else {
			b.WriteString(style.Render(padOrTruncate(line, width)))
		}
		b.WriteString("\n")
	}
	// Pad to a stable height so the footer never moves.
	for i := end - scroll; i < bodyH; i++ {
		b.WriteString(strings.Repeat(" ", width))
		b.WriteString("\n")
	}

	return listPanelStyle.Width(width).Render(strings.TrimRight(b.String(), "\n"))
}

// artifactPayload returns the decoded payload for an amenity artifact and
// whether a matching artifact was found and decoded successfully.
func artifactPayload(data deploymentCellDebugData, amenityName string, artifactKind string) (string, bool) {
	for _, artifact := range data.AmenityArtifacts {
		if artifact.AmenityName != amenityName || artifact.ArtifactKind != artifactKind || artifact.PayloadBase64 == nil {
			continue
		}
		return decodeBase64(*artifact.PayloadBase64)
	}
	return fmt.Sprintf("No %s artifact found.", artifactKind), false
}

// toPrettyJSON parses raw as JSON (falling back to YAML) and re-marshals it as
// indented JSON. It only converts structured payloads (objects/arrays); bare
// scalars and unparseable text are returned unchanged with ok=false so
// placeholder messages and Go templates are left untouched.
func toPrettyJSON(raw string) (string, bool) {
	var obj interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
			return raw, false
		}
	}

	switch obj.(type) {
	case map[string]interface{}, []interface{}:
		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return raw, false
		}
		return string(pretty), true
	default:
		return raw, false
	}
}

// workflowLine renders the workflow/run identifiers for an amenity, omitting
// whichever IDs are absent. Returns an empty string when neither exists.
func workflowLine(status deploymentCellAmenityStatus) string {
	var parts []string
	if status.WorkflowID != nil && *status.WorkflowID != "" {
		parts = append(parts, "workflow="+*status.WorkflowID)
	}
	if status.WorkflowRunID != nil && *status.WorkflowRunID != "" {
		parts = append(parts, "run="+*status.WorkflowRunID)
	}
	return strings.Join(parts, " ")
}

func decodeBase64(value string) (string, bool) {
	if value == "" {
		return "No payload found.", false
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Sprintf("Failed to decode payload: %v", err), false
	}
	return string(decoded), true
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

// padOrTruncate forces a string to exactly width display columns (runes), so
// styled rows fill the panel width and never wrap.
func padOrTruncate(s string, width int) string {
	if width < 0 {
		width = 0
	}
	r := []rune(s)
	if len(r) > width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

// omnistrateGreen approximates the green of the omnistrate.com logo.
var omnistrateGreen = lipgloss.Color("42")

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	mutedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	clipStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	rowStyle          = lipgloss.NewStyle()
	selectedRowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	listPanelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	detailWindowStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1)
	activeTabStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	inactiveTabStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	managedBadgeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).Background(omnistrateGreen).Padding(0, 1)
	managedTagStyle   = lipgloss.NewStyle().Bold(true).Foreground(omnistrateGreen)
)
