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

func TestChangeUpgradePathTargetVersion_ConstructsSDKPayload(t *testing.T) {
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"completedCount":    0,
			"createdAt":         "2026-06-30T00:00:00Z",
			"failedCount":       0,
			"inProgressCount":   0,
			"pendingCount":      5,
			"productTierId":     "pt-test",
			"releasedAt":        "2026-06-30T00:00:00Z",
			"serviceId":         "s-test",
			"skippedCount":      0,
			"sourceVersion":     "85.0",
			"sourceVersionName": "85.0",
			"status":            "SCHEDULED",
			"targetVersion":     "90.0",
			"targetVersionName": "90.0",
			"totalCount":        5,
			"type":              "ROLLING",
			"updatedAt":         "2026-06-30T00:00:00Z",
			"upgradePathId":     "upgrade-test",
		})
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	result, err := ChangeUpgradePathTargetVersion(
		context.Background(),
		"test-token",
		"s-test",
		"pt-test",
		"upgrade-test",
		"90.0",
	)
	require.NoError(t, err)

	assert.Equal(t, "upgrade-test", result.UpgradePathId)
	assert.Equal(t, "90.0", result.TargetVersion)
	assert.Equal(t, http.MethodPost, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/service/s-test/productTier/pt-test/upgrade-path/upgrade-test/target-version", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)
	assert.Equal(t, "90.0", capturedBody["targetVersion"])
}
