package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	corev1 "k8s.io/api/core/v1"
)

// TerraformProgressData holds the parsed progress from a terraform-progress configmap
type TerraformProgressData struct {
	TerraformName       string                    `json:"terraformName"`
	InstanceID          string                    `json:"instanceID"`
	ResourceID          string                    `json:"resourceID"`
	ResourceVersion     string                    `json:"resourceVersion"`
	OperationID         string                    `json:"operationID"`
	Status              string                    `json:"status"`
	StartedAt           string                    `json:"startedAt"`
	CompletedAt         string                    `json:"completedAt"`
	TotalResources      int                       `json:"totalResources"`
	InProgressResources int                       `json:"inProgressResources"`
	FailedResources     int                       `json:"failedResources"`
	Resources           []TerraformResourceDetail `json:"resources"`
	PlannedResources    []string                  `json:"plannedResources"`
}

// TerraformResourceDetail is a single resource in the terraform progress
type TerraformResourceDetail struct {
	Address  string `json:"address"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Provider string `json:"provider"`
	State    string `json:"state"`
}

// TerraformHistoryEntry is a single entry from the tf-state configmap history
type TerraformHistoryEntry struct {
	Operation   string `json:"operation"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	OperationID string `json:"operationId"`
	Error       string `json:"error,omitempty"`
}

// fetchTerraformProgress fetches and parses terraform progress for a given resource node
func fetchTerraformProgress(ctx context.Context, token string, instanceData *openapiclientfleet.ResourceInstance, instanceID, resourceID string) (*TerraformProgressData, []TerraformHistoryEntry, *k8sConnections, error) {
	index, conn, err := loadTerraformConfigMapIndexForInstance(ctx, token, instanceData, instanceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load terraform configmap index: %w", err)
	}

	progress, history := extractTerraformProgressFromIndex(index, instanceID, resourceID)
	return progress, history, conn, nil
}

// TerraformStateData holds all data extracted from a tf-state configmap:
// progress, history, and plan previews. This avoids needing a second configmap
// lookup via terraformDataForResource which can fail in some environments.
type TerraformStateData struct {
	Progress      *TerraformProgressData
	History       []TerraformHistoryEntry
	PlanPreviews  map[string]string // plan preview JSON keyed by operation ID
	PreviewErrors map[string]string // plan preview errors keyed by operation ID
}

// extractTerraformProgressFromIndex extracts terraform progress, history, and plan previews
// for a given resource from a pre-loaded configmap index, without making additional k8s calls.
// Plan previews are extracted directly from the same state configmap that provides history,
// ensuring they are always found when the configmap is accessible.
func extractTerraformProgressFromIndex(index *terraformConfigMapIndex, instanceID, resourceID string) (*TerraformProgressData, []TerraformHistoryEntry) {
	result := extractTerraformStateData(index, instanceID, resourceID)
	if result == nil {
		return nil, nil
	}
	return result.Progress, result.History
}

// extractTerraformStateData extracts all state data (progress, history, plan previews)
// from the tf-state and progress configmaps for a given resource.
// Plan previews are also sourced from dedicated tf-plan-* ConfigMaps.
func extractTerraformStateData(index *terraformConfigMapIndex, instanceID, resourceID string) *TerraformStateData {
	if index == nil {
		return nil
	}

	// Find the tf-state configmap for this resource, trying multiple key formats
	var stateConfigMap *corev1.ConfigMap
	for _, key := range resourceConfigMapKeys(resourceID) {
		if cm, ok := index.stateByResource[key]; ok {
			stateConfigMap = cm
			break
		}
	}

	var history []TerraformHistoryEntry
	var planPreviews, previewErrors map[string]string

	if stateConfigMap != nil {
		// Parse history from the configmap
		historyJSON, ok := stateConfigMap.Data["history"]
		if ok {
			if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
				// Surface history parse problems so that "no data" states are diagnosable.
				fmt.Printf("warning: failed to parse terraform history for instance %s, resource %s: %v\n", instanceID, resourceID, err)
			}
		}

		// Extract plan previews directly from the state configmap data.
		// This is the same configmap we just successfully read history from,
		// so it avoids the issue where terraformDataForResource may fail to find it.
		planPreviews, previewErrors = findAllPlanPreviews(stateConfigMap.Data)
	} else {
		planPreviews = make(map[string]string)
		previewErrors = make(map[string]string)
	}

	// Also check dedicated tf-plan-* ConfigMaps for plan previews.
	// These are per-operation ConfigMaps with data keys "plan-preview" and "plan-preview-error".
	planCMPreviews, planCMErrors := index.planPreviewsForResource(resourceID)
	mergeStringMapNewKeys(planPreviews, planCMPreviews)
	mergeStringMapNewKeys(previewErrors, planCMErrors)

	// If we have no history and no plan previews, there's nothing useful to return
	if len(history) == 0 && len(planPreviews) == 0 && len(previewErrors) == 0 {
		return nil
	}

	// Find the latest progress configmap that matches this resource/instance
	// Progress configmaps contain resourceID and instanceID fields we can match on.
	// We pick the one with the latest startedAt timestamp.
	normalizedInstanceID := strings.ToLower(instanceID)
	lowerResourceID := strings.ToLower(resourceID)

	var progressData *TerraformProgressData
	var latestProgressTime time.Time

	for _, cm := range index.progress {
		progressJSON, ok := cm.Data["progress"]
		if !ok {
			continue
		}

		var pd TerraformProgressData
		if err := json.Unmarshal([]byte(progressJSON), &pd); err != nil {
			continue
		}

		// Match by resource ID and instance ID (case-insensitive)
		if strings.ToLower(pd.ResourceID) != lowerResourceID || strings.ToLower(pd.InstanceID) != normalizedInstanceID {
			continue
		}

		t, err := time.Parse(time.RFC3339Nano, pd.StartedAt)
		if err != nil {
			// Still use it if it's the only match
			if progressData == nil {
				progressData = &pd
			}
			continue
		}

		if progressData == nil || t.After(latestProgressTime) {
			progressData = &pd
			latestProgressTime = t
		}
	}

	return &TerraformStateData{
		Progress:      progressData,
		History:       history,
		PlanPreviews:  planPreviews,
		PreviewErrors: previewErrors,
	}
}

func normalizeResourceIDForConfigMap(resourceID string) string {
	id := strings.ToLower(resourceID)
	id = strings.ReplaceAll(id, "-", "")
	return id
}

// resourceConfigMapKeys returns the ordered list of index keys to try when
// looking up a resource in stateByResource. Configmap names follow the
// pattern tf-state-tf-r-{lowercased_resource_id}-instance-{instance_id},
// so the regex-extracted index key is "tf-r-{lowercased_resource_id}".
// We try the documented format first, then fall back to less common variants.
func resourceConfigMapKeys(resourceID string) []string {
	lowered := strings.ToLower(resourceID)                    // r-eialbqvwcd
	normalized := normalizeResourceIDForConfigMap(resourceID) // reialbqvwcd
	return []string{
		"tf-" + lowered,    // tf-r-eialbqvwcd  (documented format)
		lowered,            // r-eialbqvwcd
		resourceID,         // r-EIAlBQvwCd     (raw, exact case)
		"tf-" + normalized, // tf-reialbqvwcd (fallback, no dashes)
		normalized,         // reialbqvwcd    (fallback, no dashes)
	}
}

// fetchInstanceDataForResource gets the resource instance data needed for k8s access
func fetchInstanceDataForResource(ctx context.Context, token, serviceID, environmentID, instanceID string) (*openapiclientfleet.ResourceInstance, error) {
	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to describe resource instance: %w", err)
	}
	return instanceData, nil
}

// mergeStringMapNewKeys copies entries from src into dst, skipping any key already present in dst.
func mergeStringMapNewKeys(dst, src map[string]string) {
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
}
