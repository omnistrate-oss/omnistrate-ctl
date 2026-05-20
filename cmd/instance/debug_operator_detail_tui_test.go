package instance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperatorTabNames(t *testing.T) {
	require.Equal(t, opNumTabs, len(opTabNames), "opTabNames length must match opNumTabs")
	require.Equal(t, "Deployment API parameters", opTabNames[opTabInputVars])
	require.Equal(t, "Deployment Output Parameters", opTabNames[opTabOutputVars])
	require.Equal(t, "Operator CRD Outputs", opTabNames[opTabCRDOutputVars])
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
		require.False(t, result[0].expandable)
		require.Equal(t, "3", result[0].value)
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

	t.Run("resolved value shown", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "instanceType", DisplayName: "Instance Type", Description: "Instance Type", Type: "String", DefaultValue: "t3.medium", ResolvedValue: "t3.large"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "t3.large", result[0].value, "expected resolved value as node value")
		require.False(t, result[0].expandable)
	})

	t.Run("fallback to defaultValue when no resolved value", func(t *testing.T) {
		params := []OperatorInputParam{
			{Key: "replicas", Description: "Replicas", Type: "int", DefaultValue: "3"},
		}
		result := buildOperatorParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "3", result[0].value, "expected defaultValue as node value when no resolved value")
		require.False(t, result[0].expandable)
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
		require.False(t, result[0].expandable)
	})

	t.Run("multiple params sorted", func(t *testing.T) {
		params := []OperatorOutputParam{
			{Key: "topology", Description: "Topology info"},
			{Key: "image", Description: "Image name"},
		}
		result := buildOperatorOutputParamTree(params)
		require.Len(t, result, 2)
		require.Contains(t, result[0].key, "image")
		require.Contains(t, result[1].key, "topology")
	})

	t.Run("resolved value shown", func(t *testing.T) {
		params := []OperatorOutputParam{
			{Key: "endpoint", DisplayName: "Endpoint", Description: "Connection endpoint", ValueRef: "$var.endpoint", ResolvedValue: "db.example.com:5432"},
		}
		result := buildOperatorOutputParamTree(params)
		require.Len(t, result, 1)
		require.False(t, result[0].expandable)
		require.Equal(t, "db.example.com:5432", result[0].value, "expected resolved value as node value")
	})

	t.Run("fallback to static value when no resolved value", func(t *testing.T) {
		params := []OperatorOutputParam{
			{Key: "endpoint", Description: "Connection endpoint", Value: "static-val"},
		}
		result := buildOperatorOutputParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "static-val", result[0].value, "expected static value as node value when no resolved value")
		require.False(t, result[0].expandable)
	})
}

func TestBuildOperatorCRDOutputParamTree(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := buildOperatorCRDOutputParamTree(nil)
		require.Nil(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := buildOperatorCRDOutputParamTree([]OperatorCRDOutputParam{})
		require.Nil(t, result)
	})

	t.Run("single param", func(t *testing.T) {
		params := []OperatorCRDOutputParam{
			{Key: "endpoint", Value: ".status.endpoint"},
		}
		result := buildOperatorCRDOutputParamTree(params)
		require.Len(t, result, 1)
		require.Contains(t, result[0].key, "endpoint")
		require.False(t, result[0].expandable)
		require.Equal(t, ".status.endpoint", result[0].value)
	})

	t.Run("multiple params sorted", func(t *testing.T) {
		params := []OperatorCRDOutputParam{
			{Key: "topology", Value: ".status.topology"},
			{Key: "image", Value: ".spec.image"},
		}
		result := buildOperatorCRDOutputParamTree(params)
		require.Len(t, result, 2)
		require.Contains(t, result[0].key, "image")
		require.Contains(t, result[1].key, "topology")
	})

	t.Run("resolved value shown", func(t *testing.T) {
		params := []OperatorCRDOutputParam{
			{Key: "endpoint", Value: ".status.endpoint", ResolvedValue: "db.example.com:5432"},
		}
		result := buildOperatorCRDOutputParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, "db.example.com:5432", result[0].value, "expected resolved value as node value")
		require.False(t, result[0].expandable)
	})

	t.Run("no resolved value", func(t *testing.T) {
		params := []OperatorCRDOutputParam{
			{Key: "endpoint", Value: ".status.endpoint"},
		}
		result := buildOperatorCRDOutputParamTree(params)
		require.Len(t, result, 1)
		require.Equal(t, ".status.endpoint", result[0].value, "expected jsonPath as fallback value")
		require.False(t, result[0].expandable)
	})
}

func TestOperatorCopyableContent(t *testing.T) {
	node := PlanDAGNode{ID: "r1", Key: "my-op", Name: "Op", Type: "OperatorCRD"}
	data := DebugData{InstanceID: "inst-1"}

	m := newOperatorDetailModel(node, data)
	m.loading = false
	m.operatorData = &OperatorData{
		InputParams: []OperatorInputParam{
			{Key: "replicas", Type: "int", Required: true},
		},
		OutputParams: []OperatorOutputParam{
			{Key: "status", Description: "CRD status", Type: "string"},
		},
		CRDOutputParams: []OperatorCRDOutputParam{
			{Key: "endpoint", Value: ".status.endpoint"},
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

	t.Run("crd output vars tab", func(t *testing.T) {
		m.activeTab = opTabCRDOutputVars
		content := m.opCopyableContent()
		require.Contains(t, content, "endpoint")
	})

	t.Run("wf errors tab empty", func(t *testing.T) {
		m.activeTab = opTabWfErrors
		content := m.opCopyableContent()
		require.Empty(t, content)
	})
}
