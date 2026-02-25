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
