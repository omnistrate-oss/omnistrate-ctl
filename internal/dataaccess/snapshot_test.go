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

func TestListAllSnapshotsAppliesFilters(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedAuth string
	var capturedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		capturedQuery = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"snapshots":[]}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	result, err := ListAllSnapshots(
		context.Background(),
		"test-token",
		"s-123",
		"env-123",
		ListAllSnapshotsOptions{
			ProductTierID: "pt-123",
			SnapshotType:  "ManualSnapshot",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, http.MethodGet, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/service/s-123/environment/env-123/snapshot", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)
	assert.Equal(t, "pt-123", capturedQuery.Get("productTierId"))
	assert.Equal(t, "ManualSnapshot", capturedQuery.Get("snapshotType"))
}

func TestListAllSnapshotsOmitsEmptyFilters(t *testing.T) {
	var capturedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"snapshots":[]}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	result, err := ListAllSnapshots(
		context.Background(),
		"test-token",
		"s-123",
		"env-123",
		ListAllSnapshotsOptions{},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotContains(t, capturedQuery, "productTierId")
	assert.NotContains(t, capturedQuery, "snapshotType")
}

func TestRestoreSnapshotSendsSubscriptionAndCustomNetwork(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"id":"instance-restored"}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	result, err := RestoreSnapshot(
		context.Background(),
		"test-token",
		"s-123",
		"env-123",
		"instance-ss-123",
		map[string]any{"size": "large"},
		"2.0",
		"INTERNAL",
		"cn-123",
		"sub-target",
		false,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "instance-restored", result.GetId())
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/service/s-123/environment/env-123/snapshot/instance-ss-123/restore", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)
	assert.Equal(t, "sub-target", capturedBody["subscriptionId"])
	assert.Equal(t, "cn-123", capturedBody["custom_network_id"])
	assert.Equal(t, "INTERNAL", capturedBody["network_type"])
	assert.Equal(t, "2.0", capturedBody["productTierVersionOverride"])
	assert.Equal(t, map[string]any{"size": "large"}, capturedBody["inputParametersOverride"])
	assert.NotContains(t, capturedBody, "restoreToSourceInstance")
}
