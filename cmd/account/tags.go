package account

import (
	"fmt"
	"sort"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

// parseAccountTags reads the --tags flag and converts it into the SDK custom tag format.
func parseAccountTags(cmd *cobra.Command) ([]openapiclient.CustomTag, bool, error) {
	if !cmd.Flags().Changed(tagsFlag) {
		return nil, false, nil
	}

	raw, err := cmd.Flags().GetString(tagsFlag)
	if err != nil {
		return nil, false, err
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false, fmt.Errorf("at least one tag must be provided in key=value format when using --%s", tagsFlag)
	}

	rawPairs := strings.Split(trimmed, ",")
	tags := make([]openapiclient.CustomTag, 0, len(rawPairs))
	seen := make(map[string]struct{}, len(rawPairs))
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
		if _, exists := seen[key]; exists {
			return nil, false, fmt.Errorf("duplicate tag key: %s", key)
		}
		seen[key] = struct{}{}

		tags = append(tags, openapiclient.CustomTag{Key: key, Value: strings.TrimSpace(parts[1])})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})

	return tags, true, nil
}

// formatAccountTags renders the account's custom tags as a compact key=value list for table output.
func formatAccountTags(tags []openapiclient.CustomTag) string {
	if len(tags) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(tags))
	for _, tag := range tags {
		pairs = append(pairs, tag.Key+"="+tag.Value)
	}
	sort.Strings(pairs)

	return strings.Join(pairs, ",")
}
