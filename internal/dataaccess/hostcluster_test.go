package dataaccess

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostClusterNodepoolEntityType(t *testing.T) {
	t.Run("supported providers", func(t *testing.T) {
		tests := []struct {
			cloudProvider string
			entityType    string
		}{
			{cloudProvider: "aws", entityType: "NODE_GROUP"},
			{cloudProvider: "gcp", entityType: "NODEPOOL"},
			{cloudProvider: "azure", entityType: "AZURE_NODEPOOL"},
		}

		for _, tt := range tests {
			entityType, err := hostClusterNodepoolEntityType(tt.cloudProvider)
			require.NoError(t, err)
			assert.Equal(t, tt.entityType, entityType)
		}
	})

	t.Run("nebius unsupported", func(t *testing.T) {
		entityType, err := hostClusterNodepoolEntityType("nebius")
		require.Error(t, err)
		assert.Empty(t, entityType)
		assert.Contains(t, err.Error(), "Nebius deployment cells is not yet supported")
	})
}

func TestRestartHostClusterDeployment(t *testing.T) {
	for _, tc := range []struct {
		name, id, token, apiName, message string
		status                            int
	}{
		{name: "accepted", id: "hc-12345678", token: "test-token", status: http.StatusAccepted},
		{name: "bad request", id: "hc-123456789", token: "test-token", status: http.StatusBadRequest, apiName: "bad_request", message: "Invalid host cluster ID"},
		{name: "invalid token", id: "hc-12345678", token: "invalid-token", status: http.StatusUnauthorized, apiName: "auth_failure", message: "Invalid token"},
		{name: "forbidden", id: "hc-12345678", token: "test-token", status: http.StatusForbidden, apiName: "forbidden", message: "Access denied"},
		{name: "not found", id: "hc-missing", token: "test-token", status: http.StatusNotFound, apiName: "not_found", message: "Deployment cell not found"},
		{name: "invalid state", id: "hc-12345678", token: "test-token", status: http.StatusConflict, apiName: "invalid_state", message: "Deployment cell is already restarting"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/2022-09-01-00/fleet/host-cluster/"+tc.id+"/restart-deployment", r.URL.Path)
				assert.Equal(t, "Bearer "+tc.token, r.Header.Get("Authorization"))
				assert.Empty(t, r.URL.RawQuery)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Empty(t, body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.apiName != "" {
					assert.NoError(t, json.NewEncoder(w).Encode(openapiclientfleet.Error{
						Name: tc.apiName, Message: tc.message, Id: "test-error",
					}))
				}
			}))
			t.Cleanup(server.Close)
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			t.Setenv("OMNISTRATE_HOST", serverURL.Host)
			t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
			t.Setenv("OMNISTRATE_RETRY_MAX", "0")

			err = RestartHostClusterDeployment(t.Context(), tc.token, tc.id)
			if tc.message != "" {
				want := tc.message
				if tc.apiName != "" {
					want = tc.apiName + "\nDetail: " + tc.message
				}
				require.EqualError(t, err, want)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, requests)
		})
	}
}

func TestRestartHostClusterDeploymentCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("cancelled restart must not send a request")
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = RestartHostClusterDeployment(ctx, "test-token", "hc-12345678")
	require.ErrorIs(t, err, context.Canceled)
}
