package dataaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteResourcePassesDryRun(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedRawQuery string
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedRawQuery = r.URL.RawQuery
		capturedAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	err = DeleteResource(context.Background(), "test-token", "s-123", "r-123", true)
	require.NoError(t, err)

	require.Equal(t, http.MethodDelete, capturedMethod)
	require.Equal(t, "/2022-09-01-00/service/s-123/resource/r-123", capturedPath)
	require.Equal(t, "dryRun=true", capturedRawQuery)
	require.Equal(t, "Bearer test-token", capturedAuth)
}
