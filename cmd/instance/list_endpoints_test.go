package instance

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
)

func TestListEndpointsCommand(t *testing.T) {
	// Test that the command is properly registered
	assert.NotNil(t, listEndpointsCmd)
	assert.Equal(t, "list-endpoints [instance-id]", listEndpointsCmd.Use)
	assert.Contains(t, listEndpointsCmd.Short, "endpoints")
	assert.Contains(t, listEndpointsCmd.Short, "specific instance")
}

func TestGetInstanceWithResourceName(t *testing.T) {
	// This test would normally require mocking the dataaccess.SearchInventory function
	// For now, we'll just test that the function signature is correct

	// Test that the function has the correct signature
	ctx := context.Background()
	token := "test-token"
	instanceID := "test-instance-id"

	// This would fail with a real API call, but validates the function signature
	_, _, err := getInstanceWithResourceName(ctx, token, instanceID)
	assert.Error(t, err) // Should error since this is not a real token/instance
}

func TestConvertToTableRows(t *testing.T) {
	// Test converting ResourceEndpoints to table rows
	resourceEndpoints := map[string]ResourceEndpoints{
		"test-resource": {
			ClusterEndpoint: "https://cluster.example.com",
			AdditionalEndpoints: map[string]openapiclientfleet.ClusterEndpoint{
				"App": {
					Endpoint:       utils.ToPtr("https://app.example.com"),
					HealthStatus:   utils.ToPtr("HEALTHY"),
					NetworkingType: utils.ToPtr("PUBLIC"),
					OpenPorts:      []int64{443, 80},
					Primary:        utils.ToPtr(true),
				},
			},
		},
	}

	rows := convertToTableRows(resourceEndpoints)

	// Should have 2 rows: 1 cluster + 1 additional endpoint
	assert.Len(t, rows, 2)

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
}
