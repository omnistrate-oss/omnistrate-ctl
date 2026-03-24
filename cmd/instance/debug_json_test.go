package instance

import (
	"encoding/json"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/require"
)

func TestDebugDataJSONIncludesServiceAndEnvironment(t *testing.T) {
	require := require.New(t)

	data := DebugData{
		InstanceID:    "inst-1",
		ServiceID:     "svc-1",
		EnvironmentID: "env-1",
	}

	jsonBytes, err := json.Marshal(data)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("inst-1", decoded["instanceId"])
	require.Equal("svc-1", decoded["serviceId"])
	require.Equal("env-1", decoded["environmentId"])
	require.NotContains(decoded, "token", "token should be excluded from JSON")
}

func TestDebugDataJSONOmitsEmptyServiceAndEnvironment(t *testing.T) {
	require := require.New(t)

	data := DebugData{
		InstanceID: "inst-1",
	}

	jsonBytes, err := json.Marshal(data)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.NotContains(decoded, "serviceId", "empty serviceId should be omitted")
	require.NotContains(decoded, "environmentId", "empty environmentId should be omitted")
}

func TestPlanDAGJSONIncludesWorkflowSteps(t *testing.T) {
	require := require.New(t)

	plan := &PlanDAG{
		Nodes: map[string]PlanDAGNode{
			"r-1": {ID: "r-1", Key: "db", Name: "Database", Type: "terraform"},
		},
		Edges:    []PlanDAGEdge{{From: "r-1", To: "r-2"}},
		Levels:   [][]string{{"r-1"}, {"r-2"}},
		HasCycle: false,
		WorkflowStepsByKey: map[string]*ResourceWorkflowSteps{
			"db": {
				Steps: []WorkflowStepInfo{
					{
						Name:      "Bootstrap",
						Status:    "success",
						StartTime: "2024-01-01T00:00:00Z",
						EndTime:   "2024-01-01T00:01:00Z",
						Events: []dataaccess.DebugEvent{
							{EventTime: "2024-01-01T00:00:00Z", EventType: "started", Message: "Starting bootstrap"},
							{EventTime: "2024-01-01T00:01:00Z", EventType: "completed", Message: "Bootstrap completed"},
						},
					},
					{
						Name:        "Network",
						DisplayName: "Network Setup",
						Status:      "in-progress",
						StartTime:   "2024-01-01T00:01:00Z",
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(plan)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	// Verify workflowStepsByKey is present
	require.Contains(decoded, "workflowStepsByKey")
	stepsByKey, ok := decoded["workflowStepsByKey"].(map[string]interface{})
	require.True(ok, "workflowStepsByKey should be a map")

	// Verify db resource steps
	require.Contains(stepsByKey, "db")
	dbSteps, ok := stepsByKey["db"].(map[string]interface{})
	require.True(ok)

	steps, ok := dbSteps["steps"].([]interface{})
	require.True(ok, "steps should be an array")
	require.Len(steps, 2)

	// Verify first step
	bootstrapStep, ok := steps[0].(map[string]interface{})
	require.True(ok)
	require.Equal("Bootstrap", bootstrapStep["name"])
	require.Equal("success", bootstrapStep["status"])
	require.Equal("2024-01-01T00:00:00Z", bootstrapStep["startTime"])
	require.Equal("2024-01-01T00:01:00Z", bootstrapStep["endTime"])

	events, ok := bootstrapStep["events"].([]interface{})
	require.True(ok)
	require.Len(events, 2)

	// Verify second step has displayName
	networkStep, ok := steps[1].(map[string]interface{})
	require.True(ok)
	require.Equal("Network", networkStep["name"])
	require.Equal("Network Setup", networkStep["displayName"])
	require.Equal("in-progress", networkStep["status"])
}

func TestPlanDAGJSONOmitsEmptyWorkflowSteps(t *testing.T) {
	require := require.New(t)

	plan := &PlanDAG{
		Nodes:    map[string]PlanDAGNode{},
		Edges:    []PlanDAGEdge{},
		Levels:   [][]string{},
		HasCycle: false,
	}

	jsonBytes, err := json.Marshal(plan)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.NotContains(decoded, "workflowStepsByKey", "empty workflowStepsByKey should be omitted")
}

func TestWorkflowStepInfoJSONDepTimelines(t *testing.T) {
	require := require.New(t)

	step := WorkflowStepInfo{
		Name:        "Bootstrap",
		DisplayName: "Waiting for dependencies",
		Status:      "success",
		StartTime:   "2024-01-01T00:00:00Z",
		EndTime:     "2024-01-01T00:02:00Z",
		DepTimelines: []depTimeline{
			{Name: "network", Status: "completed", FinishedAt: "2024-01-01T00:01:00Z"},
			{Name: "storage", Status: "running", FinishedAt: ""},
		},
	}

	jsonBytes, err := json.Marshal(step)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("Waiting for dependencies", decoded["displayName"])

	deps, ok := decoded["depTimelines"].([]interface{})
	require.True(ok)
	require.Len(deps, 2)

	dep0, ok := deps[0].(map[string]interface{})
	require.True(ok)
	require.Equal("network", dep0["name"])
	require.Equal("completed", dep0["status"])
	require.Equal("2024-01-01T00:01:00Z", dep0["finishedAt"])

	dep1, ok := deps[1].(map[string]interface{})
	require.True(ok)
	require.Equal("storage", dep1["name"])
	require.Equal("running", dep1["status"])
	require.NotContains(dep1, "finishedAt", "empty finishedAt should be omitted")
}

func TestDebugDataJSONRoundTrip(t *testing.T) {
	require := require.New(t)

	original := DebugData{
		InstanceID:    "inst-123",
		ServiceID:     "svc-456",
		EnvironmentID: "env-789",
		Token:         "secret-token",
		PlanDAG: &PlanDAG{
			Nodes: map[string]PlanDAGNode{
				"r-1": {ID: "r-1", Key: "db", Name: "Database", Type: "terraform"},
			},
			Edges:    []PlanDAGEdge{{From: "r-1", To: "r-2"}},
			Levels:   [][]string{{"r-1"}, {"r-2"}},
			HasCycle: false,
			ProgressByKey: map[string]ResourceProgress{
				"db": {Percent: 50, Status: "running", CompletedSteps: 2, TotalSteps: 4},
			},
			BreakpointByKey: map[string]string{
				"db": "hit",
			},
			WorkflowStepsByKey: map[string]*ResourceWorkflowSteps{
				"db": {
					Steps: []WorkflowStepInfo{
						{
							Name:   "Bootstrap",
							Status: "success",
							Events: []dataaccess.DebugEvent{
								{EventTime: "2024-01-01T00:00:00Z", EventType: "completed", Message: "Done"},
							},
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(original, "", "  ")
	require.NoError(err)

	var roundTripped DebugData
	err = json.Unmarshal(jsonBytes, &roundTripped)
	require.NoError(err)

	require.Equal(original.InstanceID, roundTripped.InstanceID)
	require.Equal(original.ServiceID, roundTripped.ServiceID)
	require.Equal(original.EnvironmentID, roundTripped.EnvironmentID)
	require.Empty(roundTripped.Token, "token should not be in JSON")

	require.NotNil(roundTripped.PlanDAG)
	require.Len(roundTripped.PlanDAG.Nodes, 1)
	require.Equal("db", roundTripped.PlanDAG.Nodes["r-1"].Key)
	require.Len(roundTripped.PlanDAG.Edges, 1)
	require.False(roundTripped.PlanDAG.HasCycle)

	require.Contains(roundTripped.PlanDAG.ProgressByKey, "db")
	require.Equal(50, roundTripped.PlanDAG.ProgressByKey["db"].Percent)
	require.Equal("running", roundTripped.PlanDAG.ProgressByKey["db"].Status)

	require.Contains(roundTripped.PlanDAG.BreakpointByKey, "db")
	require.Equal("hit", roundTripped.PlanDAG.BreakpointByKey["db"])

	require.Contains(roundTripped.PlanDAG.WorkflowStepsByKey, "db")
	require.Len(roundTripped.PlanDAG.WorkflowStepsByKey["db"].Steps, 1)
	require.Equal("Bootstrap", roundTripped.PlanDAG.WorkflowStepsByKey["db"].Steps[0].Name)
	require.Equal("success", roundTripped.PlanDAG.WorkflowStepsByKey["db"].Steps[0].Status)
	require.Len(roundTripped.PlanDAG.WorkflowStepsByKey["db"].Steps[0].Events, 1)
}

func TestDebugCommandStructure(t *testing.T) {
	require := require.New(t)

	require.Equal("debug [instance-id]", debugCmd.Use)
	require.Contains(debugCmd.Long, "--output=json")

	flag := debugCmd.Flags().Lookup("output")
	require.NotNil(flag)
	require.Equal("interactive", flag.DefValue)
}
