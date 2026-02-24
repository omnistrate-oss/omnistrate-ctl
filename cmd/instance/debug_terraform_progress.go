package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
func fetchTerraformProgress(ctx context.Context, token string, instanceData *openapiclientfleet.ResourceInstance, instanceID, resourceID string) (*TerraformProgressData, []TerraformHistoryEntry, *k8sConnection, error) {
	index, conn, err := loadTerraformConfigMapIndexForInstance(ctx, token, instanceData, instanceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load terraform configmap index: %w", err)
	}

	// Normalize resource ID for configmap lookup
	// Configmap names use format: tf-state-tf-r-{lowercased_resource_id}-instance-{instance_id}
	// The index key extracted by regex is "tf-r-{lowercased_resource_id}"
	// The node's resource ID is like "r-EIAlBQvwCd"
	normalizedResourceID := normalizeResourceIDForConfigMap(resourceID)

	// Find the tf-state configmap for this resource, trying multiple key formats
	var stateConfigMap *corev1.ConfigMap
	var ok bool
	for _, key := range []string{
		normalizedResourceID,                            // reialbqvwcd
		resourceID,                                      // r-EIAlBQvwCd
		"tf-" + normalizedResourceID,                    // tf-reialbqvwcd
		"tf-" + strings.ToLower(resourceID),             // tf-r-eialbqvwcd
	} {
		stateConfigMap, ok = index.stateByResource[key]
		if ok {
			break
		}
	}
	if !ok {
		return nil, nil, nil, nil
	}

	// Parse history from the configmap
	historyJSON, ok := stateConfigMap.Data["history"]
	if !ok {
		return nil, nil, nil, nil
	}

	var history []TerraformHistoryEntry
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse history: %w", err)
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

	if progressData == nil {
		return nil, history, conn, nil
	}

	return progressData, history, conn, nil
}

func normalizeResourceIDForConfigMap(resourceID string) string {
	id := strings.ToLower(resourceID)
	id = strings.ReplaceAll(id, "-", "")
	return id
}

// findLatestOperationID finds the latest operation ID from history that has a progress configmap
func findLatestOperationID(history []TerraformHistoryEntry) string {
	if len(history) == 0 {
		return ""
	}

	// Group by operation ID and find the latest start time
	type opInfo struct {
		id        string
		latestAt  time.Time
		hasNonErr bool
	}
	ops := make(map[string]*opInfo)

	for _, entry := range history {
		info, ok := ops[entry.OperationID]
		if !ok {
			info = &opInfo{id: entry.OperationID}
			ops[entry.OperationID] = info
		}
		if entry.Status != "failed" && entry.Status != "error" {
			info.hasNonErr = true
		}
		t, err := time.Parse(time.RFC3339Nano, entry.StartedAt)
		if err == nil && t.After(info.latestAt) {
			info.latestAt = t
		}
	}

	// Sort operations by latest timestamp descending
	sorted := make([]*opInfo, 0, len(ops))
	for _, info := range ops {
		sorted = append(sorted, info)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].latestAt.After(sorted[j].latestAt)
	})

	if len(sorted) > 0 {
		return sorted[0].id
	}
	return ""
}

// fetchInstanceDataForResource gets the resource instance data needed for k8s access
func fetchInstanceDataForResource(ctx context.Context, token, serviceID, environmentID, instanceID string) (*openapiclientfleet.ResourceInstance, error) {
	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to describe resource instance: %w", err)
	}
	return instanceData, nil
}
