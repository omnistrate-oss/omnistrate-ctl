package build

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetchGitHubUser(t *testing.T) {
	originalClient := githubHTTPClient
	t.Cleanup(func() {
		githubHTTPClient = originalClient
	})

	githubHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, githubUserAPIURL, request.URL.String())
		assert.Equal(t, "token test-pat", request.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github+json", request.Header.Get("Accept"))
		assert.Equal(t, "omnistrate-ctl", request.Header.Get("User-Agent"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"login":"test-user"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	responseBody, err := fetchGitHubUser(context.Background(), "test-pat")
	require.NoError(t, err)
	assert.JSONEq(t, `{"login":"test-user"}`, string(responseBody))
}

func TestFetchGitHubUserPreservesAPIErrorResponse(t *testing.T) {
	originalClient := githubHTTPClient
	t.Cleanup(func() {
		githubHTTPClient = originalClient
	})

	githubHTTPClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"message":"Bad credentials"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	responseBody, err := fetchGitHubUser(context.Background(), "invalid-pat")
	require.NoError(t, err)
	assert.JSONEq(t, `{"message":"Bad credentials"}`, string(responseBody))
}
