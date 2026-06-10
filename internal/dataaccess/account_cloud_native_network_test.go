package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAccountConfigCloudNativeNetworkSendsRegionPath(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedRequestURI string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := ImportAccountConfigCloudNativeNetwork(context.Background(), "test-token", "ac-test", "eastus", azureVNetID)
	require.NoError(t, err)

	assert.Equal(t,
		"/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/eastus/"+
			url.PathEscape(azureVNetID)+"/import",
		capturedRequestURI)
}

func TestSyncAccountConfigCloudNativeNetworksByTargetSendsRegionAndNetworkID(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedPath string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := SyncAccountConfigCloudNativeNetworksByTarget(context.Background(), "test-token", "ac-test", []CloudNativeNetworkTarget{
		{Region: "eastus", NetworkID: azureVNetID},
	})
	require.NoError(t, err)

	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/sync", capturedPath)
	targets, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, targets, 1)
	target, ok := targets[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, azureVNetID, target["cloudNativeNetworkId"])
	assert.Equal(t, "eastus", target["region"])
	assert.NotContains(t, target, "includeHostClusters")
}

func TestSyncAccountConfigCloudNativeNetworksByTargetSendsIncludeHostClusters(t *testing.T) {
	includeHostClusters := true
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := SyncAccountConfigCloudNativeNetworksByTarget(context.Background(), "test-token", "ac-test", []CloudNativeNetworkTarget{
		{Region: "eastus", IncludeHostClusters: &includeHostClusters},
	})
	require.NoError(t, err)

	targets, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, targets, 1)
	target, ok := targets[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "eastus", target["region"])
	assert.Equal(t, true, target["includeHostClusters"])
	assert.NotContains(t, target, "cloudNativeNetworkId")
}

func TestBulkImportAccountConfigCloudNativeNetworksSendsRegion(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedPath string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := BulkImportAccountConfigCloudNativeNetworks(context.Background(), "test-token", "ac-test", []CloudNativeNetworkTarget{
		{Region: "eastus", NetworkID: azureVNetID},
	})
	require.NoError(t, err)

	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/import", capturedPath)
	operations, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, operations, 1)
	operation, ok := operations[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, azureVNetID, operation["cloudNativeNetworkId"])
	assert.Equal(t, "eastus", operation["region"])
	assert.Equal(t, true, operation["import"])
}

func TestBulkUnimportAccountConfigCloudNativeNetworksSendsRegion(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedPath string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := BulkUnimportAccountConfigCloudNativeNetworks(context.Background(), "test-token", "ac-test", []CloudNativeNetworkTarget{
		{Region: "eastus", NetworkID: azureVNetID},
	})
	require.NoError(t, err)

	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/import", capturedPath)
	operations, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, operations, 1)
	operation, ok := operations[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, azureVNetID, operation["cloudNativeNetworkId"])
	assert.Equal(t, "eastus", operation["region"])
	assert.Equal(t, false, operation["import"])
}

func TestUnimportAccountConfigCloudNativeNetworkSendsRegionPath(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedRequestURI string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	_, err := UnimportAccountConfigCloudNativeNetwork(context.Background(), "test-token", "ac-test", "eastus", azureVNetID)
	require.NoError(t, err)

	assert.Equal(t,
		"/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/eastus/"+
			url.PathEscape(azureVNetID)+"/unimport",
		capturedRequestURI)
}

func TestImportAccountConfigCloudNativeNetworkHostClusterSendsRegionPath(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedRequestURI string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURI = r.RequestURI
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hostClusterId":"hc-test","created":true}`))
	}))
	defer server.Close()
	setAccountCloudNativeNetworkTestHost(t, server.URL)

	res, err := ImportAccountConfigCloudNativeNetworkHostCluster(
		context.Background(),
		"test-token",
		"ac-test",
		"eastus",
		azureVNetID,
		"customer-aks",
	)
	require.NoError(t, err)

	assert.Equal(t,
		"/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/eastus/"+
			url.PathEscape(azureVNetID)+"/host-clusters/customer-aks/import",
		capturedRequestURI)
	assert.Equal(t, "hc-test", res.HostClusterID)
	assert.True(t, res.Created)
}

func TestImportAccountConfigCloudNativeNetworkDeploymentCellUsesFleetRoute(t *testing.T) {
	var requestMethod string
	var requestPath string
	var authorizationHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		authorizationHeader = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hostClusterId":"hc-123","created":true}`)
	}))
	defer server.Close()

	setAccountCloudNativeNetworkTestHost(t, server.URL)

	result, err := ImportAccountConfigCloudNativeNetworkDeploymentCell(
		context.Background(),
		"test-token",
		"ac-test",
		"ap-south-1",
		"vpc-123",
		"deployment-cell-name",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hc-123", result.HostClusterID)
	assert.True(t, result.Created)
	assert.Equal(t, http.MethodPost, requestMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/ap-south-1/vpc-123/host-clusters/deployment-cell-name/import", requestPath)
	assert.Equal(t, "Bearer test-token", authorizationHeader)
}

func setAccountCloudNativeNetworkTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
}
