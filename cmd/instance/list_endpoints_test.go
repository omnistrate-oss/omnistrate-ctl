package instance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func TestListEndpointsCommand(t *testing.T) {
	// Test that the command is properly registered
	assert.NotNil(t, listEndpointsCmd)
	assert.Equal(t, "list-endpoints [instance-id]", listEndpointsCmd.Use)
	assert.Contains(t, listEndpointsCmd.Short, "endpoints")
	assert.Contains(t, listEndpointsCmd.Short, "specific instance")
}

func TestResourceEndpointsStructure(t *testing.T) {
	// Test the ResourceEndpoints structure
	endpoints := ResourceEndpoints{
		ClusterEndpoint:     "test-cluster-endpoint",
		AdditionalEndpoints: map[string]interface{}{"test": "endpoint"},
	}
	
	assert.Equal(t, "test-cluster-endpoint", endpoints.ClusterEndpoint)
	assert.NotNil(t, endpoints.AdditionalEndpoints)
}

func TestExtractEndpoints(t *testing.T) {
	// Test endpoint extraction with various scenarios
	
	// Test case 1: Instance with DetailedNetworkTopology containing endpoints
	instance1 := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: map[string]interface{}{
				"test-resource": map[string]interface{}{
					"resourceName": "test-resource",
					"clusterEndpoint": "https://cluster.example.com",
					"additionalEndpoints": map[string]interface{}{
						"admin": "https://admin.example.com",
						"api":   "https://api.example.com",
					},
				},
			},
		},
	}
	
	endpoints1 := extractEndpoints(instance1)
	assert.NotNil(t, endpoints1)
	assert.Greater(t, len(endpoints1), 0)
	
	// Check the test-resource endpoints
	testResource, exists := endpoints1["test-resource"]
	assert.True(t, exists)
	assert.Equal(t, "https://cluster.example.com", testResource.ClusterEndpoint)
	assert.NotNil(t, testResource.AdditionalEndpoints)
	
	// Test case 2: Instance with no endpoint information
	instance2 := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: map[string]interface{}{},
		},
	}
	
	endpoints2 := extractEndpoints(instance2)
	assert.Nil(t, endpoints2)
	
	// Test case 3: Instance with only additional endpoints
	instance3 := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: map[string]interface{}{
				"api-resource": map[string]interface{}{
					"resourceName": "api-resource",
					"additionalEndpoints": map[string]interface{}{
						"api": "https://api.example.com",
						"health": "https://health.example.com",
					},
				},
			},
		},
	}
	
	endpoints3 := extractEndpoints(instance3)
	assert.NotNil(t, endpoints3)
	assert.Greater(t, len(endpoints3), 0)
	
	// Check the api-resource endpoints
	apiResource, exists := endpoints3["api-resource"]
	assert.True(t, exists)
	assert.Equal(t, "", apiResource.ClusterEndpoint) // Should be empty
	assert.NotNil(t, apiResource.AdditionalEndpoints)
	assert.Equal(t, 2, len(apiResource.AdditionalEndpoints))
}

func TestGetInstanceWithResourceName(t *testing.T) {
	// This test would normally require mocking the dataaccess.SearchInventory function
	// For now, we'll just test that the function signature is correct
	
	// Test that the function has the correct signature
	ctx := context.Background()
	token := "test-token"
	instanceID := "test-instance-id"
	
	// This would fail with a real API call, but validates the function signature
	_, _, _, err := getInstanceWithResourceName(ctx, token, instanceID)
	assert.Error(t, err) // Should error since this is not a real token/instance
}

func TestConvertToTableRows(t *testing.T) {
	// Test converting ResourceEndpoints to table rows
	resourceEndpoints := map[string]ResourceEndpoints{
		"test-resource": {
			ClusterEndpoint: "https://cluster.example.com",
			AdditionalEndpoints: map[string]any{
				"App": map[string]interface{}{
					"endpoint":        "https://app.example.com",
					"healthStatus":    "HEALTHY",
					"networkingType":  "PUBLIC",
					"openPorts":       []interface{}{443.0, 80.0},
					"primary":         true,
				},
				"API": "https://api.example.com",
			},
		},
	}

	rows := convertToTableRows(resourceEndpoints)

	// Should have 3 rows: 1 cluster + 2 additional endpoints
	assert.Len(t, rows, 3)

	// Check cluster endpoint row
	clusterRow := rows[0]
	assert.Equal(t, "test-resource", clusterRow.ResourceName)
	assert.Equal(t, "cluster", clusterRow.EndpointType)
	assert.Equal(t, "cluster_endpoint", clusterRow.EndpointName)
	assert.Equal(t, "https://cluster.example.com", clusterRow.URL)

	// Check App endpoint row (complex structure)
	var appRow *EndpointTableRow
	for i := range rows {
		if rows[i].EndpointName == "App" {
			appRow = &rows[i]
			break
		}
	}
	assert.NotNil(t, appRow)
	assert.Equal(t, "test-resource", appRow.ResourceName)
	assert.Equal(t, "additional", appRow.EndpointType)
	assert.Equal(t, "App", appRow.EndpointName)
	assert.Equal(t, "https://app.example.com", appRow.URL)
	assert.Equal(t, "HEALTHY", appRow.Status)
	assert.Equal(t, "PUBLIC", appRow.NetworkType)
	assert.Equal(t, "443,80", appRow.Ports)

	// Check API endpoint row (simple string)
	var apiRow *EndpointTableRow
	for i := range rows {
		if rows[i].EndpointName == "API" {
			apiRow = &rows[i]
			break
		}
	}
	assert.NotNil(t, apiRow)
	assert.Equal(t, "test-resource", apiRow.ResourceName)
	assert.Equal(t, "additional", apiRow.EndpointType)
	assert.Equal(t, "API", apiRow.EndpointName)
	assert.Equal(t, "https://api.example.com", apiRow.URL)
}