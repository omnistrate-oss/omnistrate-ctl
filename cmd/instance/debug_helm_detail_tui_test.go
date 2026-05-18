package instance

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestHelmValuesEnterTogglesSelectedNode(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabValues,
		valuesTree: buildHelmValuesTree(map[string]interface{}{
			"parent": map[string]interface{}{
				"child": "value",
			},
		}, ""),
	}

	visibleNodes := flattenOutputTree(model.valuesTree)
	if len(visibleNodes) != 2 {
		t.Fatalf("expected 2 visible nodes before toggle, got %d", len(visibleNodes))
	}
	if !visibleNodes[0].expandable || !visibleNodes[0].expanded {
		t.Fatalf("expected root node to start expanded")
	}

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd on first toggle, got %v", cmd)
	}
	updated, ok := updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	visibleNodes = flattenOutputTree(updated.valuesTree)
	if len(visibleNodes) != 1 {
		t.Fatalf("expected 1 visible node after collapse, got %d", len(visibleNodes))
	}
	if visibleNodes[0].expanded {
		t.Fatalf("expected root node to collapse on enter")
	}

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd on second toggle, got %v", cmd)
	}
	updated, ok = updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	visibleNodes = flattenOutputTree(updated.valuesTree)
	if len(visibleNodes) != 2 {
		t.Fatalf("expected 2 visible nodes after re-expand, got %d", len(visibleNodes))
	}
	if !visibleNodes[0].expanded {
		t.Fatalf("expected root node to expand again on enter")
	}
}

func TestRenderHelmValuesTabClampsLongRows(t *testing.T) {
	values := map[string]interface{}{
		"alpha":  "short",
		"script": "line one\n" + strings.Repeat("x", 120) + "\nline three",
		"zeta":   "tail",
	}

	model := helmDetailModel{
		activeTab:    helmTabValues,
		width:        72,
		height:       14,
		helmData:     &HelmData{ChartRepoName: "repo/chart", ChartValues: values},
		valuesTree:   buildHelmValuesTree(values, ""),
		valuesCursor: 1,
	}

	rendered := model.renderHelmValuesTab()
	if !strings.Contains(rendered, "Chart Values") {
		t.Fatalf("expected chart values header in output")
	}

	maxWidth := model.helmContentWidth()
	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > maxWidth {
			t.Fatalf("rendered line width %d exceeded content width %d: %q", lipgloss.Width(line), maxWidth, line)
		}
	}
}

func TestHelmValuesUpFromBottomKeepsViewportFixed(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabValues,
		width:     72,
		height:    15, // yields 3 visible value rows
		wfErrors:  &workflowErrorsState{},
		valuesTree: buildHelmValuesTree(map[string]interface{}{
			"a": 1,
			"b": 2,
			"c": 3,
			"d": 4,
			"e": 5,
			"f": 6,
		}, ""),
	}

	for range 5 {
		updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		if cmd != nil {
			t.Fatalf("expected nil cmd while moving down, got %v", cmd)
		}
		updated, ok := updatedAny.(helmDetailModel)
		if !ok {
			t.Fatalf("expected helmDetailModel, got %T", updatedAny)
		}
		model = updated
	}

	if model.valuesCursor != 5 || model.valuesScroll != 3 {
		t.Fatalf("expected cursor/scroll at bottom to be 5/3, got %d/%d", model.valuesCursor, model.valuesScroll)
	}

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd while moving up, got %v", cmd)
	}
	updated, ok := updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	if updated.valuesCursor != 4 {
		t.Fatalf("expected cursor to move to 4, got %d", updated.valuesCursor)
	}
	if updated.valuesScroll != 3 {
		t.Fatalf("expected viewport to stay anchored at scroll 3, got %d", updated.valuesScroll)
	}
}

func TestHelmTabConstants(t *testing.T) {
	require := require.New(t)

	require.Equal(0, helmTabLogs)
	require.Equal(1, helmTabValues)
	require.Equal(2, helmTabInputVars)
	require.Equal(3, helmTabOutputVars)
	require.Equal(4, helmTabWfErrors)
	require.Equal(5, helmNumTabs)
	require.Len(helmTabNames, helmNumTabs)
	require.Equal("Input Parameters", helmTabNames[helmTabInputVars])
	require.Equal("Output Parameters", helmTabNames[helmTabOutputVars])
}

func TestHelmInputParamTreeBuildsFromOperatorInputParam(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "db_port", DisplayName: "Database Port", Description: "Port for DB", Type: "String", Required: true, Modifiable: false, DefaultValue: "5432"},
		{Key: "app_name", DisplayName: "App Name", Description: "Application name", Type: "String", Required: false},
	}

	tree := buildOperatorParamTree(params)
	require.NotNil(t, tree)
	require.Len(t, tree, 2)

	// Params are sorted by key
	require.Equal(t, "app_name (App Name)", tree[0].key)
	require.Equal(t, "db_port (Database Port)", tree[1].key)
	require.False(t, tree[0].expandable)
	require.False(t, tree[1].expandable)
}

func TestHelmOutputParamTreeBuildsFromOperatorOutputParam(t *testing.T) {
	params := []OperatorOutputParam{
		{Key: "endpoint", DisplayName: "Endpoint URL", Description: "Service endpoint", Value: "https://example.com", Type: "String"},
		{Key: "status", DisplayName: "Status", Description: "Current status", ValueRef: "$.status"},
	}

	tree := buildOperatorOutputParamTree(params)
	require.NotNil(t, tree)
	require.Len(t, tree, 2)

	// Params are sorted by key
	require.Equal(t, "endpoint (Endpoint URL)", tree[0].key)
	require.Equal(t, "status (Status)", tree[1].key)
}

func TestHelmInputTabEnterOnLeafNode(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "port", DisplayName: "Port", Description: "Port number", Type: "String", Required: true},
	}

	model := helmDetailModel{
		activeTab: helmTabInputVars,
		inputTree: buildOperatorParamTree(params),
		wfErrors:  &workflowErrorsState{},
	}

	visibleNodes := flattenOutputTree(model.inputTree)
	require.Len(t, visibleNodes, 1) // single leaf node
	require.False(t, visibleNodes[0].expandable)

	// Enter on leaf node should be a no-op
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedAny.(helmDetailModel)

	visibleNodes = flattenOutputTree(updated.inputTree)
	require.Len(t, visibleNodes, 1)
}

func TestHelmOutputTabNavigationUpDown(t *testing.T) {
	params := []OperatorOutputParam{
		{Key: "a", DisplayName: "A", Description: "Param A"},
		{Key: "b", DisplayName: "B", Description: "Param B"},
	}

	model := helmDetailModel{
		activeTab:    helmTabOutputVars,
		width:        80,
		height:       20,
		outputTree:   buildOperatorOutputParamTree(params),
		outputCursor: 0,
		wfErrors:     &workflowErrorsState{},
	}

	// Move down
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := updatedAny.(helmDetailModel)
	require.Equal(t, 1, updated.outputCursor)

	// Move up
	updatedAny, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = updatedAny.(helmDetailModel)
	require.Equal(t, 0, updated.outputCursor)
}

func TestHelmInputTabLeftRightOnLeafNode(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "config", DisplayName: "Config", Description: "Configuration", Type: "String"},
	}

	model := helmDetailModel{
		activeTab: helmTabInputVars,
		width:     80,
		height:    20,
		inputTree: buildOperatorParamTree(params),
		wfErrors:  &workflowErrorsState{},
	}

	// Left on leaf is a no-op
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated := updatedAny.(helmDetailModel)
	visibleNodes := flattenOutputTree(updated.inputTree)
	require.Len(t, visibleNodes, 1)
	require.False(t, visibleNodes[0].expandable)

	// Right on leaf is a no-op
	updatedAny, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = updatedAny.(helmDetailModel)
	visibleNodes = flattenOutputTree(updated.inputTree)
	require.Len(t, visibleNodes, 1)
	require.False(t, visibleNodes[0].expandable)
}

func TestHelmRenderParamTreeTabShowsTitle(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "name", DisplayName: "Name", Description: "Resource name", Type: "String"},
	}

	model := helmDetailModel{
		activeTab: helmTabInputVars,
		width:     80,
		height:    20,
		inputTree: buildOperatorParamTree(params),
		helmData:  &HelmData{InputParams: params},
		wfErrors:  &workflowErrorsState{},
	}

	rendered := model.renderHelmParamTreeTab("Input Parameters", model.inputTree, model.inputCursor, model.inputScroll)
	require.Contains(t, rendered, "Input Parameters")
}

func TestHelmRenderParamTreeTabEmptyShowsNoDataMessage(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabInputVars,
		width:     80,
		height:    20,
		inputTree: nil,
		wfErrors:  &workflowErrorsState{},
	}

	rendered := model.renderHelmParamTreeTab("Input Parameters", model.inputTree, 0, 0)
	require.Contains(t, rendered, "No input parameters available")
}

func TestHelmCopyableContentInputOutputTabs(t *testing.T) {
	inputParams := []OperatorInputParam{
		{Key: "port", DisplayName: "Port", Type: "String"},
	}
	outputParams := []OperatorOutputParam{
		{Key: "endpoint", DisplayName: "Endpoint", Value: "https://example.com"},
	}

	model := helmDetailModel{
		helmData: &HelmData{
			InputParams:  inputParams,
			OutputParams: outputParams,
		},
		wfErrors: &workflowErrorsState{},
	}

	// Input params copy
	model.activeTab = helmTabInputVars
	text := model.helmCopyableContent()
	require.NotEmpty(t, text)
	var decoded []OperatorInputParam
	require.NoError(t, json.Unmarshal([]byte(text), &decoded))
	require.Len(t, decoded, 1)
	require.Equal(t, "port", decoded[0].Key)

	// Output params copy
	model.activeTab = helmTabOutputVars
	text = model.helmCopyableContent()
	require.NotEmpty(t, text)
	var decodedOut []OperatorOutputParam
	require.NoError(t, json.Unmarshal([]byte(text), &decodedOut))
	require.Len(t, decodedOut, 1)
	require.Equal(t, "endpoint", decodedOut[0].Key)
}

func TestHelmTabCyclesAllFiveTabs(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabLogs,
		wfErrors:  &workflowErrorsState{},
	}

	for i := 0; i < helmNumTabs; i++ {
		require.Equal(t, i, model.activeTab)
		updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
		model = updatedAny.(helmDetailModel)
	}
	// Should wrap to 0
	require.Equal(t, helmTabLogs, model.activeTab)
}

func TestHelmDataJSONIncludesInputOutputParams(t *testing.T) {
	require := require.New(t)

	helmData := &HelmData{
		ChartRepoName: "test-repo",
		ChartVersion:  "1.0.0",
		InputParams: []OperatorInputParam{
			{Key: "port", DisplayName: "Port", Type: "String", Required: true},
		},
		OutputParams: []OperatorOutputParam{
			{Key: "endpoint", DisplayName: "Endpoint", Value: "https://example.com"},
		},
	}

	jsonBytes, err := json.Marshal(helmData)
	require.NoError(err)

	var decoded map[string]interface{}
	require.NoError(json.Unmarshal(jsonBytes, &decoded))

	require.Contains(decoded, "inputParams")
	require.Contains(decoded, "outputParams")

	inputArr, ok := decoded["inputParams"].([]interface{})
	require.True(ok)
	require.Len(inputArr, 1)

	outputArr, ok := decoded["outputParams"].([]interface{})
	require.True(ok)
	require.Len(outputArr, 1)
}

func TestHelmDataJSONOmitsEmptyInputOutputParams(t *testing.T) {
	require := require.New(t)

	helmData := &HelmData{
		ChartRepoName: "test-repo",
	}

	jsonBytes, err := json.Marshal(helmData)
	require.NoError(err)

	var decoded map[string]interface{}
	require.NoError(json.Unmarshal(jsonBytes, &decoded))

	require.NotContains(decoded, "inputParams")
	require.NotContains(decoded, "outputParams")
}

func TestHelmFooterShowsTreeNavForInputOutputTabs(t *testing.T) {
	model := helmDetailModel{
		width:    80,
		height:   20,
		wfErrors: &workflowErrorsState{},
		inputTree: buildOperatorParamTree([]OperatorInputParam{
			{Key: "x", DisplayName: "X", Description: "test"},
		}),
		outputTree: buildOperatorOutputParamTree([]OperatorOutputParam{
			{Key: "y", DisplayName: "Y", Description: "test"},
		}),
	}

	model.activeTab = helmTabInputVars
	footer := model.renderHelmFooter()
	require.Contains(t, footer, "expand/collapse")

	model.activeTab = helmTabOutputVars
	footer = model.renderHelmFooter()
	require.Contains(t, footer, "expand/collapse")
}
