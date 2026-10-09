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

func TestDescribeManagedArtifactReleasePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/release-policy/PROD/aws", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"environmentType":        "PROD",
			"cloudProvider":          "aws",
			"autoUpgrade":            true,
			"effectiveBundleVersion": "r0000020",
		}))
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := DescribeManagedArtifactReleasePolicy(context.Background(), "test-token", "PROD", "aws")
	require.NoError(t, err)
	assert.Equal(t, "PROD", result.EnvironmentType)
	assert.Equal(t, "aws", result.CloudProvider)
	assert.True(t, result.AutoUpgrade)
	assert.Equal(t, "r0000020", result.EffectiveBundleVersion)
}

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

func TestDescribeManagedArtifactRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/releases/r0000020", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"bundleVersion":   "r0000020",
			"releaseSequence": 20,
			"releasedAt":      "2026-09-18T12:00:00Z",
			"amenityCount":    1,
			"artifactCount":   1,
			"artifacts": []map[string]any{{
				"amenityName":  "cert-manager",
				"artifactKey":  "chart",
				"type":         "HELM_CHART",
				"version":      "v1.15.0",
				"sourceRef":    "oci://example.test/cert-manager",
				"relativePath": "charts/cert-manager",
			}},
		}))
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := DescribeManagedArtifactRelease(context.Background(), "test-token", "r0000020")
	require.NoError(t, err)
	assert.Equal(t, "r0000020", result.BundleVersion)
	assert.EqualValues(t, 20, result.ReleaseSequence)
	require.Len(t, result.Artifacts, 1)
	assert.Equal(t, "cert-manager", result.Artifacts[0].AmenityName)
	assert.Equal(t, "oci://example.test/cert-manager", result.Artifacts[0].SourceRef)
}

func TestListManagedArtifactSyncsSendsFilters(t *testing.T) {
	var capturedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/syncs", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"syncs": []map[string]any{{
				"id":            "spabs-123",
				"bundleVersion": "r0000020",
				"status":        "FAILED",
				"target":        map[string]any{"id": "hc-123"},
			}},
			"nextPageToken": "next-token",
		})
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := ListManagedArtifactSyncs(context.Background(), "test-token", ListManagedArtifactSyncsOptions{
		BundleVersion: "r0000020",
		Status:        "FAILED",
		TargetID:      "hc-123",
		UpdatedAfter:  "2026-09-01T00:00:00Z",
		UpdatedBefore: "2026-10-01T00:00:00Z",
		Limit:         25,
		NextPageToken: "page-token",
	})
	require.NoError(t, err)
	require.Len(t, result.Syncs, 1)
	assert.Equal(t, "spabs-123", result.Syncs[0].ID)
	require.NotNil(t, result.Syncs[0].Target)
	assert.Equal(t, "hc-123", result.Syncs[0].Target.ID)
	assert.Equal(t, "next-token", result.NextPageToken)
	assert.Equal(t, "r0000020", capturedQuery.Get("bundleVersion"))
	assert.Equal(t, "FAILED", capturedQuery.Get("status"))
	assert.Equal(t, "hc-123", capturedQuery.Get("targetId"))
	assert.Equal(t, "2026-09-01T00:00:00Z", capturedQuery.Get("updatedAfter"))
	assert.Equal(t, "2026-10-01T00:00:00Z", capturedQuery.Get("updatedBefore"))
	assert.Equal(t, "25", capturedQuery.Get("limit"))
	assert.Equal(t, "page-token", capturedQuery.Get("nextPageToken"))
	assert.False(t, capturedQuery.Has("registryType"), "legacy requests must not send new filters")
	assert.False(t, capturedQuery.Has("destinationAccountId"))
}

func TestListManagedArtifactSyncsSendsDestinationFilters(t *testing.T) {
	for _, registryType := range []string{"", "PRIVATE_ECR", "PUBLIC_ECR"} {
		t.Run("registry="+registryType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, registryType, r.URL.Query().Get("registryType"))
				assert.Equal(t, "123456789012", r.URL.Query().Get("destinationAccountId"))
				assert.Equal(t, "hc-executor", r.URL.Query().Get("targetId"))
				assert.Equal(t, "scoped-cursor", r.URL.Query().Get("nextPageToken"))
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"syncs":[],"nextPageToken":"next-scoped-cursor"}`))
				assert.NoError(t, err)
			}))
			t.Cleanup(server.Close)
			setManagedArtifactTestHost(t, server.URL)
			result, err := ListManagedArtifactSyncs(t.Context(), "test-token", ListManagedArtifactSyncsOptions{
				RegistryType: registryType, DestinationAccountID: "123456789012", TargetID: "hc-executor", NextPageToken: "scoped-cursor", // #nosec G101 -- opaque pagination fixture, not a credential.
			})
			require.NoError(t, err)
			require.Empty(t, result.Syncs)
			require.Equal(t, "next-scoped-cursor", result.NextPageToken)
		})
	}
}

func TestDescribeManagedArtifactSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/2022-09-01-00/managed-artifact/syncs/spabs-123", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            "spabs-123",
			"bundleVersion": "r0000020",
			"status":        "READY",
			"target":        map[string]any{"id": "hc-123", "region": "us-east-2"},
			"artifacts": []map[string]any{{
				"amenityName": "cert-manager",
				"artifactKey": "chart",
				"version":     "v1.15.0",
				"status":      "READY",
			}},
		})
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	result, err := DescribeManagedArtifactSync(context.Background(), "test-token", "spabs-123")
	require.NoError(t, err)
	assert.Equal(t, "spabs-123", result.ID)
	assert.Equal(t, "READY", result.Status)
	require.NotNil(t, result.Target)
	assert.Equal(t, "us-east-2", result.Target.Region)
	require.Len(t, result.Artifacts, 1)
	assert.Equal(t, "cert-manager", result.Artifacts[0].AmenityName)
}

func TestManagedArtifactRequiredParameters(t *testing.T) {
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "missing policy environment",
			call: func() error {
				_, err := DescribeManagedArtifactReleasePolicy(context.Background(), "unused", "", "aws")
				return err
			},
			want: "environment type is required",
		},
		{
			name: "missing policy cloud provider",
			call: func() error {
				_, err := UpdateManagedArtifactReleasePolicy(context.Background(), "unused", "PROD", "", model.UpdateManagedArtifactReleasePolicyRequest{})
				return err
			},
			want: "cloud provider is required",
		},
		{
			name: "missing release bundle version",
			call: func() error {
				_, err := DescribeManagedArtifactRelease(context.Background(), "unused", "")
				return err
			},
			want: "bundle version is required",
		},
		{
			name: "missing sync ID",
			call: func() error {
				_, err := DescribeManagedArtifactSync(context.Background(), "unused", "")
				return err
			},
			want: "sync ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.call(), tt.want)
		})
	}
}

func TestManagedArtifactAPIRejectsInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer invalid-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"name":"unauthorized","message":"invalid bearer token"}`))
	}))
	defer server.Close()
	setManagedArtifactTestHost(t, server.URL)

	_, err := DescribeManagedArtifactReleasePolicy(context.Background(), "invalid-token", "PROD", "aws")
	require.Error(t, err)
	assert.Equal(t, "unauthorized\nDetail: invalid bearer token", err.Error())
}

func TestManagedArtifactAPIRejectsMissingOrInvalidResults(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "empty response", status: http.StatusOK},
		{name: "whitespace response", status: http.StatusOK, body: " \n "},
		{name: "null response", status: http.StatusOK, body: "null"},
		{name: "no content response", status: http.StatusNoContent},
		{name: "invalid JSON", status: http.StatusOK, body: "not JSON"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)
			setManagedArtifactTestHost(t, server.URL)

			result, err := DescribeManagedArtifactReleasePolicy(t.Context(), "test-token", "DEV", "aws")
			require.Error(t, err)
			assert.Nil(t, result, "invalid responses must not be reported as a zero-valued policy")
		})
	}
}

func setManagedArtifactTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
}
