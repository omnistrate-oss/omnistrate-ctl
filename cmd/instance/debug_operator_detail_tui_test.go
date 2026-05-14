package instance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperatorTabNames(t *testing.T) {
	require.Equal(t, opNumTabs, len(opTabNames), "opTabNames length must match opNumTabs")
	require.Equal(t, "Input Variables", opTabNames[opTabInputVars])
	require.Equal(t, "Output Variables", opTabNames[opTabOutputVars])
	require.Equal(t, "Workflow Events", opTabNames[opTabWfErrors])
}

func TestNewOperatorDetailModel(t *testing.T) {
	node := PlanDAGNode{ID: "r1", Key: "my-op", Name: "My Operator", Type: "OperatorCRD"}
	data := DebugData{InstanceID: "inst-1", ServiceID: "svc-1"}

	m := newOperatorDetailModel(node, data)

	require.Equal(t, opTabInputVars, m.activeTab)
	require.True(t, m.loading)
	require.NotNil(t, m.wfErrors)
	require.Equal(t, "my-op", m.node.Key)
}

func TestBuildOperatorParamTree(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := buildOperatorParamTree(nil)
		require.Nil(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := buildOperatorParamTree([]OperatorInputParam{})
		require.Nil(t, result)
	})

	t.Run("single param", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "replicas", DisplayName: "Replicas", Description: "Number of replicas", Type: "int", Required: true, DefaultValue: "3"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 1)
		require.Contains(t, result[0].key, "replicas")
		require.True(t, result[0].expandable)
	})

	t.Run("multiple params sorted", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "zone", DisplayName: "Zone", Type: "string"},
			{Key: "cpu", DisplayName: "CPU", Type: "string"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 2)
		require.Contains(t, result[0].key, "cpu")
		require.Contains(t, result[1].key, "zone")
	})

	t.Run("display name differs from key", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "mem", DisplayName: "Memory Size", Type: "string"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "mem (Memory Size)", result[0].key)
	})

	t.Run("display name same as key", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "replicas", DisplayName: "replicas", Type: "int"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "replicas", result[0].key)
	})
}

func TestBuildOperatorOutputParamTree(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := buildOperatorOutputParamTree(nil)
		require.Nil(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := buildOperatorOutputParamTree([]OperatorOutputParam{})
		require.Nil(t, result)
	})

	t.Run("single param", func(t *testing.T) {
		params := []OperatorOutputParam{
			{Key: "status", DisplayName: "Status", Description: "CRD status", Type: "string"},
		}
		result := buildOperatorOutputParamTree(params)
		require.Len(t, result, 1)
		require.Contains(t, result[0].key, "status")
		require.True(t, result[0].expandable)
	})

	t.Run("multiple params sorted", func(t *testing.T) {
		params := []OperatorOutputParam{
			{Key: "topology", Type: "string"},
			{Key: "image", Type: "string"},
		}
		result := buildOperatorOutputParamTree(params)
		require.Len(t, result, 2)
		require.Contains(t, result[0].key, "image")
		require.Contains(t, result[1].key, "topology")
	})
}

func TestOperatorCopyableContent(t *testing.T) {
	node := PlanDAGNode{ID: "r1", Key: "my-op", Name: "Op", Type: "OperatorCRD"}
	data := DebugData{InstanceID: "inst-1"}

	m := newOperatorDetailModel(node, data)
	m.loading = false
	m.operatorData = &OperatorData{
		InputParams: []OperatorInputParam{
			{Key: "replicas", Type: "int"},
		},
		OutputParams: []OperatorOutputParam{
			{Key: "status", Type: "string"},
		},
	}

	t.Run("input vars tab", func(t *testing.T) {
		m.activeTab = opTabInputVars
		content := m.opCopyableContent()
		require.Contains(t, content, "replicas")
	})

	t.Run("output vars tab", func(t *testing.T) {
		m.activeTab = opTabOutputVars
		content := m.opCopyableContent()
		require.Contains(t, content, "status")
	})

	t.Run("wf errors tab empty", func(t *testing.T) {
		m.activeTab = opTabWfErrors
		content := m.opCopyableContent()
		require.Empty(t, content)
	})
}
