package instance

import (
	"testing"

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
			"tags":            "team=platform",
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

	require.True(t, filters.HasResourceInstanceFilters())
	require.Len(t, filters.ResourceInstance.Predicates, 2)
	assert.Equal(t, "instance-1", filters.ResourceInstance.Predicates[0].InstanceID)
	assert.Equal(t, "postgres", filters.ResourceInstance.Predicates[0].ServiceName)
	assert.Equal(t, "prod", filters.ResourceInstance.Predicates[0].EnvironmentName)
	assert.Equal(t, "premium", filters.ResourceInstance.Predicates[0].ProductTierName)
	assert.Equal(t, "v1", filters.ResourceInstance.Predicates[0].ProductTierVersion)
	assert.Equal(t, "writer", filters.ResourceInstance.Predicates[0].ResourceName)
	assert.Equal(t, "aws", filters.ResourceInstance.Predicates[0].CloudProvider)
	assert.Equal(t, "us-west-2", filters.ResourceInstance.Predicates[0].RegionCode)
	assert.Equal(t, "RUNNING", filters.ResourceInstance.Predicates[0].Status)
	assert.Equal(t, "sub-1", filters.ResourceInstance.Predicates[0].SubscriptionID)
	assert.Equal(t, "", filters.ResourceInstance.Predicates[0].Tags)
	assert.Equal(t, "mysql", filters.ResourceInstance.Predicates[1].ServiceName)
	assert.ElementsMatch(t, []resourceInstanceTagFilter{
		{Key: "env", Value: "prod"},
		{Key: "team", Value: "platform"},
	}, filters.ResourceInstance.Tags)
}

func TestBuildResourceInstanceSearchFiltersReturnsEmptyForUnsupportedOnly(t *testing.T) {
	filters := buildResourceInstanceSearchFilters([]map[string]string{{"tags": "team=platform"}}, nil)

	assert.False(t, filters.HasResourceInstanceFilters())
}
