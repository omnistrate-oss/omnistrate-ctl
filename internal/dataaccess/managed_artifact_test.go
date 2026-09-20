package dataaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListManagedArtifactReleasesSendsFilters(t *testing.T) {
	var capturedAuthorization string
	var capturedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/releases", r.URL.Path)
		capturedAuthorization = r.Header.Get("Authorization")
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"releases": []map[string]any{{
				"bundleVersion":   "r0000020",
				"releaseSequence": 20,
				"releasedAt":      "2026-09-18T12:00:00Z",
				"amenityCount":    15,
				"artifactCount":   57,
				"targetStatus": map[string]any{
					"totalTargetCount":      2,
					"pendingTargetCount":    0,
					"inProgressTargetCount": 0,
					"readyTargetCount":      2,
					"failedTargetCount":     0,
					"skippedTargetCount":    0,
				},
			}},
			"nextPageToken": "next-token",
		})
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := ListManagedArtifactReleases(context.Background(), "test-token", ListManagedArtifactReleasesOptions{
		BundleVersion:  "r0000020",
		ReleasedAfter:  "2026-09-01T00:00:00Z",
		ReleasedBefore: "2026-10-01T00:00:00Z",
		Limit:          25,
		NextPageToken:  "page-token",
	})
	require.NoError(t, err)
	require.Len(t, result.Releases, 1)
	assert.Equal(t, "r0000020", result.Releases[0].BundleVersion)
	assert.Equal(t, "next-token", result.NextPageToken)
	assert.Equal(t, "Bearer test-token", capturedAuthorization)
	assert.Equal(t, "r0000020", capturedQuery.Get("bundleVersion"))
	assert.Equal(t, "2026-09-01T00:00:00Z", capturedQuery.Get("releasedAfter"))
	assert.Equal(t, "2026-10-01T00:00:00Z", capturedQuery.Get("releasedBefore"))
	assert.Equal(t, "25", capturedQuery.Get("limit"))
	assert.Equal(t, "page-token", capturedQuery.Get("nextPageToken"))
}

func TestUpdateManagedArtifactReleasePolicySendsRequest(t *testing.T) {
	var capturedBody model.UpdateManagedArtifactReleasePolicyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/PROD/aws", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"environmentType":        "PROD",
			"cloudProvider":          "aws",
			"autoUpgrade":            false,
			"preferredBundleVersion": "r0000020",
			"effectiveBundleVersion": "r0000020",
		})
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := UpdateManagedArtifactReleasePolicy(
		context.Background(),
		"test-token",
		"PROD",
		"aws",
		model.UpdateManagedArtifactReleasePolicyRequest{
			AutoUpgrade:            false,
			PreferredBundleVersion: "r0000020",
		},
	)
	require.NoError(t, err)
	assert.False(t, capturedBody.AutoUpgrade)
	assert.Equal(t, "r0000020", capturedBody.PreferredBundleVersion)
	assert.Equal(t, "r0000020", result.EffectiveBundleVersion)
}

func TestManagedArtifactAPIErrorIncludesMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"name":"forbidden","message":"managed artifact access is not enabled"}`))
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	_, err := DescribeManagedArtifactReleasePolicy(context.Background(), "test-token", "PROD", "aws")
	require.Error(t, err)
	assert.Equal(t, "forbidden: managed artifact access is not enabled", err.Error())
}

func setManagedArtifactTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
}
