package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

type HelmData struct {
	ChartRepoName string                 `json:"chartRepoName"`
	ChartRepoURL  string                 `json:"chartRepoURL"`
	ChartVersion  string                 `json:"chartVersion"`
	ChartValues   map[string]interface{} `json:"chartValues"`
	InstallLog    string                 `json:"installLog"`
	Namespace     string                 `json:"namespace"`
	ReleaseName   string                 `json:"releaseName"`
	InputParams   []OperatorInputParam   `json:"inputParams,omitempty"`
	OutputParams  []OperatorOutputParam  `json:"outputParams,omitempty"`
}

type TerraformData struct {
	Files map[string]string `json:"files"`
	Logs  map[string]string `json:"logs"`
}

// OperatorInputParam describes a single input parameter for an operator resource.
// These are fetched from the ListInputParameter V1 API.
type OperatorInputParam struct {
	Key           string `json:"key"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	Modifiable    bool   `json:"modifiable"`
	DefaultValue  string `json:"defaultValue,omitempty"`
	ResolvedValue string `json:"resolvedValue,omitempty"`
}

// OperatorOutputParam describes a single output parameter for a resource.
// These are API parameters with export: true, fetched from the ListOutputParameter V1 API.
type OperatorOutputParam struct {
	Key           string `json:"key"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Value         string `json:"value,omitempty"`
	ValueRef      string `json:"valueRef,omitempty"`
	Type          string `json:"type,omitempty"`
	ResolvedValue string `json:"resolvedValue,omitempty"`
}

// OperatorCRDOutputParam describes a single output parameter from operatorCRDConfiguration.
type OperatorCRDOutputParam struct {
	Key           string `json:"key"`
	Value         string `json:"value"`
	ResolvedValue string `json:"resolvedValue,omitempty"`
}

// OperatorData holds debug information specific to operator-type resources.
type OperatorData struct {
	InputParams     []OperatorInputParam     `json:"inputParams,omitempty"`
	OutputParams    []OperatorOutputParam    `json:"outputParams,omitempty"`
	CRDOutputParams []OperatorCRDOutputParam `json:"crdOutputParams,omitempty"`
}

// ComposeData holds debug information specific to compose-type resources.
type ComposeData struct {
	InputParams  []OperatorInputParam  `json:"inputParams,omitempty"`
	OutputParams []OperatorOutputParam `json:"outputParams,omitempty"`
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
	TerraformPlanPreview      map[string]string       `json:"terraformPlanPreview,omitempty"`
	TerraformPlanPreviewError map[string]string       `json:"terraformPlanPreviewError,omitempty"`

	// Operator-specific data (populated for operator resources)
	Operator *OperatorData `json:"operator,omitempty"`

	// Compose-specific data (populated for compose resources)
	Compose *ComposeData `json:"compose,omitempty"`
}

// hasData returns true if any debug data has been populated for this resource.
func (r *ResourceDebugInfo) hasData() bool {
	return r.Helm != nil || r.Operator != nil || r.Compose != nil || r.TerraformProgress != nil ||
		len(r.TerraformHistory) > 0 || len(r.TerraformFiles) > 0 || len(r.TerraformLogs) > 0 ||
		len(r.TerraformPlanPreview) > 0 || len(r.TerraformPlanPreviewError) > 0
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

// fetchInputParams fetches input parameters from the ListInputParameter V1 API
// and converts them to OperatorInputParam structs. If inputParams is provided,
// resolved values are looked up by key and populated. Used by both helm and operator TUIs.
func fetchInputParams(ctx context.Context, token, serviceID, resourceID, productTierID, tierVersion string, inputParams map[string]interface{}) ([]OperatorInputParam, error) {
	result, err := dataaccess.ListInputParameters(ctx, token, serviceID, resourceID, productTierID, tierVersion)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	var params []OperatorInputParam
	for _, ip := range result.InputParameters {
		param := OperatorInputParam{
			Key:         ip.Key,
			DisplayName: ip.Name,
			Description: ip.Description,
			Type:        ip.Type,
			Required:    ip.Required,
			Modifiable:  ip.Modifiable,
		}
		if ip.DefaultValue != nil {
			param.DefaultValue = *ip.DefaultValue
		}
		// Resolve value from instance input_params
		if inputParams != nil {
			if v, ok := inputParams[ip.Key]; ok {
				param.ResolvedValue = fmt.Sprintf("%v", v)
			}
		}
		params = append(params, param)
	}
	return params, nil
}

// fetchOutputParams fetches output parameters from the ListOutputParameter V1 API
// and converts them to OperatorOutputParam structs. If resultParams is provided,
// resolved values are looked up by key and populated. Used by both helm and operator TUIs.
func fetchOutputParams(ctx context.Context, token, serviceID, resourceID, productTierID, tierVersion string, resultParams map[string]interface{}) ([]OperatorOutputParam, error) {
	result, err := dataaccess.ListOutputParameters(ctx, token, serviceID, resourceID, productTierID, tierVersion)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	var params []OperatorOutputParam
	for _, op := range result.OutputParameters {
		param := OperatorOutputParam{
			Key:         op.Key,
			DisplayName: op.Name,
			Description: op.Description,
		}
		if op.Value != nil {
			param.Value = *op.Value
		}
		if op.ValueRef != nil {
			param.ValueRef = *op.ValueRef
		}
		if op.ValueType != nil {
			param.Type = *op.ValueType
		}
		// Resolve value from instance result_params
		if resultParams != nil {
			if v, ok := resultParams[op.Key]; ok {
				param.ResolvedValue = fmt.Sprintf("%v", v)
			}
		}
		params = append(params, param)
	}
	return params, nil
}
