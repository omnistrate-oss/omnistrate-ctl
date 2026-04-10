package instance

import (
	"encoding/json"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
		ResourceDebugInfo: map[string]*ResourceDebugInfo{
			"db": {
				ResourceID:   "r-1",
				ResourceKey:  "db",
				ResourceType: "terraform",
				TerraformProgress: &TerraformProgressData{
					TerraformName:  "tf-db",
					Status:         "completed",
					TotalResources: 3,
					Resources: []TerraformResourceDetail{
						{Address: "aws_instance.main", Type: "aws_instance", Name: "main", State: "ready"},
					},
				},
				TerraformHistory: []TerraformHistoryEntry{
					{Operation: "apply", Status: "completed", OperationID: "op-1"},
				},
				TerraformFiles: map[string]string{"main.tf": "resource \"aws_instance\" {}"},
				TerraformLogs:  map[string]string{"log/op-1-apply.log": "Apply complete!"},
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

	// Verify ResourceDebugInfo round-trips
	require.Contains(roundTripped.ResourceDebugInfo, "db")
	dbInfo := roundTripped.ResourceDebugInfo["db"]
	require.Equal("r-1", dbInfo.ResourceID)
	require.Equal("db", dbInfo.ResourceKey)
	require.Equal("terraform", dbInfo.ResourceType)

	require.NotNil(dbInfo.TerraformProgress)
	require.Equal("tf-db", dbInfo.TerraformProgress.TerraformName)
	require.Equal("completed", dbInfo.TerraformProgress.Status)
	require.Equal(3, dbInfo.TerraformProgress.TotalResources)
	require.Len(dbInfo.TerraformProgress.Resources, 1)
	require.Equal("aws_instance.main", dbInfo.TerraformProgress.Resources[0].Address)

	require.Len(dbInfo.TerraformHistory, 1)
	require.Equal("apply", dbInfo.TerraformHistory[0].Operation)

	require.Contains(dbInfo.TerraformFiles, "main.tf")
	require.Contains(dbInfo.TerraformLogs, "log/op-1-apply.log")
}

func TestDebugCommandStructure(t *testing.T) {
	require := require.New(t)

	require.Equal("debug [instance-id]", debugCmd.Use)
	require.Contains(debugCmd.Long, "--output=json")

	flag := debugCmd.Flags().Lookup("output")
	require.NotNil(flag)
	require.Equal("interactive", flag.DefValue)
}

func TestResourceDebugInfoHelmJSON(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "r-helm-1",
		ResourceKey:  "web-server",
		ResourceType: "helm",
		Helm: &HelmData{
			ChartRepoName: "bitnami",
			ChartRepoURL:  "https://charts.bitnami.com/bitnami",
			ChartVersion:  "1.0.0",
			ChartValues:   map[string]interface{}{"replicaCount": float64(3)},
			InstallLog:    "Install complete",
			Namespace:     "default",
			ReleaseName:   "web-server",
		},
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("r-helm-1", decoded["resourceId"])
	require.Equal("web-server", decoded["resourceKey"])
	require.Equal("helm", decoded["resourceType"])

	helm, ok := decoded["helm"].(map[string]interface{})
	require.True(ok, "helm should be a map")
	require.Equal("bitnami", helm["chartRepoName"])
	require.Equal("https://charts.bitnami.com/bitnami", helm["chartRepoURL"])
	require.Equal("1.0.0", helm["chartVersion"])
	require.Equal("Install complete", helm["installLog"])
	require.Equal("default", helm["namespace"])
	require.Equal("web-server", helm["releaseName"])

	// Verify terraform fields are omitted
	require.NotContains(decoded, "terraformProgress")
	require.NotContains(decoded, "terraformHistory")
	require.NotContains(decoded, "terraformFiles")
	require.NotContains(decoded, "terraformLogs")
}

func TestResourceDebugInfoTerraformJSON(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "tf-r-1",
		ResourceKey:  "database",
		ResourceType: "terraform",
		TerraformProgress: &TerraformProgressData{
			TerraformName:   "tf-database",
			InstanceID:      "inst-1",
			ResourceID:      "tf-r-1",
			Status:          "in-progress",
			TotalResources:  5,
			FailedResources: 1,
			Resources: []TerraformResourceDetail{
				{Address: "aws_rds_instance.main", Type: "aws_rds_instance", Name: "main", State: "ready"},
				{Address: "aws_security_group.db", Type: "aws_security_group", Name: "db", State: "in-progress"},
			},
		},
		TerraformHistory: []TerraformHistoryEntry{
			{Operation: "plan", Status: "completed", OperationID: "op-1", StartedAt: "2024-01-01T00:00:00Z"},
			{Operation: "apply", Status: "failed", OperationID: "op-2", Error: "timeout"},
		},
		TerraformFiles: map[string]string{
			"main.tf":      "resource \"aws_rds_instance\" \"main\" {}",
			"variables.tf": "variable \"instance_class\" {}",
		},
		TerraformLogs: map[string]string{
			"log/op-2-apply.log": "Error: timeout waiting for resource",
		},
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded ResourceDebugInfo
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("tf-r-1", decoded.ResourceID)
	require.Equal("database", decoded.ResourceKey)
	require.Equal("terraform", decoded.ResourceType)
	require.Nil(decoded.Helm, "helm should be nil for terraform resource")

	require.NotNil(decoded.TerraformProgress)
	require.Equal("in-progress", decoded.TerraformProgress.Status)
	require.Equal(5, decoded.TerraformProgress.TotalResources)
	require.Equal(1, decoded.TerraformProgress.FailedResources)
	require.Len(decoded.TerraformProgress.Resources, 2)

	require.Len(decoded.TerraformHistory, 2)
	require.Equal("apply", decoded.TerraformHistory[1].Operation)
	require.Equal("timeout", decoded.TerraformHistory[1].Error)

	require.Len(decoded.TerraformFiles, 2)
	require.Contains(decoded.TerraformFiles, "main.tf")

	require.Len(decoded.TerraformLogs, 1)
	require.Contains(decoded.TerraformLogs, "log/op-2-apply.log")
}

func TestResourceDebugInfoOmitsEmptyFields(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "r-1",
		ResourceKey:  "cache",
		ResourceType: "other",
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("r-1", decoded["resourceId"])
	require.NotContains(decoded, "helm")
	require.NotContains(decoded, "terraformProgress")
	require.NotContains(decoded, "terraformHistory")
	require.NotContains(decoded, "terraformFiles")
	require.NotContains(decoded, "terraformLogs")
}

func TestDebugDataJSONOmitsEmptyResourceDebugInfo(t *testing.T) {
	require := require.New(t)

	data := DebugData{
		InstanceID: "inst-1",
	}

	jsonBytes, err := json.Marshal(data)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.NotContains(decoded, "resourceDebugInfo", "empty resourceDebugInfo should be omitted")
}

func TestExtractTerraformProgressFromIndex(t *testing.T) {
	require := require.New(t)

	// Create a mock configmap index with state and progress configmaps
	index := &terraformConfigMapIndex{
		instanceID:     "inst-1",
		instanceSuffix: "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{
			"tf-r-abc123": {
				Data: map[string]string{
					"history": `[{"operation":"apply","status":"completed","operationId":"op-1","startedAt":"2024-01-01T00:00:00Z","completedAt":"2024-01-01T00:05:00Z"}]`,
				},
			},
		},
		progress: []*corev1.ConfigMap{
			{
				Data: map[string]string{
					"progress": `{"terraformName":"tf-db","instanceID":"inst-1","resourceID":"r-abc123","status":"completed","startedAt":"2024-01-01T00:00:00Z","totalResources":2,"resources":[{"address":"aws_instance.main","state":"ready"}]}`,
				},
			},
		},
		planPreviewByResource: make(map[string][]planPreviewEntry),
	}

	progress, history := extractTerraformProgressFromIndex(index, "inst-1", "r-abc123")

	require.NotNil(progress)
	require.Equal("completed", progress.Status)
	require.Equal(2, progress.TotalResources)
	require.Len(progress.Resources, 1)
	require.Equal("aws_instance.main", progress.Resources[0].Address)

	require.Len(history, 1)
	require.Equal("apply", history[0].Operation)
	require.Equal("completed", history[0].Status)
}

func TestExtractTerraformProgressFromIndexNilIndex(t *testing.T) {
	progress, history := extractTerraformProgressFromIndex(nil, "inst-1", "r-1")
	require.Nil(t, progress)
	require.Nil(t, history)
}

func TestExtractTerraformProgressFromIndexNoMatch(t *testing.T) {
	index := &terraformConfigMapIndex{
		instanceID:            "inst-1",
		instanceSuffix:        "inst-1",
		stateByResource:       map[string]*corev1.ConfigMap{},
		progress:              []*corev1.ConfigMap{},
		planPreviewByResource: make(map[string][]planPreviewEntry),
	}

	progress, history := extractTerraformProgressFromIndex(index, "inst-1", "r-nonexistent")
	require.Nil(t, progress)
	require.Nil(t, history)
}

func TestExtractTerraformStateDataProgressOnly(t *testing.T) {
	require := require.New(t)

	// Resource has progress data but no history, no plan previews.
	// Should still return a result with progress (best-effort).
	index := &terraformConfigMapIndex{
		instanceID:      "inst-1",
		instanceSuffix:  "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{},
		progress: []*corev1.ConfigMap{
			{
				Data: map[string]string{
					"progress": `{"terraformName":"tf-test","instanceID":"inst-1","resourceID":"r-abc123","status":"running","startedAt":"2026-01-01T00:00:00Z","totalResources":5,"inProgressResources":2}`,
				},
			},
		},
		planPreviewByResource: make(map[string][]planPreviewEntry),
	}

	stateData := extractTerraformStateData(index, "inst-1", "r-abc123")
	require.NotNil(stateData, "should return data when only progress exists")
	require.NotNil(stateData.Progress)
	require.Equal("tf-test", stateData.Progress.TerraformName)
	require.Equal("running", stateData.Progress.Status)
	require.Equal(5, stateData.Progress.TotalResources)
	require.Empty(stateData.History)
	require.Empty(stateData.PlanPreviews)
}

func TestExtractTerraformStateDataWithDedicatedPlanPreviewCMs(t *testing.T) {
	require := require.New(t)

	index := &terraformConfigMapIndex{
		instanceID:     "inst-1",
		instanceSuffix: "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{
			"tf-r-abc123": {
				Data: map[string]string{
					"history": `[{"operation":"apply","status":"completed","operationId":"op-1"}]`,
				},
			},
		},
		progress: []*corev1.ConfigMap{},
		planPreviewByResource: map[string][]planPreviewEntry{
			"tf-r-abc123": {
				{
					cm: &corev1.ConfigMap{
						Data: map[string]string{
							"plan-preview": `{"format_version":"1.2","planned_values":{}}`,
						},
					},
					opSuffix: "op-1-hash123",
				},
				{
					cm: &corev1.ConfigMap{
						Data: map[string]string{
							"plan-preview-error": "Error: timeout waiting",
						},
					},
					opSuffix: "op-2-hash456",
				},
			},
		},
	}

	stateData := extractTerraformStateData(index, "inst-1", "r-abc123")
	require.NotNil(stateData)
	require.Len(stateData.History, 1)
	require.Len(stateData.PlanPreviews, 1)
	require.Equal(`{"format_version":"1.2","planned_values":{}}`, stateData.PlanPreviews["op-1-hash123"])
	require.Len(stateData.PreviewErrors, 1)
	require.Equal("Error: timeout waiting", stateData.PreviewErrors["op-2-hash456"])
}

func TestExtractTerraformStateDataPlanPreviewCMsOnlyNoStateCM(t *testing.T) {
	require := require.New(t)

	// No state configmap, but dedicated plan preview CMs exist
	index := &terraformConfigMapIndex{
		instanceID:      "inst-1",
		instanceSuffix:  "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{},
		progress:        []*corev1.ConfigMap{},
		planPreviewByResource: map[string][]planPreviewEntry{
			"tf-r-abc123": {
				{
					cm: &corev1.ConfigMap{
						Data: map[string]string{
							"plan-preview": `{"planned_values":{"outputs":{}}}`,
						},
					},
					opSuffix: "myop-hash789",
				},
			},
		},
	}

	stateData := extractTerraformStateData(index, "inst-1", "r-abc123")
	require.NotNil(stateData, "should return state data when plan preview CMs exist even without state CM")
	require.Empty(stateData.History)
	require.Len(stateData.PlanPreviews, 1)
	require.Equal(`{"planned_values":{"outputs":{}}}`, stateData.PlanPreviews["myop-hash789"])
}

func TestExtractTerraformStateDataDedicatedCMOnly(t *testing.T) {
	require := require.New(t)

	// When dedicated tf-plan-* CMs exist, state CM plan preview keys are ignored.
	index := &terraformConfigMapIndex{
		instanceID:     "inst-1",
		instanceSuffix: "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{
			"tf-r-abc123": {
				Data: map[string]string{
					"history":             `[{"operation":"apply","status":"completed","operationId":"op-1"}]`,
					"shared-plan-preview": `{"from":"state-cm"}`,
				},
			},
		},
		progress: []*corev1.ConfigMap{},
		planPreviewByResource: map[string][]planPreviewEntry{
			"tf-r-abc123": {
				{
					cm: &corev1.ConfigMap{
						Data: map[string]string{
							"plan-preview": `{"from":"dedicated-cm"}`,
						},
					},
					opSuffix: "shared",
				},
				{
					cm: &corev1.ConfigMap{
						Data: map[string]string{
							"plan-preview": `{"from":"dedicated-cm-2"}`,
						},
					},
					opSuffix: "unique-op",
				},
			},
		},
	}

	stateData := extractTerraformStateData(index, "inst-1", "r-abc123")
	require.NotNil(stateData)

	// Only dedicated CM previews — state CM is not consulted when dedicated CMs have data
	require.Len(stateData.PlanPreviews, 2)
	require.Equal(`{"from":"dedicated-cm"}`, stateData.PlanPreviews["shared"])
	require.Equal(`{"from":"dedicated-cm-2"}`, stateData.PlanPreviews["unique-op"])
}

func TestExtractTerraformStateDataStateCMFallback(t *testing.T) {
	require := require.New(t)

	// No dedicated CMs exist → fall back to state CM for plan previews.
	index := &terraformConfigMapIndex{
		instanceID:     "inst-1",
		instanceSuffix: "inst-1",
		stateByResource: map[string]*corev1.ConfigMap{
			"tf-r-abc123": {
				Data: map[string]string{
					"history":                 `[{"operation":"apply","status":"completed","operationId":"op-1"}]`,
					"state-only-plan-preview": `{"from":"state-cm-only"}`,
				},
			},
		},
		progress:              []*corev1.ConfigMap{},
		planPreviewByResource: map[string][]planPreviewEntry{},
	}

	stateData := extractTerraformStateData(index, "inst-1", "r-abc123")
	require.NotNil(stateData)

	// History is available from state CM
	require.Len(stateData.History, 1)
	// Plan preview falls back to state CM when no dedicated CMs found
	require.Len(stateData.PlanPreviews, 1)
	require.Equal(`{"from":"state-cm-only"}`, stateData.PlanPreviews["state-only"])
}

func TestResourceDebugInfoPlanPreviewJSON(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "tf-r-1",
		ResourceKey:  "database",
		ResourceType: "terraform",
		TerraformPlanPreview: map[string]string{
			"op-1": `{"format_version":"1.2","planned_values":{"outputs":{"db_endpoints":{"sensitive":false}}}}`,
		},
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("tf-r-1", decoded["resourceId"])
	require.Contains(decoded, "terraformPlanPreview")
	require.NotContains(decoded, "terraformPlanPreviewError")
}

func TestResourceDebugInfoPlanPreviewErrorJSON(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "tf-r-1",
		ResourceKey:  "database",
		ResourceType: "terraform",
		TerraformPlanPreviewError: map[string]string{
			"op-1": "Error: Failed to refresh state",
		},
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.Equal("tf-r-1", decoded["resourceId"])
	require.NotContains(decoded, "terraformPlanPreview")
	require.Contains(decoded, "terraformPlanPreviewError")

	previewErr, ok := decoded["terraformPlanPreviewError"].(map[string]interface{})
	require.True(ok)
	require.Equal("Error: Failed to refresh state", previewErr["op-1"])
}

func TestResourceDebugInfoOmitsEmptyPlanPreview(t *testing.T) {
	require := require.New(t)

	info := ResourceDebugInfo{
		ResourceID:   "r-1",
		ResourceKey:  "cache",
		ResourceType: "other",
	}

	jsonBytes, err := json.Marshal(info)
	require.NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(err)

	require.NotContains(decoded, "terraformPlanPreview")
	require.NotContains(decoded, "terraformPlanPreviewError")
}

func TestResourceDebugInfoHasDataWithPlanPreview(t *testing.T) {
	require := require.New(t)

	// Plan preview makes hasData() return true
	info := ResourceDebugInfo{
		ResourceID:   "r-1",
		ResourceKey:  "db",
		ResourceType: "terraform",
		TerraformPlanPreview: map[string]string{
			"op-1": `{"planned_values":{}}`,
		},
	}
	require.True(info.hasData())

	// Plan preview error also makes hasData() return true
	info2 := ResourceDebugInfo{
		ResourceID:   "r-1",
		ResourceKey:  "db",
		ResourceType: "terraform",
		TerraformPlanPreviewError: map[string]string{
			"op-1": "Error: timeout",
		},
	}
	require.True(info2.hasData())

	// Empty info still returns false
	info3 := ResourceDebugInfo{
		ResourceID:   "r-1",
		ResourceKey:  "db",
		ResourceType: "terraform",
	}
	require.False(info3.hasData())
}

func TestFindAllPlanPreviewsBasic(t *testing.T) {
	require := require.New(t)

	files := map[string]string{
		"op-1-plan-preview":       `{"format_version":"1.0","planned_values":{}}`,
		"op-2-plan-preview":       `{"format_version":"1.2","planned_values":{"outputs":{}}}`,
		"op-1-output.log":         `{"db_endpoints":{"value":"endpoint:3306"}}`,
		"history":                 `[{"operation":"diff","operationId":"op-1"},{"operation":"apply","operationId":"op-1"}]`,
		"op-3-plan-preview-error": "Error: Failed to refresh state",
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Len(previews, 2)
	require.Equal(`{"format_version":"1.0","planned_values":{}}`, previews["op-1"])
	require.Equal(`{"format_version":"1.2","planned_values":{"outputs":{}}}`, previews["op-2"])
	require.Len(previewErrors, 1)
	require.Equal("Error: Failed to refresh state", previewErrors["op-3"])
}

func TestFindAllPlanPreviewsErrorOnly(t *testing.T) {
	require := require.New(t)

	files := map[string]string{
		"op-1-plan-preview":       `{"format_version":"1.0"}`,
		"op-2-plan-preview-error": "Error: Failed to refresh state",
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Len(previews, 1)
	require.Equal(`{"format_version":"1.0"}`, previews["op-1"])
	require.Len(previewErrors, 1)
	require.Equal("Error: Failed to refresh state", previewErrors["op-2"])
}

func TestFindAllPlanPreviewsFallbackNoHistory(t *testing.T) {
	require := require.New(t)

	files := map[string]string{
		"abc123-plan-preview": `{"planned_values":{}}`,
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Len(previews, 1)
	require.Equal(`{"planned_values":{}}`, previews["abc123"])
	require.Empty(previewErrors)
}

func TestFindAllPlanPreviewsFallbackErrorOnly(t *testing.T) {
	require := require.New(t)

	files := map[string]string{
		"abc123-plan-preview-error": "Error: something failed",
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Empty(previews)
	require.Len(previewErrors, 1)
	require.Equal("Error: something failed", previewErrors["abc123"])
}

func TestFindAllPlanPreviewsEmpty(t *testing.T) {
	require := require.New(t)

	// No plan preview data at all
	files := map[string]string{
		"op-1-output.log": `{"db_endpoints":{}}`,
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Empty(previews)
	require.Empty(previewErrors)
}

func TestFindAllPlanPreviewsNilFiles(t *testing.T) {
	previews, previewErrors := findAllPlanPreviews(nil)
	require.Empty(t, previews)
	require.Empty(t, previewErrors)
}

func TestFindAllPlanPreviewsSkipsEmptyValues(t *testing.T) {
	require := require.New(t)

	files := map[string]string{
		"op-1-plan-preview":       "",
		"op-2-plan-preview-error": "",
		"op-3-plan-preview":       `{"value":"real"}`,
	}

	previews, previewErrors := findAllPlanPreviews(files)
	require.Len(previews, 1)
	require.Equal(`{"value":"real"}`, previews["op-3"])
	require.Empty(previewErrors)
}

func TestDebugDataJSONRoundTripWithPlanPreview(t *testing.T) {
	require := require.New(t)

	original := DebugData{
		InstanceID:    "inst-123",
		ServiceID:     "svc-456",
		EnvironmentID: "env-789",
		ResourceDebugInfo: map[string]*ResourceDebugInfo{
			"db": {
				ResourceID:   "r-1",
				ResourceKey:  "db",
				ResourceType: "terraform",
				TerraformPlanPreview: map[string]string{
					"op-1": `{"format_version":"1.2","planned_values":{"outputs":{"db_endpoints":{"sensitive":false}}}}`,
				},
			},
			"cache": {
				ResourceID:   "r-2",
				ResourceKey:  "cache",
				ResourceType: "terraform",
				TerraformPlanPreviewError: map[string]string{
					"op-2": "Error: Failed to refresh state for resource",
				},
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(original, "", "  ")
	require.NoError(err)

	var roundTripped DebugData
	err = json.Unmarshal(jsonBytes, &roundTripped)
	require.NoError(err)

	require.Contains(roundTripped.ResourceDebugInfo, "db")
	dbInfo := roundTripped.ResourceDebugInfo["db"]
	require.Len(dbInfo.TerraformPlanPreview, 1)
	require.Equal(`{"format_version":"1.2","planned_values":{"outputs":{"db_endpoints":{"sensitive":false}}}}`, dbInfo.TerraformPlanPreview["op-1"])
	require.Empty(dbInfo.TerraformPlanPreviewError)

	require.Contains(roundTripped.ResourceDebugInfo, "cache")
	cacheInfo := roundTripped.ResourceDebugInfo["cache"]
	require.Empty(cacheInfo.TerraformPlanPreview)
	require.Len(cacheInfo.TerraformPlanPreviewError, 1)
	require.Equal("Error: Failed to refresh state for resource", cacheInfo.TerraformPlanPreviewError["op-2"])
}
