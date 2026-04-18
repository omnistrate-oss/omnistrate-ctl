package dataaccess

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListVersionsSendsJSONPayload(t *testing.T) {
	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/2022-09-01-00/service/service-123/productTier/tier-456/version-set", r.URL.Path)
		require.Equal(t, "Bearer token-abc", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Contains(t, r.Header.Get("Accept"), "application/json")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"tierVersionSets":[{"baseVersion":"1.0","createdAt":"2024-01-01T00:00:00Z","enabledFeatures":[],"features":{},"productTierId":"tier-456","releasedAt":"2024-01-01T00:00:00Z","serviceId":"service-123","serviceModelId":"model-1","status":"Preferred","type":"MAJOR","updatedAt":"2024-01-01T00:00:00Z","version":"1.0"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	t.Setenv("OMNISTRATE_HOST", strings.TrimPrefix(server.URL, "http://"))

	result, err := ListVersions(context.Background(), "token-abc", "service-123", "tier-456")
	require.NoError(t, err)
	require.Len(t, result.TierVersionSets, 1)
	require.Equal(t, "1.0", result.TierVersionSets[0].Version)
}

func TestListVersionsFormatsAPIError(t *testing.T) {
	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"fault":false,"id":"1","message":"missing required payload","name":"missing_payload","temporary":false,"timeout":false}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	t.Setenv("OMNISTRATE_HOST", strings.TrimPrefix(server.URL, "http://"))

	_, err := ListVersions(context.Background(), "token-abc", "service-123", "tier-456")
	require.EqualError(t, err, "missing_payload\nDetail: missing required payload")
}
