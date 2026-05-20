package instance

import (
	"encoding/json"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestComposeTabNames(t *testing.T) {
	require.Equal(t, composeNumTabs, len(composeTabNames), "composeTabNames length must match composeNumTabs")
	require.Equal(t, "Deployment API parameters", composeTabNames[composeTabInputVars])
	require.Equal(t, "Deployment Output Parameters", composeTabNames[composeTabOutputVars])
	require.Equal(t, "Workflow Events", composeTabNames[composeTabWfErrors])
}

func TestNewComposeDetailModel(t *testing.T) {
	node := PlanDAGNode{ID: "r1", Key: "my-compose", Name: "My Compose", Type: "DockerCompose"}
	data := DebugData{InstanceID: "inst-1", ServiceID: "svc-1"}

	m := newComposeDetailModel(node, data)

	require.Equal(t, composeTabInputVars, m.activeTab)
	require.True(t, m.loading)
	require.NotNil(t, m.wfErrors)
	require.Equal(t, "my-compose", m.node.Key)
}

func TestComposeTabCyclesAllThreeTabs(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabInputVars,
		wfErrors:  &workflowErrorsState{},
	}

	for i := 0; i < composeNumTabs; i++ {
		require.Equal(t, i, model.activeTab)
		updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
		model = updatedAny.(composeDetailModel)
	}
	// Should wrap to 0
	require.Equal(t, composeTabInputVars, model.activeTab)
}

func TestComposeShiftTabReverses(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabInputVars,
		wfErrors:  &workflowErrorsState{},
	}

	// Shift+tab from first goes to last
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updatedAny.(composeDetailModel)
	require.Equal(t, composeTabWfErrors, model.activeTab)
}

func TestComposeInputTabNavigationUpDown(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "a", DisplayName: "A", Type: "String", ResolvedValue: "val-a"},
		{Key: "b", DisplayName: "B", Type: "String", ResolvedValue: "val-b"},
	}

	model := composeDetailModel{
		activeTab: composeTabInputVars,
		width:     80,
		height:    20,
		inputTree: buildOperatorParamTree(params),
		wfErrors:  &workflowErrorsState{},
	}

	// Move down
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := updatedAny.(composeDetailModel)
	require.Equal(t, 1, updated.inputCursor)

	// Move up
	updatedAny, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = updatedAny.(composeDetailModel)
	require.Equal(t, 0, updated.inputCursor)
}

func TestComposeOutputTabNavigationUpDown(t *testing.T) {
	params := []OperatorOutputParam{
		{Key: "x", DisplayName: "X", Description: "Param X"},
		{Key: "y", DisplayName: "Y", Description: "Param Y"},
	}

	model := composeDetailModel{
		activeTab:    composeTabOutputVars,
		width:        80,
		height:       20,
		outputTree:   buildOperatorOutputParamTree(params),
		outputCursor: 0,
		wfErrors:     &workflowErrorsState{},
	}

	// Move down
	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := updatedAny.(composeDetailModel)
	require.Equal(t, 1, updated.outputCursor)

	// Move up
	updatedAny, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = updatedAny.(composeDetailModel)
	require.Equal(t, 0, updated.outputCursor)
}

func TestComposeCopyableContent(t *testing.T) {
	node := PlanDAGNode{ID: "r1", Key: "my-compose", Name: "Compose", Type: "DockerCompose"}
	data := DebugData{InstanceID: "inst-1"}

	m := newComposeDetailModel(node, data)
	m.loading = false
	m.composeData = &ComposeData{
		InputParams: []OperatorInputParam{
			{Key: "replicas", Type: "int", Required: true},
		},
		OutputParams: []OperatorOutputParam{
			{Key: "status", Description: "Status", Type: "string"},
		},
	}

	t.Run("input vars tab", func(t *testing.T) {
		m.activeTab = composeTabInputVars
		content := m.composeCopyableContent()
		require.Contains(t, content, "replicas")
	})

	t.Run("output vars tab", func(t *testing.T) {
		m.activeTab = composeTabOutputVars
		content := m.composeCopyableContent()
		require.Contains(t, content, "status")
	})

	t.Run("wf errors tab with no events", func(t *testing.T) {
		m.activeTab = composeTabWfErrors
		content := m.composeCopyableContent()
		require.Empty(t, content)
	})
}

func TestComposeDataJSONIncludesInputOutputParams(t *testing.T) {
	require := require.New(t)

	composeData := &ComposeData{
		InputParams: []OperatorInputParam{
			{Key: "port", DisplayName: "Port", Type: "String", Required: true},
		},
		OutputParams: []OperatorOutputParam{
			{Key: "endpoint", DisplayName: "Endpoint", Value: "https://example.com"},
		},
	}

	jsonBytes, err := json.Marshal(composeData)
	require.NoError(err)

	var decoded map[string]interface{}
	require.NoError(json.Unmarshal(jsonBytes, &decoded))

	require.Contains(decoded, "inputParams")
	require.Contains(decoded, "outputParams")
}

func TestComposeDataJSONOmitsEmptyInputOutputParams(t *testing.T) {
	require := require.New(t)

	composeData := &ComposeData{}

	jsonBytes, err := json.Marshal(composeData)
	require.NoError(err)

	var decoded map[string]interface{}
	require.NoError(json.Unmarshal(jsonBytes, &decoded))

	require.NotContains(decoded, "inputParams")
	require.NotContains(decoded, "outputParams")
}

func TestComposeRenderParamTreeTabShowsTitle(t *testing.T) {
	params := []OperatorInputParam{
		{Key: "name", DisplayName: "Name", Type: "String", ResolvedValue: "my-app"},
	}

	model := composeDetailModel{
		activeTab:   composeTabInputVars,
		width:       80,
		height:      20,
		inputTree:   buildOperatorParamTree(params),
		composeData: &ComposeData{InputParams: params},
		wfErrors:    &workflowErrorsState{},
	}

	rendered := model.renderComposeParamTreeTab("Deployment API parameters", model.inputTree, model.inputCursor, model.inputScroll, nil)
	require.Contains(t, rendered, "Deployment API parameters")
}

func TestComposeRenderParamTreeTabEmptyShowsNoDataMessage(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabInputVars,
		width:     80,
		height:    20,
		inputTree: nil,
		wfErrors:  &workflowErrorsState{},
	}

	rendered := model.renderComposeParamTreeTab("Deployment API parameters", model.inputTree, 0, 0, nil)
	require.Contains(t, rendered, "No deployment api parameters available")
}

func TestComposeRenderParamTreeTabShowsFetchError(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabInputVars,
		width:     80,
		height:    20,
		wfErrors:  &workflowErrorsState{},
	}

	rendered := model.renderComposeParamTreeTab("Deployment API parameters", model.inputTree, 0, 0, errors.New("request failed"))
	require.Contains(t, rendered, "Error fetching deployment api parameters")
	require.Contains(t, rendered, "request failed")
	require.NotContains(t, rendered, "No input parameters available")
}

func TestComposeFooterShowsNavigationForInputOutputTabs(t *testing.T) {
	model := composeDetailModel{
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

	model.activeTab = composeTabInputVars
	footer := model.renderComposeFooter()
	require.Contains(t, footer, "↑↓: navigate")
	require.Contains(t, footer, "y: copy")
	require.NotContains(t, footer, "expand/collapse")

	model.activeTab = composeTabOutputVars
	footer = model.renderComposeFooter()
	require.Contains(t, footer, "↑↓: navigate")
	require.Contains(t, footer, "y: copy")
	require.NotContains(t, footer, "expand/collapse")

	model.activeTab = composeTabWfErrors
	footer = model.renderComposeFooter()
	require.Contains(t, footer, "pgup/pgdn")
}

func TestResourceDebugInfoHasDataWithCompose(t *testing.T) {
	info := &ResourceDebugInfo{
		Compose: &ComposeData{
			InputParams: []OperatorInputParam{{Key: "x"}},
		},
	}
	require.True(t, info.hasData())

	emptyInfo := &ResourceDebugInfo{}
	require.False(t, emptyInfo.hasData())
}
