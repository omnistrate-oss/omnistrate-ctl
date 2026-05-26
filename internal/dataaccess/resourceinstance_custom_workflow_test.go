package dataaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteResourceInstanceCustomWorkflow(t *testing.T) {
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
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"workflowExecutionId":"exec-123","workflowId":"cwt-123","status":"RUNNING"}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	result, err := ExecuteResourceInstanceCustomWorkflow(
		context.Background(),
		"test-token",
		"s-123",
		"env-123",
		"instance-123",
		"r-123",
		"cwt-123",
		map[string]any{"primaryPodName": "postgres-1"},
	)
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/service/s-123/environment/env-123/instance/instance-123/custom-workflow/cwt-123/execute", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)
	assert.Equal(t, "r-123", capturedBody["resourceId"])
	assert.Equal(t, map[string]any{"primaryPodName": "postgres-1"}, capturedBody["requestParams"])
	assert.Equal(t, "exec-123", result.GetWorkflowExecutionId())
	assert.Equal(t, "cwt-123", result.GetWorkflowId())
	require.NotNil(t, result.Status)
	assert.Equal(t, "RUNNING", *result.Status)
}

func TestExecuteResourceInstanceCustomWorkflowPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(openapiclientfleet.Error{
			Name:    "bad_request",
			Message: "workflow is not invokable",
		})
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	_, err = ExecuteResourceInstanceCustomWorkflow(context.Background(), "test-token", "s-123", "env-123", "instance-123", "r-123", "cwt-123", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
	assert.Contains(t, err.Error(), "workflow is not invokable")
}
