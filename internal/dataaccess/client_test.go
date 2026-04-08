package dataaccess

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateResourceInstance_RetriesRateLimitedSDKCalls(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"cloud_provider":"nebius","region":"eu-north1"}`, string(body))

		if attempt < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"name":"rate_limited","message":"try again shortly"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"instance-123"}`))
	}))
	defer server.Close()

	setSDKTestHost(t, server.URL)

	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}
	request.SetCloudProvider("nebius")
	request.SetRegion("eu-north1")

	resp, err := CreateResourceInstance(
		context.Background(),
		"test-token",
		"sp",
		"service",
		"v1",
		"dev",
		"model",
		"plan",
		"resource",
		request,
	)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "instance-123", resp.GetId())
	assert.EqualValues(t, 3, attempts.Load())
}

func TestCreateResourceInstance_DoesNotRetryNonRateLimitedSDKErrors(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"name":"bad_request","message":"invalid request"}`))
	}))
	defer server.Close()

	setSDKTestHost(t, server.URL)

	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}
	request.SetCloudProvider("nebius")
	request.SetRegion("eu-north1")

	resp, err := CreateResourceInstance(
		context.Background(),
		"test-token",
		"sp",
		"service",
		"v1",
		"dev",
		"model",
		"plan",
		"resource",
		request,
	)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "bad_request")
	assert.Contains(t, err.Error(), "invalid request")
	assert.EqualValues(t, 1, attempts.Load())
}

func TestGetRetryableHttpClient_UsesConfiguredTimeout(t *testing.T) {
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "7")

	client := getRetryableHttpClient()

	assert.Equal(t, 7*time.Second, client.Timeout)
}

func setSDKTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)

	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
}
