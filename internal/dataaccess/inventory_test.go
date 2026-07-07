package dataaccess

import (
	"encoding/json"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchInventoryIncludesOptionalFilters(t *testing.T) {
	req := newSearchInventoryRequest("resourceinstance:i", openapiclientfleet.SearchInventoryFilters{
		ResourceInstance: &openapiclientfleet.ResourceInstanceSearchFilters{
			Predicates: []openapiclientfleet.ResourceInstanceFilterGroup{
				{
					ServiceName: ptrTo("postgres"),
				},
			},
			Tags: []openapiclientfleet.ResourceInstanceTagFilter{
				{
					Key:   "env",
					Value: "prod",
				},
			},
		},
	})
	require.NotNil(t, req.Filters)
	assert.Nil(t, req.AdditionalProperties)

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

func ptrTo[T any](value T) *T {
	return &value
}
