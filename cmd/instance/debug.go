package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

var debugCmd = &cobra.Command{
	Use:     "debug [instance-id]",
	Short:   "Debug instance resources",
	Long:    `Debug instance resources with an interactive TUI showing helm charts, terraform files, and logs.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDebug,
	Example: `  omnistrate-ctl instance debug <instance-id>`,
}


type DebugData struct {
	InstanceID string         `json:"instanceId"`
	Resources  []ResourceInfo `json:"resources"`
}

type ResourceInfo struct {
	ID            string                             `json:"id"`
	Name          string                             `json:"name"`
	Type          string                             `json:"type"` // "helm" or "terraform"
	DebugData     interface{}                        `json:"debugData"`
	HelmData      *HelmData                          `json:"helmData,omitempty"`
	TerraformData *TerraformData                     `json:"terraformData,omitempty"`
	GenericData   *GenericData                       `json:"genericData,omitempty"` // For generic resources
	WorkflowEvents *dataaccess.WorkflowEventsByCategory `json:"workflowEvents,omitempty"` // Debug events
}

type GenericData struct {
	LiveLogs    []dataaccess.LogsStream    `json:"liveLogs"`
}

type HelmData struct {
	ChartRepoName string                 `json:"chartRepoName"`
	ChartRepoURL  string                 `json:"chartRepoURL"`
	ChartVersion  string                 `json:"chartVersion"`
	ChartValues   map[string]interface{} `json:"chartValues"`
	InstallLog    string                 `json:"installLog"`
	LiveLogs    []dataaccess.LogsStream    `json:"liveLogs"`

	Namespace     string                 `json:"namespace"`
	ReleaseName   string                 `json:"releaseName"`
}

type TerraformData struct {
	Files   map[string]string `json:"files"`
	Logs    map[string]string `json:"logs"`
	LiveLogs    []dataaccess.LogsStream    `json:"liveLogs"`
}

func runDebug(_ *cobra.Command, args []string) error {
	instanceID := args[0]

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()

	// Get instance details
	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Get debug information
	debugResult, err := dataaccess.DebugResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get debug information: %w", err)
	}

	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return  fmt.Errorf("failed to describe resource instance: %w", err)
	}

	// Process debug result and identify resource types
	data := DebugData{
		InstanceID: instanceID,
		Resources:  []ResourceInfo{},
	}

	// Use instanceData directly as a struct for BuildLogStreams and IsLogsEnabledStruct
	logsService := dataaccess.NewLogsService()
	IsLogsEnabled := logsService.IsLogsEnabled(instanceData)
	
	if debugResult.ResourcesDebug != nil {
		for resourceKey, resourceDebugInfo := range debugResult.ResourcesDebug {
			// Skip adding omnistrateobserv as a resource
			if resourceKey == "omnistrateobserv" {
				continue
			}
			
			// Process each resource based on its type
			resourceInfo := processResourceByType(resourceKey, resourceDebugInfo, instanceData, instanceID, IsLogsEnabled, logsService, ctx, token, serviceID, environmentID)
			if resourceInfo != nil {
				data.Resources = append(data.Resources, *resourceInfo)
			}
		}
	}

	// Launch TUI
	return launchDebugTUI(data)
}

// processResourceByType identifies the resource type and processes it accordingly
func processResourceByType(resourceKey string, resourceDebugInfo interface{}, instanceData *fleet.ResourceInstance, instanceID string, isLogsEnabled bool, logsService *dataaccess.LogsService, ctx context.Context, token, serviceID, environmentID string) *ResourceInfo {
	resourceInfo := &ResourceInfo{
		ID:        resourceKey,
		Name:      resourceKey,
		Type:      "unknown",
		DebugData: resourceDebugInfo,
	}

	debugData, ok := resourceDebugInfo.(map[string]interface{})
	if !ok {
		return processGenericResource(resourceInfo, instanceData, instanceID, isLogsEnabled, logsService, ctx, token, serviceID, environmentID)
	}

	actualDebugData, ok := debugData["debugData"].(map[string]interface{})
	if !ok {
		return processGenericResource(resourceInfo, instanceData, instanceID, isLogsEnabled, logsService, ctx, token, serviceID, environmentID)
	}

	// Check if it's a helm resource
	if _, hasChart := actualDebugData["chartRepoName"]; hasChart {
		return processHelmResource(resourceInfo, actualDebugData, instanceData, instanceID, isLogsEnabled, logsService, ctx, token, serviceID, environmentID)
	}

	// Check if it's a terraform resource
	if isTerraformResource(actualDebugData) {
		return processTerraformResource(resourceInfo, actualDebugData, ctx, token, serviceID, environmentID, instanceID)
	}

	// Default to generic resource
	return processGenericResource(resourceInfo, instanceData, instanceID, isLogsEnabled, logsService, ctx, token, serviceID, environmentID)
}

// processHelmResource handles Helm resource processing
func processHelmResource(resourceInfo *ResourceInfo, actualDebugData map[string]interface{}, instanceData *fleet.ResourceInstance, instanceID string, isLogsEnabled bool, logsService *dataaccess.LogsService, ctx context.Context, token, serviceID, environmentID string) *ResourceInfo {
	resourceInfo.Type = "helm"
	resourceInfo.HelmData = parseHelmData(actualDebugData)
	
	if isLogsEnabled {
		nodeData, err := logsService.BuildLogStreams(instanceData, instanceID, resourceInfo.ID)
		if err == nil && nodeData != nil {
			resourceInfo.HelmData.LiveLogs = nodeData
		}
	}
	
	// Fetch workflow events for this resource
	workflowEvents, err := dataaccess.GetDebugEventsForResource(ctx, token, serviceID, environmentID, instanceID, resourceInfo.ID)
	if err == nil && workflowEvents != nil {
		resourceInfo.WorkflowEvents = workflowEvents
	}
	
	return resourceInfo
}

// processTerraformResource handles Terraform resource processing
func processTerraformResource(resourceInfo *ResourceInfo, actualDebugData map[string]interface{}, ctx context.Context, token, serviceID, environmentID, instanceID string) *ResourceInfo {
	resourceInfo.Type = "terraform"
	resourceInfo.TerraformData = parseTerraformData(actualDebugData)
	
	// Fetch workflow events for this resource
	workflowEvents, err := dataaccess.GetDebugEventsForResource(ctx, token, serviceID, environmentID, instanceID, resourceInfo.ID)
	if err == nil && workflowEvents != nil {
		resourceInfo.WorkflowEvents = workflowEvents
	}
	
	return resourceInfo
}

// processGenericResource handles Generic resource processing
func processGenericResource(resourceInfo *ResourceInfo, instanceData *fleet.ResourceInstance, instanceID string, isLogsEnabled bool, logsService *dataaccess.LogsService, ctx context.Context, token, serviceID, environmentID string) *ResourceInfo {
	resourceInfo.Type = "generic"
	resourceInfo.GenericData = &GenericData{}
	
	if isLogsEnabled {
		nodeData, err := logsService.BuildLogStreams(instanceData, instanceID, resourceInfo.ID)
		if err == nil && nodeData != nil {
			resourceInfo.GenericData.LiveLogs = nodeData
		}
	}
	
	// Fetch workflow events for this resource
	workflowEvents, err := dataaccess.GetDebugEventsForResource(ctx, token, serviceID, environmentID, instanceID, resourceInfo.ID)
	if err == nil && workflowEvents != nil {
		resourceInfo.WorkflowEvents = workflowEvents
	}
	
	return resourceInfo
}

// isTerraformResource checks if the resource is a Terraform resource
func isTerraformResource(actualDebugData map[string]interface{}) bool {
	hasTerraformFiles := false
	hasTerraformLogs := false
	
	for key := range actualDebugData {
		if strings.HasPrefix(key, "rendered/") && strings.HasSuffix(key, ".tf") {
			hasTerraformFiles = true
		} else if strings.HasPrefix(key, "log/") && strings.Contains(key, "terraform") {
			hasTerraformLogs = true
		}
	}
	
	return hasTerraformFiles || hasTerraformLogs
}

func parseHelmData(debugData map[string]interface{}) *HelmData {
	helmData := &HelmData{
		ChartValues: make(map[string]interface{}),
	}

	if chartRepoName, ok := debugData["chartRepoName"].(string); ok {
		helmData.ChartRepoName = chartRepoName
	}
	if chartRepoURL, ok := debugData["chartRepoURL"].(string); ok {
		helmData.ChartRepoURL = chartRepoURL
	}
	if chartVersion, ok := debugData["chartVersion"].(string); ok {
		helmData.ChartVersion = chartVersion
	}
	if namespace, ok := debugData["namespace"].(string); ok {
		helmData.Namespace = namespace
	}
	if releaseName, ok := debugData["releaseName"].(string); ok {
		helmData.ReleaseName = releaseName
	}

	// Parse chart values
	if chartValuesStr, ok := debugData["chartValues"].(string); ok {
		var chartValues map[string]interface{}
		if err := json.Unmarshal([]byte(chartValuesStr), &chartValues); err == nil {
			helmData.ChartValues = chartValues
		}
	}

	// Parse install log
	if installLog, ok := debugData["log/install.log"].(string); ok {
		helmData.InstallLog = installLog
	}

	return helmData
}

func parseTerraformData(debugData map[string]interface{}) *TerraformData {
	terraformData := &TerraformData{
		Files: make(map[string]string),
		Logs:  make(map[string]string),
	}

	// Parse all files and logs
	for key, value := range debugData {
		if strValue, ok := value.(string); ok {
			if strings.HasPrefix(key, "rendered/") && strings.HasSuffix(key, ".tf") {
				terraformData.Files[key] = strValue
			} else if strings.HasPrefix(key, "log/") {
				terraformData.Logs[key] = strValue
			}
		}
	}

	return terraformData
}

func launchDebugTUI(data DebugData) error {
	app := tview.NewApplication()

	// Global state to track current selection and terraform data for file browser
	var currentTerraformData *TerraformData
	var currentSelectionIsTerraformFiles bool
	var currentSelectionIsTerraformLogs bool

	// Create main layout
	flex := tview.NewFlex()

	// Left panel - Resources (accordion style)
	leftPanel := tview.NewTreeView()
	leftPanel.SetBorder(true).SetTitle("Resources")

	// Create root node
	root := tview.NewTreeNode(fmt.Sprintf("Instance: %s", data.InstanceID))
	root.SetColor(tcell.ColorYellow)
	leftPanel.SetRoot(root)

	// Add resources based on their type
	for _, resource := range data.Resources {
		// Skip unknown resource types
		if resource.Type != "helm" && resource.Type != "terraform" && resource.Type != "generic" {
			continue
		}

		// Use separate functions for each resource type
		var resourceNode *tview.TreeNode
		switch resource.Type {
		case "helm":
			resourceNode = buildHelmResourceNode(resource)
		case "terraform":
			resourceNode = buildTerraformResourceNode(resource)
		case "generic":
			resourceNode = buildGenericResourceNode(resource)
		}

		if resourceNode != nil {
			root.AddChild(resourceNode)
		}
	}

	root.SetExpanded(true)

	// Right panel - Content
	rightPanel := tview.NewTextView()
	rightPanel.SetBorder(true).SetTitle("Content")
	rightPanel.SetDynamicColors(true)
	rightPanel.SetWrap(true)
	rightPanel.SetScrollable(true)
	rightPanel.SetText("Select a resource option to view details")

	// Add focus handlers to show which panel is active
	leftPanel.SetFocusFunc(func() {
		leftPanel.SetBorderColor(tcell.ColorGreen)
		rightPanel.SetBorderColor(tcell.ColorDefault)
	})
	rightPanel.SetFocusFunc(func() {
		rightPanel.SetBorderColor(tcell.ColorGreen)
		leftPanel.SetBorderColor(tcell.ColorDefault)
	})

	// Handle tree selection
	leftPanel.SetChangedFunc(func(node *tview.TreeNode) {
		handleTreeNodeSelection(node, rightPanel, app, &currentTerraformData, &currentSelectionIsTerraformFiles, &currentSelectionIsTerraformLogs)
	})

	// Also handle direct selection (Enter key)
	leftPanel.SetSelectedFunc(func(node *tview.TreeNode) {
		handleTreeNodeEnterSelection(node, rightPanel, &currentTerraformData, &currentSelectionIsTerraformFiles, &currentSelectionIsTerraformLogs)
	})

	// Set up layout
	flex.AddItem(leftPanel, 0, 1, true)
	flex.AddItem(rightPanel, 0, 2, false)

	// Create main layout with help text
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainFlex.AddItem(flex, 0, 1, true)
	mainFlex.AddItem(createHelpText(), 1, 0, false)

	// Create main input handler
	var mainInputHandler func(event *tcell.EventKey) *tcell.EventKey
	mainInputHandler = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyEnter:
			// Switch from left panel to right panel to view content
			if app.GetFocus() == leftPanel {
				app.SetFocus(rightPanel)
				return nil
			}
			// If on right panel, let default behavior handle scrolling
		case tcell.KeyEscape:
			// Go back to left panel from right panel
			if app.GetFocus() == rightPanel {
				app.SetFocus(leftPanel)
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q':
				app.Stop()
				return nil
			case 'f', 'F':
				if currentSelectionIsTerraformFiles && currentTerraformData != nil && len(currentTerraformData.Files) > 0 {
					showFileBrowser(app, currentTerraformData, mainFlex, mainInputHandler)
				}
				return nil
			case 'l', 'L':
				if currentSelectionIsTerraformLogs && currentTerraformData != nil && len(currentTerraformData.Logs) > 0 {
					showLogsBrowser(app, currentTerraformData, mainFlex, mainInputHandler)
				}
				return nil
			}
		}
		return event
	}

	// Set the main input handler
	app.SetInputCapture(mainInputHandler)

	// Set initial focus and selection
	app.SetFocus(leftPanel)

	// Set initial selection to first resource if available
	if len(data.Resources) > 0 {
		// Find the first resource node
		if len(root.GetChildren()) > 0 {
			firstResource := root.GetChildren()[0]
			leftPanel.SetCurrentNode(firstResource)
			// Expand the first resource to show its options
			firstResource.SetExpanded(true)
		}
	}

	// Start the application (disable mouse to allow terminal text selection)
	if err := app.SetRoot(mainFlex, true).EnableMouse(false).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// buildHelmResourceNode creates a tree node for Helm resources with their specific options
func buildHelmResourceNode(resource ResourceInfo) *tview.TreeNode {
	nodeLabel := fmt.Sprintf("%s (%s)", resource.Name, resource.Type)
	resourceNode := tview.NewTreeNode(nodeLabel)
	resourceNode.SetReference(resource)
	resourceNode.SetColor(tcell.ColorBlue)

	if resource.HelmData != nil {
		// Add Chart Values option
		chartValuesNode := tview.NewTreeNode("Chart Values")
		chartValuesNode.SetReference(map[string]interface{}{
			"type":     "helm-chart-values",
			"resource": resource,
		})
		chartValuesNode.SetColor(tcell.ColorGreen)
		resourceNode.AddChild(chartValuesNode)

		// Add Install Log option
		if resource.HelmData.InstallLog != "" {
			installLogNode := tview.NewTreeNode("Install Log")
			installLogNode.SetReference(map[string]interface{}{
				"type":     "helm-install-log",
				"resource": resource,
			})
			installLogNode.SetColor(tcell.ColorGreen)
			resourceNode.AddChild(installLogNode)
		}

		// Add Live Logs tree
		if len(resource.HelmData.LiveLogs) > 0 {
			liveLogsNode := tview.NewTreeNode("Live Log")
			liveLogsNode.SetColor(tcell.ColorGreen)
			for _, log := range resource.HelmData.LiveLogs {
				podNode := tview.NewTreeNode(log.PodName)
				podNode.SetReference(map[string]interface{}{
					"type":     "live-log-pod",
					"resource": resource,
					"podName":  log.PodName,
					"logsUrl":  log.LogsURL,
				})
				podNode.SetColor(tcell.ColorLightCyan)
				liveLogsNode.AddChild(podNode)
			}
			resourceNode.AddChild(liveLogsNode)
		}
	}

	// Add Debug Events node
	if debugEventsNode := buildDebugEventsNode(resource); debugEventsNode != nil {
		resourceNode.AddChild(debugEventsNode)
	}

	return resourceNode
}

// buildTerraformResourceNode creates a tree node for Terraform resources with their specific options
func buildTerraformResourceNode(resource ResourceInfo) *tview.TreeNode {
	nodeLabel := fmt.Sprintf("%s (%s)", resource.Name, resource.Type)
	resourceNode := tview.NewTreeNode(nodeLabel)
	resourceNode.SetReference(resource)
	resourceNode.SetColor(tcell.ColorBlue)

	if resource.TerraformData != nil {
		// Add Terraform Files option
		if len(resource.TerraformData.Files) > 0 {
			filesNode := tview.NewTreeNode("Terraform Files")
			filesNode.SetReference(map[string]interface{}{
				"type":     "terraform-files",
				"resource": resource,
			})
			filesNode.SetColor(tcell.ColorGreen)
			resourceNode.AddChild(filesNode)
		}

		// Add Install Log option
		if len(resource.TerraformData.Logs) > 0 {
			installLogNode := tview.NewTreeNode("Install Log")
			installLogNode.SetReference(map[string]interface{}{
				"type":     "terraform-install-logs",
				"resource": resource,
			})
			installLogNode.SetColor(tcell.ColorGreen)
			resourceNode.AddChild(installLogNode)
		}
	}

	// Add Debug Events node
	if debugEventsNode := buildDebugEventsNode(resource); debugEventsNode != nil {
		resourceNode.AddChild(debugEventsNode)
	}

	return resourceNode
}

// buildGenericResourceNode creates a tree node for Generic resources with their specific options
func buildGenericResourceNode(resource ResourceInfo) *tview.TreeNode {
	// Generic resources show only their name (no type suffix)
	nodeLabel := resource.Name
	resourceNode := tview.NewTreeNode(nodeLabel)
	resourceNode.SetReference(resource)
	resourceNode.SetColor(tcell.ColorBlue)

	if resource.GenericData != nil {
		// Add Live Logs tree
		if len(resource.GenericData.LiveLogs) > 0 {
			liveLogsNode := tview.NewTreeNode("Live Log")
			liveLogsNode.SetColor(tcell.ColorGreen)
			for _, log := range resource.GenericData.LiveLogs {
				podNode := tview.NewTreeNode(log.PodName)
				podNode.SetReference(map[string]interface{}{
					"type":     "live-log-pod",
					"resource": resource,
					"podName":  log.PodName,
					"logsUrl":  log.LogsURL,
				})
				podNode.SetColor(tcell.ColorLightCyan)
				liveLogsNode.AddChild(podNode)
			}
			resourceNode.AddChild(liveLogsNode)
		}
	}

	// Add Debug Events node
	if debugEventsNode := buildDebugEventsNode(resource); debugEventsNode != nil {
		resourceNode.AddChild(debugEventsNode)
	}

	return resourceNode
}

// buildDebugEventsNode creates a tree node for workflow events with categories
func buildDebugEventsNode(resource ResourceInfo) *tview.TreeNode {
	if resource.WorkflowEvents == nil {
		return nil
	}

	debugEventsNode := tview.NewTreeNode("Debug Events")
	debugEventsNode.SetColor(tcell.ColorYellow)
	
	hasEvents := false
	
	// Add category nodes
	if len(resource.WorkflowEvents.Bootstrap) > 0 {
		bootstrapNode := tview.NewTreeNode(fmt.Sprintf("Bootstrap (%d)", len(resource.WorkflowEvents.Bootstrap)))
		bootstrapNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Bootstrap",
			"events":   resource.WorkflowEvents.Bootstrap,
		})
		bootstrapNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(bootstrapNode)
		hasEvents = true
	}
	
	if len(resource.WorkflowEvents.Storage) > 0 {
		storageNode := tview.NewTreeNode(fmt.Sprintf("Storage (%d)", len(resource.WorkflowEvents.Storage)))
		storageNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Storage",
			"events":   resource.WorkflowEvents.Storage,
		})
		storageNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(storageNode)
		hasEvents = true
	}
	
	if len(resource.WorkflowEvents.Network) > 0 {
		networkNode := tview.NewTreeNode(fmt.Sprintf("Network (%d)", len(resource.WorkflowEvents.Network)))
		networkNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Network",
			"events":   resource.WorkflowEvents.Network,
		})
		networkNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(networkNode)
		hasEvents = true
	}

	if len(resource.WorkflowEvents.Compute) > 0 {
		computeNode := tview.NewTreeNode(fmt.Sprintf("Compute (%d)", len(resource.WorkflowEvents.Compute)))
		computeNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Compute",
			"events":   resource.WorkflowEvents.Compute,
		})
		computeNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(computeNode)
		hasEvents = true
	}

	
	if len(resource.WorkflowEvents.Deployment) > 0 {
		deploymentNode := tview.NewTreeNode(fmt.Sprintf("Deployment (%d)", len(resource.WorkflowEvents.Deployment)))
		deploymentNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Deployment",
			"events":   resource.WorkflowEvents.Deployment,
		})
		deploymentNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(deploymentNode)
		hasEvents = true
	}

		if len(resource.WorkflowEvents.Monitoring) > 0 {
		monitoringNode := tview.NewTreeNode(fmt.Sprintf("Monitoring (%d)", len(resource.WorkflowEvents.Monitoring)))
		monitoringNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Monitoring",
			"events":   resource.WorkflowEvents.Monitoring,
		})
		monitoringNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(monitoringNode)
		hasEvents = true
	}
	
	
	if len(resource.WorkflowEvents.Other) > 0 {
		otherNode := tview.NewTreeNode(fmt.Sprintf("Other (%d)", len(resource.WorkflowEvents.Other)))
		otherNode.SetReference(map[string]interface{}{
			"type":     "debug-events-category",
			"resource": resource,
			"category": "Other",
			"events":   resource.WorkflowEvents.Other,
		})
		otherNode.SetColor(tcell.ColorWhite)
		debugEventsNode.AddChild(otherNode)
		hasEvents = true
	}
	
	// If no events to display, show an informational node
	if !hasEvents {
		noEventsNode := tview.NewTreeNode("No events available")
		noEventsNode.SetColor(tcell.ColorGray)
		debugEventsNode.AddChild(noEventsNode)
	}
	
	return debugEventsNode
}

// handleTreeNodeSelection processes tree node selection (when node changes/is highlighted)
func handleTreeNodeSelection(node *tview.TreeNode, rightPanel *tview.TextView, app *tview.Application, currentTerraformData **TerraformData, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	reference := node.GetReference()
	if reference == nil {
		handleNonReferencedNodeSelection(node, rightPanel, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
		return
	}

	switch ref := reference.(type) {
	case ResourceInfo:
		handleResourceInfoSelection(ref, rightPanel, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
	case map[string]interface{}:
		handleOptionMapSelection(ref, rightPanel, app, currentTerraformData, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
	}
}

// handleTreeNodeEnterSelection processes tree node selection (when Enter key is pressed)
func handleTreeNodeEnterSelection(node *tview.TreeNode, rightPanel *tview.TextView, currentTerraformData **TerraformData, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	reference := node.GetReference()
	if reference != nil {
		// If it's an option, show its content
		switch ref := reference.(type) {
		case ResourceInfo:
			handleResourceInfoSelection(ref, rightPanel, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
		case map[string]interface{}:
			handleOptionMapSelectionForEnter(ref, rightPanel, currentTerraformData, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
			return // Don't toggle expansion for options
		}
	}
	// Toggle expansion for resource nodes
	node.SetExpanded(!node.IsExpanded())
}

// handleNonReferencedNodeSelection handles selection of nodes without references (like "Live Log" header nodes)
func handleNonReferencedNodeSelection(node *tview.TreeNode, rightPanel *tview.TextView, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	// If the currently selected node is a Live Logs node, show pod list
	if node.GetText() == "Live Log" {
		rightPanel.SetTitle("Live Log")
		// Find pod children and list their names
		podNames := []string{}
		for _, child := range node.GetChildren() {
			podNames = append(podNames, child.GetText())
		}
		if len(podNames) > 0 {
			rightPanel.SetText("[yellow]Live Log[white]\n\n Nodes:\n  " + strings.Join(podNames, "\n  ") + "\n\nSelect a node option to view details")
		} else {
			rightPanel.SetText("[yellow]Live Log[white]\n No pods available")
		}
	} else {
		rightPanel.SetTitle("Content")
		rightPanel.SetText("Select a resource option to view details")
	}
	// Clear terraform selection state when no valid selection
	*currentSelectionIsTerraformFiles = false
	*currentSelectionIsTerraformLogs = false
}

// handleResourceInfoSelection handles selection of ResourceInfo nodes
func handleResourceInfoSelection(resource ResourceInfo, rightPanel *tview.TextView, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	// Show resource information
	content := formatResourceInfo(resource)
	rightPanel.SetTitle(fmt.Sprintf("Resource: %s", resource.Name))
	rightPanel.SetText(content)
	// Clear terraform selection state when selecting resource node
	*currentSelectionIsTerraformFiles = false
	*currentSelectionIsTerraformLogs = false
}

// handleOptionMapSelection handles selection of option map nodes (for tree selection changes)
func handleOptionMapSelection(ref map[string]interface{}, rightPanel *tview.TextView, app *tview.Application, currentTerraformData **TerraformData, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	if t, ok := ref["type"].(string); ok && t == "live-log-pod" {
		handleLiveLogPodSelection(ref, rightPanel, app)
	} else if t, ok := ref["type"].(string); ok && t == "debug-events-category" {
		handleDebugEventsCategorySelection(ref, rightPanel)
	} else {
		handleOptionSelection(ref, rightPanel)
		updateTerraformSelectionState(ref, currentTerraformData, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
	}
}

// handleOptionMapSelectionForEnter handles selection of option map nodes (for Enter key presses)
func handleOptionMapSelectionForEnter(ref map[string]interface{}, rightPanel *tview.TextView, currentTerraformData **TerraformData, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	if t, ok := ref["type"].(string); ok && t == "debug-events-category" {
		handleDebugEventsCategorySelection(ref, rightPanel)
	} else {
		handleOptionSelection(ref, rightPanel)
		updateTerraformSelectionState(ref, currentTerraformData, currentSelectionIsTerraformFiles, currentSelectionIsTerraformLogs)
	}
}

// handleLiveLogPodSelection handles selection of live log pod nodes
func handleLiveLogPodSelection(ref map[string]interface{}, rightPanel *tview.TextView, app *tview.Application) {
	// Open pod log view (websocket connect)
	podName, _ := ref["podName"].(string)
	logsUrl, _ := ref["logsUrl"].(string)
	rightPanel.SetTitle(fmt.Sprintf("Live Log: %s", podName))
	rightPanel.SetText(fmt.Sprintf("Connecting to logs for %s...", podName))
	go connectAndStreamLogs(app, logsUrl, rightPanel)
}

// getMessageTypeColor returns the appropriate color based on messageType and eventType
func getMessageTypeColor(messageType, eventType string) string {
	if messageType == "" {
		// Check if it's a failed event type
		failedEventTypes := []string{"WorkflowStepFailed", "WorkflowFailed", "Error"}
		for _, failedType := range failedEventTypes {
			if eventType == failedType {
				return "red"
			}
		}
		return "green"
	}

	switch messageType {
	case "Completed":
		return "green"
	case "Failed", "Cancelled", "TimedOut", "Terminated":
		return "red"
	default:
		return "yellow"
	}
}

// handleDebugEventsCategorySelection handles selection of debug events category nodes
func handleDebugEventsCategorySelection(ref map[string]interface{}, rightPanel *tview.TextView) {
	category, _ := ref["category"].(string)
	events, _ := ref["events"].([]dataaccess.CustomWorkflowEvent)
	resource, _ := ref["resource"].(ResourceInfo)
	
	rightPanel.SetTitle(fmt.Sprintf("Debug Events: %s - %s", resource.Name, category))
	
	var content strings.Builder
	content.WriteString(fmt.Sprintf("[yellow]=== %s Events for %s ===[white]\n\n", category, resource.Name))
	
	if len(events) == 0 {
		content.WriteString("[gray]No events found in this category.[white]\n")
	} else {
		for i, event := range events {
			// Determine event type color
			var eventTypeColor string
			switch event.EventType {
			case "WorkflowStepStarted":
				eventTypeColor = "blue"
			case "WorkflowStepCompleted":
				eventTypeColor = "green"
			case "WorkflowStepFailed", "WorkflowFailed":
				eventTypeColor = "red"
			case "WorkflowStepDebug":
				eventTypeColor = "yellow"
			default:
				eventTypeColor = "white"
			}
			
			content.WriteString(fmt.Sprintf("[orange]Event %d:[white]\n", i+1))
			content.WriteString(fmt.Sprintf("  [lightcyan]Time:[white] %s\n", event.EventTime))
			content.WriteString(fmt.Sprintf("  [lightcyan]Type:[white] [%s]%s[white]\n", eventTypeColor, event.EventType))
			
			// Try to parse and format the message
			if strings.HasPrefix(event.Message, "{") && strings.HasSuffix(event.Message, "}") {
				// It's JSON, try to format it nicely
				var messageData map[string]interface{}
				if err := json.Unmarshal([]byte(event.Message), &messageData); err == nil {
					content.WriteString("  [lightcyan]Details:[white]\n")
					for key, value := range messageData {
						// Apply color coding based on actionStatus
						if key == "actionStatus" {
							actionStatus := fmt.Sprintf("%v", value)
							color := getMessageTypeColor(actionStatus, event.EventType)
							content.WriteString(fmt.Sprintf("    [lightcyan]%s:[white] [%s]%v[white]\n", key, color, value))
						} else if key == "message" {
							// Color code messages based on content
							messageText := fmt.Sprintf("%v", value)
							var color string
							if strings.Contains(strings.ToLower(messageText), "completed") || 
							   strings.Contains(strings.ToLower(messageText), "success") {
								color = "green"
							} else if strings.Contains(strings.ToLower(messageText), "failed") || 
							         strings.Contains(strings.ToLower(messageText), "error") {
								color = "red"
							} else {
								color = "white"
							}
							content.WriteString(fmt.Sprintf("    [lightcyan]%s:[white] [%s]%v[white]\n", key, color, value))
						} else {
							content.WriteString(fmt.Sprintf("    [lightcyan]%s:[white] %v\n", key, value))
						}
					}
				} else {
					content.WriteString(fmt.Sprintf("  [lightcyan]Message:[white] %s\n", event.Message))
				}
			} else {
				content.WriteString(fmt.Sprintf("  [lightcyan]Message:[white] %s\n", event.Message))
			}
			
			content.WriteString("\n")
		}
	}
	
	rightPanel.SetText(content.String())
}

// updateTerraformSelectionState updates terraform selection state based on option type
func updateTerraformSelectionState(ref map[string]interface{}, currentTerraformData **TerraformData, currentSelectionIsTerraformFiles *bool, currentSelectionIsTerraformLogs *bool) {
	// Update current terraform data and selection state for file browser
	if optionType, ok := ref["type"].(string); ok {
		switch optionType {
		case "terraform-files":
			if resource, ok := ref["resource"].(ResourceInfo); ok {
				*currentTerraformData = resource.TerraformData
				*currentSelectionIsTerraformFiles = true
				*currentSelectionIsTerraformLogs = false
			}
		case "terraform-install-logs":
			if resource, ok := ref["resource"].(ResourceInfo); ok {
				*currentTerraformData = resource.TerraformData
				*currentSelectionIsTerraformFiles = false
				*currentSelectionIsTerraformLogs = true
			}
		default:
			*currentSelectionIsTerraformFiles = false
			*currentSelectionIsTerraformLogs = false
		}
	} else {
		*currentSelectionIsTerraformFiles = false
		*currentSelectionIsTerraformLogs = false
	}
}

func createHelpText() *tview.TextView {
	helpText := tview.NewTextView()
	helpText.SetText("Navigate: ↑/↓ to move | Enter: view content/expand | Esc: go back | f: file browser | l: logs browser | q: quit")
	helpText.SetTextAlign(tview.AlignCenter)
	helpText.SetDynamicColors(true)
	return helpText
}

func handleOptionSelection(ref map[string]interface{}, rightPanel *tview.TextView) {
	optionType, _ := ref["type"].(string)
	resource, _ := ref["resource"].(ResourceInfo)

	switch optionType {
	case "helm-chart-values":
		if resource.HelmData != nil {
			content := formatHelmChartValues(resource.HelmData)
			rightPanel.SetTitle("Chart Values")
			rightPanel.SetText(content)
		}
	case "helm-install-log":
		if resource.HelmData != nil {
			content := formatHelmInstallLog(resource.HelmData.InstallLog)
			rightPanel.SetTitle("Install Log")
			rightPanel.SetText(content)
		}
	case "helm-live-log":
		if resource.HelmData != nil {
			content := formatLiveLogs(resource.HelmData.LiveLogs)
			rightPanel.SetTitle("Live Log")
			rightPanel.SetText(content)
		}
	case "terraform-files":
		if resource.TerraformData != nil {
			content := formatTerraformFileList(resource.TerraformData.Files)
			rightPanel.SetTitle("Terraform Files")
			rightPanel.SetText(content)
		}
	case "terraform-install-logs":
		if resource.TerraformData != nil {
			content := formatTerraformLogsHierarchical(resource.TerraformData.Logs)
			rightPanel.SetTitle("Install Logs")
			rightPanel.SetText(content)
		}
	
	case "generic-live-logs":
		if resource.GenericData != nil {
			content := formatLiveLogs(resource.GenericData.LiveLogs)
			rightPanel.SetTitle("Live Logs")
			rightPanel.SetText(content)
		}
	}
}

func formatResourceInfo(resource ResourceInfo) string {
	debugInfo := ""
	if resource.Type == "terraform" && resource.TerraformData != nil {
		debugInfo = fmt.Sprintf("\n\nTerraform Files: %d\nTerraform Logs: %d", len(resource.TerraformData.Files), len(resource.TerraformData.Logs))
	} else if resource.Type == "helm" && resource.HelmData != nil {
		debugInfo = fmt.Sprintf("\n\nChart: %s\nInstall Log: %t", resource.HelmData.ChartRepoName, resource.HelmData.InstallLog != "")
	}

	return fmt.Sprintf(`[yellow]Resource Information[white]

Name: %s
Type: %s
ID: %s%s

Select an option from the tree to view specific details.`, resource.Name, resource.Type, resource.ID, debugInfo)
}

func formatHelmChartValues(helmData *HelmData) string {
	content := fmt.Sprintf(`[yellow]Helm Chart Values[white]

Chart: %s
Version: %s
Repo: %s
Namespace: %s
Release: %s

[yellow]Values:[white]
`, helmData.ChartRepoName, helmData.ChartVersion, helmData.ChartRepoURL, helmData.Namespace, helmData.ReleaseName)

	if len(helmData.ChartValues) > 0 {
		jsonBytes, err := json.MarshalIndent(helmData.ChartValues, "", "  ")
		if err == nil {
			// Apply YAML syntax highlighting to the JSON output (similar structure)
			highlightedContent := addYAMLSyntaxHighlighting(string(jsonBytes))
			content += highlightedContent
		} else {
			content += fmt.Sprintf("Error formatting values: %v", err)
		}
	} else {
		content += "No chart values available"
	}

	return content
}

func formatHelmInstallLog(installLog string) string {
	if installLog == "" {
		return "[yellow]Install Log[white]\n\nNo install log available"
	}
	// Apply log syntax highlighting
	highlightedLog := addLogSyntaxHighlighting(installLog)
	return fmt.Sprintf(`[yellow]Install Log[white]

%s`, highlightedLog)
}

func formatTerraformFileList(files map[string]string) string {
	if len(files) == 0 {
		return "[yellow]Terraform Files[white]\n\nNo terraform files available"
	}

	content := "[yellow]Terraform Files[white]\n\nFiles available (press 'f' to open file browser):\n\n"

	// Build a hierarchical tree structure
	type TreeNode struct {
		Name     string
		IsDir    bool
		Children map[string]*TreeNode
		Files    []string
	}

	root := &TreeNode{
		Name:     "root",
		IsDir:    true,
		Children: make(map[string]*TreeNode),
		Files:    []string{},
	}

	// Get sorted file paths for deterministic ordering
	filePaths := make([]string, 0, len(files))
	for filePath := range files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	// Build the tree structure
	for _, filePath := range filePaths {
		parts := strings.Split(filePath, "/")
		currentNode := root

		// Navigate through directory parts
		for i, part := range parts {
			if i == len(parts)-1 {
				// This is a file
				currentNode.Files = append(currentNode.Files, part)
			} else {
				// This is a directory
				if currentNode.Children[part] == nil {
					currentNode.Children[part] = &TreeNode{
						Name:     part,
						IsDir:    true,
						Children: make(map[string]*TreeNode),
						Files:    []string{},
					}
				}
				currentNode = currentNode.Children[part]
			}
		}
	}

	// Function to render the tree
	var renderTree func(node *TreeNode, prefix string, isLast bool) string
	renderTree = func(node *TreeNode, prefix string, isLast bool) string {
		result := ""

		// Sort children directories and files
		var childNames []string
		for name := range node.Children {
			childNames = append(childNames, name)
		}
		sort.Strings(childNames)
		sort.Strings(node.Files)

		// Render child directories
		for i, childName := range childNames {
			child := node.Children[childName]
			isLastChild := (i == len(childNames)-1) && len(node.Files) == 0

			// Choose the right tree symbol
			var symbol, nextPrefix string
			if isLastChild {
				symbol = "└── "
				nextPrefix = prefix + "    "
			} else {
				symbol = "├── "
				nextPrefix = prefix + "│   "
			}

			result += fmt.Sprintf("%s[blue]%s%s/[-]\n", prefix, symbol, childName)
			result += renderTree(child, nextPrefix, true)
		}

		// Render files
		for i, fileName := range node.Files {
			isLastFile := i == len(node.Files)-1
			var symbol string
			if isLastFile {
				symbol = "└── "
			} else {
				symbol = "├── "
			}
			result += fmt.Sprintf("%s%s%s\n", prefix, symbol, fileName)
		}

		return result
	}

	// Render the tree starting from root
	content += renderTree(root, "", true)
	content += "\n[green]Press 'f' to open file browser and view individual files[-]"

	return content
}


func formatTerraformLogsHierarchical(logs map[string]string) string {
	if len(logs) == 0 {
		return "[yellow]Terraform Logs[white]\n\nNo terraform logs available"
	}

	content := "[yellow]Terraform Logs[white]\n\nLogs available (press 'l' to open logs browser):\n\n"

	// Build a hierarchical tree structure for logs
	type LogTreeNode struct {
		Name     string
		IsPhase  bool
		Children map[string]*LogTreeNode
		Logs     []string
	}

	root := &LogTreeNode{
		Name:     "root",
		IsPhase:  true,
		Children: make(map[string]*LogTreeNode),
		Logs:     []string{},
	}

	// Get sorted log paths for deterministic ordering
	logPaths := make([]string, 0, len(logs))
	for logPath := range logs {
		logPaths = append(logPaths, logPath)
	}
	sort.Strings(logPaths)

	// Parse logs into hierarchical structure
	// Pattern: log/[previous_]<stream>_terraform_<phase>.log
	// Example: log/stdout_terraform_init.log, log/previous_stderr_terraform_apply.log
	for _, logPath := range logPaths {
		if !strings.HasPrefix(logPath, "log/") {
			continue
		}

		// Extract log filename without log/ prefix
		logName := strings.TrimPrefix(logPath, "log/")
		
		// Parse the log name to extract phase and stream info
		// Pattern: [previous_]<stream>_terraform_<phase>.log
		phase := "unknown"
		stream := "unknown"
		isPrevious := false

		if strings.HasPrefix(logName, "previous_") {
			isPrevious = true
			logName = strings.TrimPrefix(logName, "previous_")
		}

		// Parse stream_terraform_phase.log
		parts := strings.Split(logName, "_")
		if len(parts) >= 3 && parts[1] == "terraform" {
			stream = parts[0] // stdout or stderr
			phasePart := strings.Join(parts[2:], "_")
			phase = strings.TrimSuffix(phasePart, ".log")
		}

		// Create phase node (init, apply, destroy, etc.)
		phaseKey := phase
		if isPrevious {
			phaseKey = "previous_" + phase
		}

		if root.Children[phaseKey] == nil {
			var displayName string
			if isPrevious {
				displayName = "Previous " + strings.ToTitle(phase[:1]) + phase[1:]
			} else {
				displayName = strings.ToTitle(phase[:1]) + phase[1:]
			}
			root.Children[phaseKey] = &LogTreeNode{
				Name:     displayName,
				IsPhase:  true,
				Children: make(map[string]*LogTreeNode),
				Logs:     []string{},
			}
		}

		// Add stream (stdout/stderr) under the phase
		phaseNode := root.Children[phaseKey]
		if phaseNode.Children[stream] == nil {
			streamDisplayName := strings.ToUpper(stream)
			phaseNode.Children[stream] = &LogTreeNode{
				Name:     streamDisplayName,
				IsPhase:  false,
				Children: make(map[string]*LogTreeNode),
				Logs:     []string{},
			}
		}

		// Add the actual log file
		streamNode := phaseNode.Children[stream]
		streamNode.Logs = append(streamNode.Logs, logPath)
	}

	// Function to render the tree
	var renderLogTree func(node *LogTreeNode, prefix string, isLast bool) string
	renderLogTree = func(node *LogTreeNode, prefix string, isLast bool) string {
		result := ""

		// Sort children (phases and streams)
		var childNames []string
		for name := range node.Children {
			childNames = append(childNames, name)
		}
		
		// Sort phases in logical order: init, apply, destroy, then previous runs
		phaseOrder := map[string]int{
			"init":             1,
			"plan":             2,
			"apply":            3,
			"destroy":          4,
			"previous_init":    5,
			"previous_plan":    6,
			"previous_apply":   7,
			"previous_destroy": 8,
		}
		
		sort.Slice(childNames, func(i, j int) bool {
			orderI, hasI := phaseOrder[childNames[i]]
			orderJ, hasJ := phaseOrder[childNames[j]]
			
			if hasI && hasJ {
				return orderI < orderJ
			} else if hasI {
				return true
			} else if hasJ {
				return false
			}
			return childNames[i] < childNames[j]
		})
		
		sort.Strings(node.Logs)

		// Render child phases/streams
		for i, childName := range childNames {
			child := node.Children[childName]
			isLastChild := (i == len(childNames)-1) && len(node.Logs) == 0

			// Choose the right tree symbol
			var symbol, nextPrefix string
			if isLastChild {
				symbol = "└── "
				nextPrefix = prefix + "    "
			} else {
				symbol = "├── "
				nextPrefix = prefix + "│   "
			}

			if child.IsPhase {
				result += fmt.Sprintf("%s[blue]%s%s/[-]\n", prefix, symbol, child.Name)
			} else {
				result += fmt.Sprintf("%s[lightblue]%s%s[-]\n", prefix, symbol, child.Name)
			}
			result += renderLogTree(child, nextPrefix, true)
		}

		// Render log files
		for i, logPath := range node.Logs {
			isLastLog := i == len(node.Logs)-1
			var symbol string
			if isLastLog {
				symbol = "└── "
			} else {
				symbol = "├── "
			}
			
			// Extract just the filename for display
			logName := filepath.Base(logPath)
			
			// Color code based on content or status
			logContent := logs[logPath]
			if strings.Contains(strings.ToLower(logContent), "error") || strings.Contains(strings.ToLower(logContent), "failed") {
				result += fmt.Sprintf("%s[red]%s%s[-]\n", prefix, symbol, logName)
			} else if strings.Contains(strings.ToLower(logContent), "warn") {
				result += fmt.Sprintf("%s[yellow]%s%s[-]\n", prefix, symbol, logName)
			} else if logContent != "" {
				result += fmt.Sprintf("%s[green]%s%s[-]\n", prefix, symbol, logName)
			} else {
				result += fmt.Sprintf("%s[gray]%s%s (empty)[-]\n", prefix, symbol, logName)
			}
		}

		return result
	}

	// Render the tree starting from root
	content += renderLogTree(root, "", true)
	content += "\n[green]Press 'l' to open logs browser and view individual log contents[-]"

	return content
}

// formatLiveLogs formats the live logs for display in the TUI.
func formatLiveLogs(liveLogs []dataaccess.LogsStream) string {
	if len(liveLogs) == 0 {
		return "[yellow]Live Logs[white]\n\nNo live logs available"
	}
	var sb strings.Builder
	sb.WriteString("[yellow]Live Logs[white]\n\n")
	for _, log := range liveLogs {
		sb.WriteString(fmt.Sprintf("Pod: [cyan]%s[white]\nURL: [blue]%s[white]\n\n", log.PodName, log.LogsURL))
	}
	return sb.String()
}

// addYAMLSyntaxHighlighting adds basic syntax highlighting for YAML content
func addYAMLSyntaxHighlighting(content string) string {
	lines := strings.Split(content, "\n")
	var highlighted []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			highlighted = append(highlighted, line)
			continue
		}

		// Comments
		if strings.HasPrefix(trimmed, "#") {
			highlighted = append(highlighted, fmt.Sprintf("[green]%s[-]", line))
			continue
		}

		// Keys (lines containing ':')
		if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				highlighted = append(highlighted, fmt.Sprintf("[cyan]%s[-]:[yellow]%s[-]", key, value))
				continue
			}
		}

		// List items
		if strings.HasPrefix(trimmed, "-") {
			highlighted = append(highlighted, fmt.Sprintf("[blue]%s[-]", line))
			continue
		}

		highlighted = append(highlighted, line)
	}

	return strings.Join(highlighted, "\n")
}

// addTerraformSyntaxHighlighting adds basic syntax highlighting for Terraform files
func addTerraformSyntaxHighlighting(content string) string {
	lines := strings.Split(content, "\n")
	var highlighted []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			highlighted = append(highlighted, line)
			continue
		}

		// Comments
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			highlighted = append(highlighted, fmt.Sprintf("[green]%s[-]", line))
			continue
		}

		// Resource/data/variable/output blocks
		if strings.HasPrefix(trimmed, "resource ") || strings.HasPrefix(trimmed, "data ") ||
			strings.HasPrefix(trimmed, "variable ") || strings.HasPrefix(trimmed, "output ") ||
			strings.HasPrefix(trimmed, "provider ") || strings.HasPrefix(trimmed, "module ") {
			highlighted = append(highlighted, fmt.Sprintf("[fuchsia]%s[-]", line))
			continue
		}

		// String assignments (key = "value")
		if strings.Contains(line, "=") && strings.Contains(line, "\"") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := strings.TrimSpace(parts[1])
				// Highlight strings in quotes
				if strings.Contains(value, "\"") {
					value = strings.ReplaceAll(value, "\"", "[yellow]\"[-]")
				}
				highlighted = append(highlighted, fmt.Sprintf("[cyan]%s[-]= %s", key, value))
				continue
			}
		}

		// Simple assignments
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				highlighted = append(highlighted, fmt.Sprintf("[cyan]%s[-]=[blue]%s[-]", key, value))
				continue
			}
		}

		highlighted = append(highlighted, line)
	}

	return strings.Join(highlighted, "\n")
}

// addLogSyntaxHighlighting adds basic syntax highlighting for log content
func addLogSyntaxHighlighting(content string) string {
	lines := strings.Split(content, "\n")
	var highlighted []string

	for _, line := range lines {
		lower := strings.ToLower(line)

		// Error messages
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
			strings.Contains(lower, "panic") || strings.Contains(lower, "fatal") {
			highlighted = append(highlighted, fmt.Sprintf("[red]%s[-]", line))
			continue
		}

		// Warning messages
		if strings.Contains(lower, "warn") || strings.Contains(lower, "warning") {
			highlighted = append(highlighted, fmt.Sprintf("[yellow]%s[-]", line))
			continue
		}

		// Success messages
		if strings.Contains(lower, "success") || strings.Contains(lower, "complete") ||
			strings.Contains(lower, "applied") || strings.Contains(lower, "created") {
			highlighted = append(highlighted, fmt.Sprintf("[green]%s[-]", line))
			continue
		}

		// Info messages
		if strings.Contains(lower, "info") || strings.Contains(lower, "applying") ||
			strings.Contains(lower, "planning") || strings.Contains(lower, "refreshing") {
			highlighted = append(highlighted, fmt.Sprintf("[blue]%s[-]", line))
			continue
		}

		// Timestamps (basic detection)
		if strings.Contains(line, ":") && (strings.Contains(line, "T") ||
			strings.Contains(line, "[") && strings.Contains(line, "]")) {
			highlighted = append(highlighted, fmt.Sprintf("[gray]%s[-]", line))
			continue
		}

		highlighted = append(highlighted, line)
	}

	return strings.Join(highlighted, "\n")
}

func showFileBrowser(app *tview.Application, terraformData *TerraformData, mainFlex *tview.Flex, originalInputHandler func(event *tcell.EventKey) *tcell.EventKey) {
	// Create file tree view (hierarchical)
	fileTree := tview.NewTreeView()
	fileTree.SetBorder(true).SetTitle("Terraform Files")

	// Create root node
	root := tview.NewTreeNode("Files")
	root.SetColor(tcell.ColorYellow)
	fileTree.SetRoot(root)

	// Build hierarchical file structure
	dirNodes := make(map[string]*tview.TreeNode)

	// Get sorted file paths for deterministic ordering
	filePaths := make([]string, 0, len(terraformData.Files))
	for filePath := range terraformData.Files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	// Helper function to get or create directory node
	var getOrCreateDirNode func(path string) *tview.TreeNode
	getOrCreateDirNode = func(path string) *tview.TreeNode {
		if path == "." || path == "" {
			return root
		}

		// Check if we already have this directory
		if node, exists := dirNodes[path]; exists {
			return node
		}

		// Create the directory node
		dirName := filepath.Base(path)
		dirNode := tview.NewTreeNode(dirName + "/")
		dirNode.SetColor(tcell.ColorBlue)
		dirNode.SetExpanded(false) // Allow user to expand/collapse
		dirNodes[path] = dirNode

		// Get parent directory and add this node to it
		parentPath := filepath.Dir(path)
		parentNode := getOrCreateDirNode(parentPath)
		parentNode.AddChild(dirNode)

		return dirNode
	}

	// Build the tree structure
	for _, filePath := range filePaths {
		dir := filepath.Dir(filePath)
		fileName := filepath.Base(filePath)

		// Get the parent directory node (creates all intermediate directories)
		parentNode := getOrCreateDirNode(dir)

		// Add file to parent directory
		fileNode := tview.NewTreeNode(fileName)
		fileNode.SetReference(filePath)
		fileNode.SetColor(tcell.ColorWhite)
		parentNode.AddChild(fileNode)
	}

	root.SetExpanded(true)

	// Create file content viewer
	fileViewer := tview.NewTextView()
	fileViewer.SetBorder(true).SetTitle("File Content")
	fileViewer.SetScrollable(true)
	fileViewer.SetWrap(false)
	fileViewer.SetDynamicColors(true) // Enable color rendering
	fileViewer.SetText("Select a file from the tree to view its content")

	// Handle tree selection
	fileTree.SetChangedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference != nil {
			if filePath, ok := reference.(string); ok {
				if content, exists := terraformData.Files[filePath]; exists {
					fileViewer.SetTitle(fmt.Sprintf("File: %s", filePath))
					// Apply syntax highlighting based on file extension
					if strings.HasSuffix(filePath, ".tf") || strings.HasSuffix(filePath, ".tfvars") {
						highlightedContent := addTerraformSyntaxHighlighting(content)
						fileViewer.SetText(highlightedContent)
					} else {
						fileViewer.SetText(content)
					}
				}
			}
		}
	})

	// Handle tree node selection (Enter key)
	fileTree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference != nil {
			// If it's a file, show content and don't toggle expansion
			if filePath, ok := reference.(string); ok {
				if content, exists := terraformData.Files[filePath]; exists {
					fileViewer.SetTitle(fmt.Sprintf("File: %s", filePath))
					// Apply syntax highlighting based on file extension
					if strings.HasSuffix(filePath, ".tf") || strings.HasSuffix(filePath, ".tfvars") {
						highlightedContent := addTerraformSyntaxHighlighting(content)
						fileViewer.SetText(highlightedContent)
					} else {
						fileViewer.SetText(content)
					}
				}
				return // Don't toggle expansion for files
			}
		}
		// Toggle expansion for directory nodes (including root and subdirectories)
		node.SetExpanded(!node.IsExpanded())
	})

	// Add focus handlers to show which panel is active
	fileTree.SetFocusFunc(func() {
		fileTree.SetBorderColor(tcell.ColorGreen)
		fileViewer.SetBorderColor(tcell.ColorDefault)
	})
	fileViewer.SetFocusFunc(func() {
		fileViewer.SetBorderColor(tcell.ColorGreen)
		fileTree.SetBorderColor(tcell.ColorDefault)
	})

	// Create layout for file browser
	fileBrowserFlex := tview.NewFlex()
	fileBrowserFlex.AddItem(fileTree, 0, 1, true)
	fileBrowserFlex.AddItem(fileViewer, 0, 2, false)

	// Create modal frame
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(fileBrowserFlex, 0, 8, true).
		AddItem(nil, 0, 1, false), 0, 8, true)
	modal.AddItem(nil, 0, 1, false)

	// Help text for file browser
	helpText := tview.NewTextView()
	helpText.SetText("Navigate: ↑/↓ to select file | Enter: view content/expand | Esc: back/close | Content scrollable when focused")
	helpText.SetTextAlign(tview.AlignCenter)
	helpText.SetDynamicColors(true)

	// Final modal layout
	modalLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	modalLayout.AddItem(modal, 0, 1, true)
	modalLayout.AddItem(helpText, 1, 0, false)

	// Handle key events in file browser
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if app.GetFocus() == fileViewer {
				// If viewing content, go back to file tree
				app.SetFocus(fileTree)
				return nil
			} else {
				// If on file tree, close file browser and return to main view
				app.SetInputCapture(originalInputHandler) // Restore original input handler
				app.SetRoot(mainFlex, true)
				return nil
			}
		case tcell.KeyEnter:
			if app.GetFocus() == fileTree {
				// Let the tree view handle Enter first (for expand/collapse)
				// Only switch to content viewer if a file is selected
				currentNode := fileTree.GetCurrentNode()
				if currentNode != nil {
					reference := currentNode.GetReference()
					// If it's a file (has reference), switch to content viewer
					if _, isFile := reference.(string); isFile {
						app.SetFocus(fileViewer)
						return nil
					}
					// If it's a directory (no reference), let tree handle expansion
					// Don't consume the event, let it pass through to the tree
					return event
				}
			}
			// If already viewing content, let default behavior handle scrolling
		}
		return event
	})

	// Set initial focus and selection
	app.SetFocus(fileTree)

	// Set initial selection to first file if available
	if len(filePaths) > 0 {
		// Find the first file node in the tree
		var findFirstFileNode func(node *tview.TreeNode) *tview.TreeNode
		findFirstFileNode = func(node *tview.TreeNode) *tview.TreeNode {
			if node.GetReference() != nil {
				// This is a file node
				return node
			}
			// Check children for file nodes
			for _, child := range node.GetChildren() {
				if result := findFirstFileNode(child); result != nil {
					return result
				}
			}
			return nil
		}

		if firstFileNode := findFirstFileNode(root); firstFileNode != nil {
			fileTree.SetCurrentNode(firstFileNode)
		}
	}

	app.SetRoot(modalLayout, true).EnableMouse(false)
}

func showLogsBrowser(app *tview.Application, terraformData *TerraformData, mainFlex *tview.Flex, originalInputHandler func(event *tcell.EventKey) *tcell.EventKey) {
	// Create log tree view (hierarchical)
	logTree := tview.NewTreeView()
	logTree.SetBorder(true).SetTitle("Terraform Logs")

	// Create root node
	root := tview.NewTreeNode("Logs")
	root.SetColor(tcell.ColorYellow)
	logTree.SetRoot(root)

	// Build hierarchical log structure (same as in formatTerraformLogsHierarchical)
	type LogTreeNode struct {
		Name     string
		IsPhase  bool
		Children map[string]*LogTreeNode
		Logs     []string
	}

	logStructure := &LogTreeNode{
		Name:     "root",
		IsPhase:  true,
		Children: make(map[string]*LogTreeNode),
		Logs:     []string{},
	}

	// Get sorted log paths for deterministic ordering
	logPaths := make([]string, 0, len(terraformData.Logs))
	for logPath := range terraformData.Logs {
		logPaths = append(logPaths, logPath)
	}
	sort.Strings(logPaths)

	// Parse logs into hierarchical structure
	for _, logPath := range logPaths {
		if !strings.HasPrefix(logPath, "log/") {
			continue
		}

		// Extract log filename without log/ prefix
		logName := strings.TrimPrefix(logPath, "log/")
		
		// Parse the log name to extract phase and stream info
		phase := "unknown"
		stream := "unknown"
		isPrevious := false

		if strings.HasPrefix(logName, "previous_") {
			isPrevious = true
			logName = strings.TrimPrefix(logName, "previous_")
		}

		// Parse stream_terraform_phase.log
		parts := strings.Split(logName, "_")
		if len(parts) >= 3 && parts[1] == "terraform" {
			stream = parts[0] // stdout or stderr
			phasePart := strings.Join(parts[2:], "_")
			phase = strings.TrimSuffix(phasePart, ".log")
		}

		// Create phase node (init, apply, destroy, etc.)
		phaseKey := phase
		if isPrevious {
			phaseKey = "previous_" + phase
		}

		if logStructure.Children[phaseKey] == nil {
			var displayName string
			if isPrevious {
				displayName = "Previous " + strings.ToTitle(phase[:1]) + phase[1:]
			} else {
				displayName = strings.ToTitle(phase[:1]) + phase[1:]
			}
			logStructure.Children[phaseKey] = &LogTreeNode{
				Name:     displayName,
				IsPhase:  true,
				Children: make(map[string]*LogTreeNode),
				Logs:     []string{},
			}
		}

		// Add stream (stdout/stderr) under the phase
		phaseNode := logStructure.Children[phaseKey]
		if phaseNode.Children[stream] == nil {
			streamDisplayName := strings.ToUpper(stream)
			phaseNode.Children[stream] = &LogTreeNode{
				Name:     streamDisplayName,
				IsPhase:  false,
				Children: make(map[string]*LogTreeNode),
				Logs:     []string{},
			}
		}

		// Add the actual log file
		streamNode := phaseNode.Children[stream]
		streamNode.Logs = append(streamNode.Logs, logPath)
	}

	// Build TreeView nodes from log structure
	dirNodes := make(map[string]*tview.TreeNode)

	// Helper function to get or create directory node
	var getOrCreateLogNode = func(path string, node *LogTreeNode, parent *tview.TreeNode) *tview.TreeNode {
		if existingNode, exists := dirNodes[path]; exists {
			return existingNode
		}

		// Create the node
		var treeNode *tview.TreeNode
		if node.IsPhase {
			treeNode = tview.NewTreeNode(node.Name + "/")
			treeNode.SetColor(tcell.ColorBlue)
		} else {
			treeNode = tview.NewTreeNode(node.Name)
			treeNode.SetColor(tcell.ColorLightBlue)
		}
		treeNode.SetExpanded(false) // Allow user to expand/collapse
		dirNodes[path] = treeNode
		parent.AddChild(treeNode)

		return treeNode
	}

	// Sort phases in logical order
	phaseOrder := map[string]int{
		"init":             1,
		"plan":             2,
		"apply":            3,
		"destroy":          4,
		"previous_init":    5,
		"previous_plan":    6,
		"previous_apply":   7,
		"previous_destroy": 8,
	}

	// Get sorted phase names
	var phaseNames []string
	for phaseName := range logStructure.Children {
		phaseNames = append(phaseNames, phaseName)
	}
	sort.Slice(phaseNames, func(i, j int) bool {
		orderI, hasI := phaseOrder[phaseNames[i]]
		orderJ, hasJ := phaseOrder[phaseNames[j]]
		
		if hasI && hasJ {
			return orderI < orderJ
		} else if hasI {
			return true
		} else if hasJ {
			return false
		}
		return phaseNames[i] < phaseNames[j]
	})

	// Build the tree structure
	for _, phaseName := range phaseNames {
		phaseNode := logStructure.Children[phaseName]
		phaseTreeNode := getOrCreateLogNode(phaseName, phaseNode, root)

		// Get sorted stream names (stdout, stderr)
		var streamNames []string
		for streamName := range phaseNode.Children {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)

		for _, streamName := range streamNames {
			streamNode := phaseNode.Children[streamName]
			streamPath := phaseName + "/" + streamName
			streamTreeNode := getOrCreateLogNode(streamPath, streamNode, phaseTreeNode)

			// Add log files under the stream
			for _, logPath := range streamNode.Logs {
				logName := filepath.Base(logPath)
				
				// Color code based on content or status
				logContent := terraformData.Logs[logPath]
				logFileNode := tview.NewTreeNode(logName)
				logFileNode.SetReference(logPath)
				
				if strings.Contains(strings.ToLower(logContent), "error") || strings.Contains(strings.ToLower(logContent), "failed") {
					logFileNode.SetColor(tcell.ColorRed)
				} else if strings.Contains(strings.ToLower(logContent), "warn") {
					logFileNode.SetColor(tcell.ColorYellow)
				} else if logContent != "" {
					logFileNode.SetColor(tcell.ColorGreen)
				} else {
					logFileNode.SetColor(tcell.ColorGray)
				}
				
				streamTreeNode.AddChild(logFileNode)
			}
		}
	}

	root.SetExpanded(true)

	// Create log content viewer
	logViewer := tview.NewTextView()
	logViewer.SetBorder(true).SetTitle("Log Content")
	logViewer.SetScrollable(true)
	logViewer.SetWrap(false)
	logViewer.SetDynamicColors(true) // Enable color rendering
	logViewer.SetText("Select a log from the tree to view its content")

	// Handle tree selection
	logTree.SetChangedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference != nil {
			if logPath, ok := reference.(string); ok {
				if content, exists := terraformData.Logs[logPath]; exists {
					logViewer.SetTitle(fmt.Sprintf("Log: %s", logPath))
					// Apply log syntax highlighting
					highlightedContent := addLogSyntaxHighlighting(content)
					logViewer.SetText(highlightedContent)
				}
			}
		}
	})

	// Handle tree node selection (Enter key)
	logTree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference != nil {
			// If it's a log file, show content and don't toggle expansion
			if logPath, ok := reference.(string); ok {
				if content, exists := terraformData.Logs[logPath]; exists {
					logViewer.SetTitle(fmt.Sprintf("Log: %s", logPath))
					// Apply log syntax highlighting
					highlightedContent := addLogSyntaxHighlighting(content)
					logViewer.SetText(highlightedContent)
				}
				return // Don't toggle expansion for log files
			}
		}
		// Toggle expansion for directory nodes (phases and streams)
		node.SetExpanded(!node.IsExpanded())
	})

	// Add focus handlers to show which panel is active
	logTree.SetFocusFunc(func() {
		logTree.SetBorderColor(tcell.ColorGreen)
		logViewer.SetBorderColor(tcell.ColorDefault)
	})
	logViewer.SetFocusFunc(func() {
		logViewer.SetBorderColor(tcell.ColorGreen)
		logTree.SetBorderColor(tcell.ColorDefault)
	})

	// Create layout for log browser
	logBrowserFlex := tview.NewFlex()
	logBrowserFlex.AddItem(logTree, 0, 1, true)
	logBrowserFlex.AddItem(logViewer, 0, 2, false)

	// Create modal frame
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(logBrowserFlex, 0, 8, true).
		AddItem(nil, 0, 1, false), 0, 8, true)
	modal.AddItem(nil, 0, 1, false)

	// Help text for log browser
	helpText := tview.NewTextView()
	helpText.SetText("Navigate: ↑/↓ to select log | Enter: view content/expand | Esc: back/close | Content scrollable when focused")
	helpText.SetTextAlign(tview.AlignCenter)
	helpText.SetDynamicColors(true)

	// Final modal layout
	modalLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	modalLayout.AddItem(modal, 0, 1, true)
	modalLayout.AddItem(helpText, 1, 0, false)

	// Handle key events in log browser
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if app.GetFocus() == logViewer {
				// If viewing content, go back to log tree
				app.SetFocus(logTree)
				return nil
			} else {
				// If on log tree, close log browser and return to main view
				app.SetInputCapture(originalInputHandler) // Restore original input handler
				app.SetRoot(mainFlex, true)
				return nil
			}
		case tcell.KeyEnter:
			if app.GetFocus() == logTree {
				// Let the tree view handle Enter first (for expand/collapse)
				// Only switch to content viewer if a log is selected
				currentNode := logTree.GetCurrentNode()
				if currentNode != nil {
					reference := currentNode.GetReference()
					// If it's a log file (has reference), switch to content viewer
					if _, isLogFile := reference.(string); isLogFile {
						app.SetFocus(logViewer)
						return nil
					}
					// If it's a directory (no reference), let tree handle expansion
					// Don't consume the event, let it pass through to the tree
					return event
				}
			}
			// If already viewing content, let default behavior handle scrolling
		}
		return event
	})

	// Set initial focus and selection
	app.SetFocus(logTree)

	// Set initial selection to first log if available
	if len(logPaths) > 0 {
		// Find the first log file node in the tree
		var findFirstLogNode func(node *tview.TreeNode) *tview.TreeNode
		findFirstLogNode = func(node *tview.TreeNode) *tview.TreeNode {
			if node.GetReference() != nil {
				// This is a log file node
				return node
			}
			// Check children for log nodes
			for _, child := range node.GetChildren() {
				if result := findFirstLogNode(child); result != nil {
					return result
				}
			}
			return nil
		}

		if firstLogNode := findFirstLogNode(root); firstLogNode != nil {
			logTree.SetCurrentNode(firstLogNode)
		}
	}

	app.SetRoot(modalLayout, true).EnableMouse(false)
}

func init() {
	// Command will be added by the parent instance command
}





var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Clean log line for live logs (no color, no escape codes)
func cleanLiveLogLine(line string) string {
	return ansiEscape.ReplaceAllString(line, "")
}


// Connect to websocket and stream logs to the rightPanel (reusable, modeled after logs.go)
func connectAndStreamLogs(app *tview.Application, logsUrl string, rightPanel *tview.TextView) {
	if logsUrl == "" {
		app.QueueUpdateDraw(func() {
			rightPanel.SetText("[red]No log URL provided[-]")
		})
		return
	}
	go func() {
		for {
			c, _, err := websocket.DefaultDialer.Dial(logsUrl, nil)
			if err != nil {
				app.QueueUpdateDraw(func() {
					rightPanel.SetText(fmt.Sprintf("[red]Failed to connect: %v[-]", err))
				})
				time.Sleep(5 * time.Second)
				continue
			}
			defer c.Close()
			app.QueueUpdateDraw(func() {
				rightPanel.SetText("")
			})
			
			// Batching mechanism for performance optimization
			var logBatch []string
			batchTicker := time.NewTicker(100 * time.Millisecond) // Process batch every 100ms
			defer batchTicker.Stop()
			
			// Channel to signal when to stop batching
			done := make(chan bool)
			
			// Goroutine to process batched logs
			go func() {
				for {
					select {
					case <-batchTicker.C:
						if len(logBatch) > 0 {
							// Process and display the batch
							var formattedBatch strings.Builder
							for _, line := range logBatch {
								cleanedLogLine := cleanLiveLogLine(line)
								formatted := addLogSyntaxHighlighting(cleanedLogLine)
								formattedBatch.WriteString(formatted + "\n")
							}
							
							app.QueueUpdateDraw(func() {
								_, _ = rightPanel.Write([]byte(formattedBatch.String()))
							})
							
							// Clear the batch
							logBatch = logBatch[:0]
						}
					case <-done:
						return
					}
				}
			}()
			
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					done <- true // Stop the batching goroutine
					app.QueueUpdateDraw(func() {
						rightPanel.SetText(fmt.Sprintf("[yellow]Connection closed: %v[-]", err))
					})
					break
				}
				
				// Add to batch instead of processing immediately
				logBatch = append(logBatch, string(message))
				
				// If batch gets too large, process immediately to avoid memory issues
				if len(logBatch) >= 50 {
					var formattedBatch strings.Builder
					for _, line := range logBatch {
						cleanedLogLine := cleanLiveLogLine(line)
						formatted := addLogSyntaxHighlighting(cleanedLogLine)
						formattedBatch.WriteString(formatted + "\n")
					}
					
					app.QueueUpdateDraw(func() {
						_, _ = rightPanel.Write([]byte(formattedBatch.String()))
					})
					
					// Clear the batch
					logBatch = logBatch[:0]
				}
			}
			c.Close()
			time.Sleep(5 * time.Second)
		}
	}()
}