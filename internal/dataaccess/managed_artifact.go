package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

const managedArtifactAPIPath = "/2022-09-01-00/managed-artifact"

// The public managed-artifact API contract is newer than the currently
// released omnistrate-sdk-go client, so this file calls the REST routes
// directly. Keep the request and response types aligned with the public API.

// ListManagedArtifactReleasesOptions contains optional release filters.
type ListManagedArtifactReleasesOptions struct {
	BundleVersion  string
	ReleasedAfter  string
	ReleasedBefore string
	Limit          int
	NextPageToken  string
}

// ListManagedArtifactSyncsOptions contains optional synchronization filters.
type ListManagedArtifactSyncsOptions struct {
	BundleVersion string
	Status        string
	TargetID      string
	UpdatedAfter  string
	UpdatedBefore string
	Limit         int
	NextPageToken string
}

func managedArtifactURL(pathSegments ...string) string {
	requestURL := fmt.Sprintf("%s://%s%s", config.GetHostScheme(), config.GetHost(), managedArtifactAPIPath)
	for _, segment := range pathSegments {
		requestURL += "/" + url.PathEscape(segment)
	}
	return requestURL
}

func doManagedArtifactRequest(ctx context.Context, token, method, requestURL string, body, result any) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal managed artifact request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create managed artifact request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", config.GetUserAgent())
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := getRetryableHttpClient().Do(request) //nolint:gosec // CLI intentionally targets the configured Omnistrate API host.
	if err != nil {
		return fmt.Errorf("managed artifact request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read managed artifact response: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		var apiError struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		}
		if json.Unmarshal(responseBody, &apiError) == nil && apiError.Message != "" {
			if apiError.Name != "" {
				return fmt.Errorf("%s: %s", apiError.Name, apiError.Message)
			}
			return fmt.Errorf("managed artifact API returned %d: %s", response.StatusCode, apiError.Message)
		}
		return fmt.Errorf("managed artifact API returned %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if result != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("failed to decode managed artifact response: %w", err)
		}
	}
	return nil
}

// DescribeManagedArtifactReleasePolicy returns the release policy for one environment and cloud provider.
func DescribeManagedArtifactReleasePolicy(ctx context.Context, token, environmentType, cloudProvider string) (*model.ManagedArtifactReleasePolicy, error) {
	var result model.ManagedArtifactReleasePolicy
	if err := doManagedArtifactRequest(ctx, token, http.MethodGet,
		managedArtifactURL("release-policy", environmentType, cloudProvider), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateManagedArtifactReleasePolicy updates the release policy for one environment and cloud provider.
func UpdateManagedArtifactReleasePolicy(ctx context.Context, token, environmentType, cloudProvider string, request model.UpdateManagedArtifactReleasePolicyRequest) (*model.ManagedArtifactReleasePolicy, error) {
	var result model.ManagedArtifactReleasePolicy
	if err := doManagedArtifactRequest(ctx, token, http.MethodPut,
		managedArtifactURL("release-policy", environmentType, cloudProvider), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListManagedArtifactReleases lists tenant-visible managed artifact releases.
func ListManagedArtifactReleases(ctx context.Context, token string, options ListManagedArtifactReleasesOptions) (*model.ManagedArtifactReleaseList, error) {
	requestURL, err := url.Parse(managedArtifactURL("releases"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse managed artifact releases URL: %w", err)
	}
	query := requestURL.Query()
	if options.BundleVersion != "" {
		query.Set("bundleVersion", options.BundleVersion)
	}
	if options.ReleasedAfter != "" {
		query.Set("releasedAfter", options.ReleasedAfter)
	}
	if options.ReleasedBefore != "" {
		query.Set("releasedBefore", options.ReleasedBefore)
	}
	if options.Limit > 0 {
		query.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.NextPageToken != "" {
		query.Set("nextPageToken", options.NextPageToken)
	}
	requestURL.RawQuery = query.Encode()

	var result model.ManagedArtifactReleaseList
	if err := doManagedArtifactRequest(ctx, token, http.MethodGet, requestURL.String(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DescribeManagedArtifactRelease returns a release and its managed amenity artifacts.
func DescribeManagedArtifactRelease(ctx context.Context, token, bundleVersion string) (*model.ManagedArtifactRelease, error) {
	var result model.ManagedArtifactRelease
	if err := doManagedArtifactRequest(ctx, token, http.MethodGet,
		managedArtifactURL("releases", bundleVersion), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListManagedArtifactSyncs lists synchronization records for provisioner targets.
func ListManagedArtifactSyncs(ctx context.Context, token string, options ListManagedArtifactSyncsOptions) (*model.ManagedArtifactSyncList, error) {
	requestURL, err := url.Parse(managedArtifactURL("syncs"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse managed artifact syncs URL: %w", err)
	}
	query := requestURL.Query()
	if options.BundleVersion != "" {
		query.Set("bundleVersion", options.BundleVersion)
	}
	if options.Status != "" {
		query.Set("status", options.Status)
	}
	if options.TargetID != "" {
		query.Set("targetId", options.TargetID)
	}
	if options.UpdatedAfter != "" {
		query.Set("updatedAfter", options.UpdatedAfter)
	}
	if options.UpdatedBefore != "" {
		query.Set("updatedBefore", options.UpdatedBefore)
	}
	if options.Limit > 0 {
		query.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.NextPageToken != "" {
		query.Set("nextPageToken", options.NextPageToken)
	}
	requestURL.RawQuery = query.Encode()

	var result model.ManagedArtifactSyncList
	if err := doManagedArtifactRequest(ctx, token, http.MethodGet, requestURL.String(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DescribeManagedArtifactSync returns one synchronization and its artifact results.
func DescribeManagedArtifactSync(ctx context.Context, token, syncID string) (*model.ManagedArtifactSync, error) {
	var result model.ManagedArtifactSync
	if err := doManagedArtifactRequest(ctx, token, http.MethodGet,
		managedArtifactURL("syncs", syncID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
