package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResourceInstanceSearchFilters(t *testing.T) {
	filterMaps := []map[string]string{
		{
			"instance_id":     "instance-1",
			"service":         "postgres",
			"environment":     "prod",
			"plan":            "premium",
			"version":         "v1",
			"resource":        "writer",
			"cloud_provider":  "aws",
			"region":          "us-west-2",
			"status":          "RUNNING",
			"subscription_id": "sub-1",
		},
		{
			"service": "mysql",
		},
	}
	tagFilters := map[string]string{
		"env":  "prod",
		"team": "platform",
	}

	filters := buildResourceInstanceSearchFilters(filterMaps, tagFilters)

	require.True(t, hasResourceInstanceFilters(filters))
	require.NotNil(t, filters.ResourceInstance)
	require.Len(t, filters.ResourceInstance.Predicates, 2)
	assert.Equal(t, "instance-1", filters.ResourceInstance.Predicates[0].GetInstanceId())
	assert.Equal(t, "postgres", filters.ResourceInstance.Predicates[0].GetServiceName())
	assert.Equal(t, "prod", filters.ResourceInstance.Predicates[0].GetEnvironmentName())
	assert.Equal(t, "premium", filters.ResourceInstance.Predicates[0].GetProductTierName())
	assert.Equal(t, "v1", filters.ResourceInstance.Predicates[0].GetProductTierVersion())
	assert.Equal(t, "writer", filters.ResourceInstance.Predicates[0].GetResourceName())
	assert.Equal(t, "aws", filters.ResourceInstance.Predicates[0].GetCloudProvider())
	assert.Equal(t, "us-west-2", filters.ResourceInstance.Predicates[0].GetRegionCode())
	assert.Equal(t, "RUNNING", filters.ResourceInstance.Predicates[0].GetStatus())
	assert.Equal(t, "sub-1", filters.ResourceInstance.Predicates[0].GetSubscriptionId())
	assert.Equal(t, "mysql", filters.ResourceInstance.Predicates[1].GetServiceName())
	assert.ElementsMatch(t, []openapiclientfleet.ResourceInstanceTagFilter{
		{Key: "env", Value: "prod"},
		{Key: "team", Value: "platform"},
	}, filters.ResourceInstance.Tags)
}

func TestBuildResourceInstanceSearchFiltersReturnsEmptyForUnsupportedOnly(t *testing.T) {
	filters := buildResourceInstanceSearchFilters([]map[string]string{{"tags": "team=platform"}}, nil)

	assert.False(t, hasResourceInstanceFilters(filters))
}
