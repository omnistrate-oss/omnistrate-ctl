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

// parseCustomTags reads the --tags flag and converts it into the SDK custom tag format.
func parseCustomTags(cmd *cobra.Command) ([]openapiclientfleet.CustomTag, bool, error) {
	if !cmd.Flags().Changed("tags") {
		return nil, false, nil
	}

	raw, err := cmd.Flags().GetString("tags")
	if err != nil {
		return nil, false, err
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []openapiclientfleet.CustomTag{}, true, nil
	}

	rawPairs := strings.Split(trimmed, ",")
	tags := make([]openapiclientfleet.CustomTag, 0, len(rawPairs))
	for _, rawPair := range rawPairs {
		pair := strings.TrimSpace(rawPair)
		if pair == "" {
			return nil, false, fmt.Errorf("tag pair cannot be empty")
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, false, fmt.Errorf("invalid tag %q. Tags must use key=value format", pair)
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, false, fmt.Errorf("tag key cannot be empty")
		}

		value := strings.TrimSpace(parts[1])
		tags = append(tags, openapiclientfleet.CustomTag{Key: key, Value: value})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})

	if err := ensureUniqueTagKeys(tags); err != nil {
		return nil, false, err
	}

	return tags, true, nil
}

func ensureUniqueTagKeys(tags []openapiclientfleet.CustomTag) error {
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if _, ok := seen[tag.Key]; ok {
			return fmt.Errorf("duplicate tag key %q", tag.Key)
		}
		seen[tag.Key] = struct{}{}
	}
	return nil
}

// formatTags converts CustomTag slice to a comma-separated string in key=value format
func formatTags(tags []openapiclientfleet.CustomTag) string {
	if len(tags) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts = append(parts, fmt.Sprintf("%s=%s", tag.Key, tag.Value))
	}
	return strings.Join(parts, ",")
}

// parseTagFilters parses tag filters from the command line in the format key=value
// Returns a map of tag key to tag value
func parseTagFilters(tagFilters []string) (map[string]string, error) {
	parsedFilters := make(map[string]string)
	for _, tagFilter := range tagFilters {
		if tagFilter == "[]" {
			continue // Empty filter that is reset to default value
		}
		parts := strings.SplitN(tagFilter, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag filter format: %s, expected key=value", tagFilter)
		}
		parsedFilters[parts[0]] = parts[1]
	}
	return parsedFilters, nil
}

// matchesTagFilters checks if the instance tags match all the provided tag filters
// The tags string is in the format: key1=value1,key2=value2,...
// All tag filters must match for the function to return true (AND operation)
func matchesTagFilters(instanceTagsStr string, tagFilters map[string]string) bool {
	if len(tagFilters) == 0 {
		return true
	}

	if instanceTagsStr == "" {
		return false
	}

	// Parse the instance tags into a map
	instanceTags := make(map[string]string)
	tagPairs := strings.Split(instanceTagsStr, ",")
	for _, tagPair := range tagPairs {
		parts := strings.SplitN(tagPair, "=", 2)
		if len(parts) == 2 {
			instanceTags[parts[0]] = parts[1]
		}
	}

	// Check if all tag filters match
	for key, value := range tagFilters {
		instanceValue, exists := instanceTags[key]
		if !exists || instanceValue != value {
			return false
		}
	}

	return true
}
