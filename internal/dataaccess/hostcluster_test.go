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

func TestRestartHostClusterDeployment(t *testing.T) {
	for _, tt := range []struct {
		name         string
		statusCode   int
		errorName    string
		errorMessage string
		wantErr      string
	}{
		{name: "no content", statusCode: http.StatusNoContent},
		{name: "accepted", statusCode: http.StatusAccepted},
		{name: "bad request", statusCode: http.StatusBadRequest, errorName: "bad_request", errorMessage: "restart is not allowed", wantErr: "bad_request\nDetail: restart is not allowed"},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, errorName: "unauthorized", errorMessage: "invalid token", wantErr: "unauthorized\nDetail: invalid token"},
		{name: "forbidden", statusCode: http.StatusForbidden, errorName: "forbidden", errorMessage: "access denied", wantErr: "forbidden\nDetail: access denied"},
		{name: "not found", statusCode: http.StatusNotFound, errorName: "not_found", errorMessage: "host cluster not found", wantErr: "not_found\nDetail: host cluster not found"},
		{name: "conflict", statusCode: http.StatusConflict, errorName: "conflict", errorMessage: "restart already in progress", wantErr: "conflict\nDetail: restart already in progress"},
		{name: "server error", statusCode: http.StatusInternalServerError, errorName: "internal_error", errorMessage: "restart failed", wantErr: "internal_error\nDetail: restart failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var method, path, auth, query, cookie string
			var body []byte
			var readErr error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path = r.Method, r.URL.Path
				auth, query, cookie = r.Header.Get("Authorization"), r.URL.RawQuery, r.Header.Get("Cookie")
				body, readErr = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.errorName != "" {
					_ = json.NewEncoder(w).Encode(openapiclientfleet.Error{
						Name: tt.errorName, Message: tt.errorMessage,
					})
				}
			}))
			t.Cleanup(server.Close)
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			t.Setenv("OMNISTRATE_HOST", serverURL.Host)
			t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
			t.Setenv("OMNISTRATE_RETRY_MAX", "0")

			err = RestartHostClusterDeployment(t.Context(), "test-token", "hc-test")
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, readErr)
			require.Equal(t, http.MethodPost, method)
			require.Equal(t, "/2022-09-01-00/fleet/host-cluster/hc-test/restart-deployment", path)
			require.Equal(t, "Bearer test-token", auth)
			require.Empty(t, body)
			require.Empty(t, query)
			require.Empty(t, cookie)
		})
	}
}

func TestRestartHostClusterDeploymentRejectsEmptyID(t *testing.T) {
	for _, id := range []string{"", " ", "\t\n"} {
		require.EqualError(t, RestartHostClusterDeployment(t.Context(), "test-token", id), "host cluster ID cannot be empty")
	}
}

func TestRestartHostClusterDeploymentCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// A canceled request must not reach a server or panic when no response exists.
	t.Setenv("OMNISTRATE_HOST", "127.0.0.1:1")
	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	require.ErrorIs(t, RestartHostClusterDeployment(ctx, "test-token", "hc-test"), context.Canceled)
}

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
