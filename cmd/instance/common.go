package instance

import (
	"fmt"
	"sort"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	TerraformDeploymentType DeploymentType = "terraform"
)

type DeploymentType string

func getTerraformDeploymentName(resourceID, instanceID string) string {
	return strings.ToLower(fmt.Sprintf("tf-%s-%s", resourceID, instanceID))
}

// parseCustomTags reads the repeated --tag flag and converts it into the SDK custom tag format.
func parseCustomTags(cmd *cobra.Command) ([]openapiclientfleet.CustomTag, bool, error) {
	if !cmd.Flags().Changed("tag") {
		return nil, false, nil
	}

	tagMap, err := cmd.Flags().GetStringToString("tag")
	if err != nil {
		return nil, false, err
	}

	if len(tagMap) == 0 {
		return []openapiclientfleet.CustomTag{}, true, nil
	}

	normalized := make(map[string]string, len(tagMap))
	keys := make([]string, 0, len(tagMap))

	for rawKey, rawValue := range tagMap {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, false, fmt.Errorf("tag key cannot be empty")
		}
		if _, exists := normalized[key]; exists {
			return nil, false, fmt.Errorf("duplicate tag key %q", key)
		}
		normalized[key] = rawValue
		keys = append(keys, key)
	}

	sort.Strings(keys)

	customTags := make([]openapiclientfleet.CustomTag, 0, len(keys))
	for _, key := range keys {
		customTags = append(customTags, openapiclientfleet.CustomTag{
			Key:   key,
			Value: normalized[key],
		})
	}

	return customTags, true, nil
}
