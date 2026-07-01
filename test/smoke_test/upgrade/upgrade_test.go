package upgrade

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_upgrade_change_target_version(t *testing.T) {
	testutils.SmokeTest(t)

	captured := &changeTargetVersionCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2022-09-01-00/signin":
			handleSmokeSignin(t, w, r)
		case "/2022-09-01-00/fleet/service/s-test/productTier/pt-test/upgrade-path/upgrade-test/target-version":
			captured.method = r.Method
			captured.auth = r.Header.Get("Authorization")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&captured.body))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode(changeTargetVersionResponse()))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	pointSmokeClientAt(t, server)
	t.Setenv("OMNISTRATE_API_KEY", "om_test_key")

	cmd.RootCmd.SetArgs([]string{
		"upgrade", "change-target-version", "upgrade-test",
		"--service-id", "s-test",
		"--product-tier-id", "pt-test",
		"--target-version", "90.0",
		"--output", "json",
	})
	require.NoError(t, cmd.RootCmd.ExecuteContext(context.Background()))

	require.Equal(t, http.MethodPost, captured.method)
	require.Equal(t, "Bearer fake-jwt-for-testing", captured.auth)
	require.Equal(t, "90.0", captured.body["targetVersion"])
}

type changeTargetVersionCapture struct {
	method string
	auth   string
	body   map[string]any
}

func handleSmokeSignin(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	var req map[string]string
	require.NoError(t, json.Unmarshal(body, &req))
	require.Equal(t, dataaccess.APIKeySigninEmail, req["email"])
	require.Equal(t, "om_test_key", req["password"])

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = io.WriteString(w, `{"jwtToken":"fake-jwt-for-testing"}`)
	require.NoError(t, err)
}

func pointSmokeClientAt(t *testing.T, server *httptest.Server) {
	t.Helper()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
	homedir.Reset()
	t.Cleanup(homedir.Reset)
}

func changeTargetVersionResponse() map[string]any {
	return map[string]any{
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
	}
}
