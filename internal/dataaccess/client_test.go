package dataaccess

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryPolicyRetriesTransientPostGatewayErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
	}{
		{name: "too many requests", statusCode: http.StatusTooManyRequests},
		{name: "bad gateway", statusCode: http.StatusBadGateway},
		{name: "service unavailable", statusCode: http.StatusServiceUnavailable},
		{name: "gateway timeout", statusCode: http.StatusGatewayTimeout},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com", nil)
			require.NoError(t, err)

			resp := &http.Response{
				StatusCode: tc.statusCode,
				Request:    req,
			}

			shouldRetry, retryErr := retryPolicy(context.Background(), resp, errors.New(http.StatusText(tc.statusCode)))
			require.NoError(t, retryErr)
			require.True(t, shouldRetry)
		})
	}
}

func TestRetryPolicyDoesNotRetryNonTransientPostErrors(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com", nil)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Request:    req,
	}

	shouldRetry, retryErr := retryPolicy(context.Background(), resp, errors.New(http.StatusText(http.StatusBadRequest)))
	require.NoError(t, retryErr)
	require.False(t, shouldRetry)
}

func TestRetryPolicyPreservesDefaultRetryBehaviorForGet(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Request:    req,
	}

	shouldRetry, retryErr := retryPolicy(context.Background(), resp, errors.New(http.StatusText(http.StatusBadGateway)))
	require.NoError(t, retryErr)
	require.True(t, shouldRetry)
}
