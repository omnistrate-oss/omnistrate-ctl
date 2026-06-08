package dataaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAccountConfigCloudNativeNetworks_SendsTargetedAzureVNetID(t *testing.T) {
	azureVNetID := "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/customer-rg/providers/Microsoft.Network/virtualNetworks/customer-vnet"
	var capturedMethod string
	var capturedPath string
	var capturedAuth string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudNativeNetworks":[]}`))
	}))
	defer server.Close()
	setTestHost(t, server.URL)

	_, err := SyncAccountConfigCloudNativeNetworks(context.Background(), "test-token", "ac-test", []CloudNativeNetworkTarget{
		{Region: "eastus", NetworkID: azureVNetID},
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/sync", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)

	targets, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, targets, 1)
	target, ok := targets[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "eastus", target["region"])
	assert.Equal(t, azureVNetID, target["cloudNativeNetworkId"])
}

func TestImportAccountConfigCloudNativeNetwork_EscapesAzureVNetIDPath(t *testing.T) {
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
	setTestHost(t, server.URL)

	_, err := ImportAccountConfigCloudNativeNetwork(context.Background(), "test-token", "ac-test", azureVNetID)
	require.NoError(t, err)

	assert.Equal(t,
		"/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/"+
			url.PathEscape(azureVNetID)+"/import",
		capturedRequestURI)
}

func TestBulkImportAccountConfigCloudNativeNetworks_SendsAzureVNetID(t *testing.T) {
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
	setTestHost(t, server.URL)

	_, err := BulkImportAccountConfigCloudNativeNetworks(context.Background(), "test-token", "ac-test", []string{azureVNetID})
	require.NoError(t, err)

	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/import", capturedPath)
	operations, ok := capturedBody["cloudNativeNetworks"].([]any)
	require.True(t, ok)
	require.Len(t, operations, 1)
	operation, ok := operations[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, azureVNetID, operation["cloudNativeNetworkId"])
	assert.Equal(t, true, operation["import"])
}

func TestUnimportAccountConfigCloudNativeNetwork_EscapesAzureVNetIDPath(t *testing.T) {
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
	setTestHost(t, server.URL)

	_, err := UnimportAccountConfigCloudNativeNetwork(context.Background(), "test-token", "ac-test", azureVNetID)
	require.NoError(t, err)

	assert.Equal(t,
		"/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/"+
			url.PathEscape(azureVNetID)+"/unimport",
		capturedRequestURI)
}

func setTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
}
