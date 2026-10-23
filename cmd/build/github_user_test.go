package build

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchGitHubUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "token test-pat", request.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github+json", request.Header.Get("Accept"))
		assert.Equal(t, "omnistrate-ctl", request.Header.Get("User-Agent"))
		_, err := writer.Write([]byte(`{"login":"test-user"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	responseBody, err := fetchGitHubUser(context.Background(), server.URL, "test-pat")
	require.NoError(t, err)
	assert.JSONEq(t, `{"login":"test-user"}`, string(responseBody))
}

func TestFetchGitHubUserPreservesAPIErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, err := writer.Write([]byte(`{"message":"Bad credentials"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	responseBody, err := fetchGitHubUser(context.Background(), server.URL, "invalid-pat")
	require.NoError(t, err)
	assert.JSONEq(t, `{"message":"Bad credentials"}`, string(responseBody))
}
