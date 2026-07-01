package dataaccess

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchInventoryIncludesOptionalFilters(t *testing.T) {
	req := newSearchInventoryRequest("resourceinstance:i", map[string]any{
		"resourceInstance": map[string]any{
			"predicates": []map[string]string{{"serviceName": "postgres"}},
			"tags":       []map[string]string{{"key": "env", "value": "prod"}},
		},
	})
	bodyBytes, err := json.Marshal(req)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(bodyBytes, &body))
	assert.Equal(t, "resourceinstance:i", body["query"])
	assert.Equal(t, map[string]any{
		"resourceInstance": map[string]any{
			"predicates": []any{map[string]any{"serviceName": "postgres"}},
			"tags":       []any{map[string]any{"key": "env", "value": "prod"}},
		},
	}, body["filters"])
}
