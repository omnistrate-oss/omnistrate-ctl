package instance

import "encoding/json"

type HelmData struct {
	ChartRepoName string                 `json:"chartRepoName"`
	ChartRepoURL  string                 `json:"chartRepoURL"`
	ChartVersion  string                 `json:"chartVersion"`
	ChartValues   map[string]interface{} `json:"chartValues"`
	InstallLog    string                 `json:"installLog"`
	Namespace     string                 `json:"namespace"`
	ReleaseName   string                 `json:"releaseName"`
}

type TerraformData struct {
	Files map[string]string `json:"files"`
	Logs  map[string]string `json:"logs"`
}

// ResourceDebugInfo holds all debug information for a specific resource in the plan DAG.
type ResourceDebugInfo struct {
	ResourceID   string `json:"resourceId"`
	ResourceKey  string `json:"resourceKey"`
	ResourceType string `json:"resourceType"`

	// Helm-specific data (populated for helm resources)
	Helm *HelmData `json:"helm,omitempty"`

	// Terraform-specific data (populated for terraform resources)
	TerraformProgress         *TerraformProgressData  `json:"terraformProgress,omitempty"`
	TerraformHistory          []TerraformHistoryEntry `json:"terraformHistory,omitempty"`
	TerraformFiles            map[string]string       `json:"terraformFiles,omitempty"`
	TerraformLogs             map[string]string       `json:"terraformLogs,omitempty"`
	TerraformPlanPreview      string                  `json:"terraformPlanPreview,omitempty"`
	TerraformPlanPreviewError string                  `json:"terraformPlanPreviewError,omitempty"`
}

// hasData returns true if any debug data has been populated for this resource.
func (r *ResourceDebugInfo) hasData() bool {
	return r.Helm != nil || r.TerraformProgress != nil ||
		len(r.TerraformHistory) > 0 || len(r.TerraformFiles) > 0 || len(r.TerraformLogs) > 0 ||
		r.TerraformPlanPreview != "" || r.TerraformPlanPreviewError != ""
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

	if chartValuesStr, ok := debugData["chartValues"].(string); ok {
		var chartValues map[string]interface{}
		if err := json.Unmarshal([]byte(chartValuesStr), &chartValues); err == nil {
			helmData.ChartValues = chartValues
		}
	}

	if installLog, ok := debugData["log/install.log"].(string); ok {
		helmData.InstallLog = installLog
	}

	return helmData
}
