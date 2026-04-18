package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
)

func ListVersions(ctx context.Context, token, serviceID, productTierID string) (*openapiclient.ListTierVersionSetsResult, error) {
	return listVersionsWithPayload(ctx, token, serviceID, productTierID)
}

func FindLatestVersion(ctx context.Context, token, serviceID, productTierID string) (string, error) {
	res, err := listVersionsWithPayload(ctx, token, serviceID, productTierID)
	if err != nil {
		return "", err
	}

	if len(res.TierVersionSets) == 0 {
		return "", errors.New("no version found")
	}

	return res.TierVersionSets[0].Version, nil
}

func FindPreferredVersion(ctx context.Context, token, serviceID, productTierID string) (string, error) {
	res, err := listVersionsWithPayload(ctx, token, serviceID, productTierID)
	if err != nil {
		return "", err
	}

	if len(res.TierVersionSets) == 0 {
		return "", errors.New("no version found")
	}

	for _, versionSet := range res.TierVersionSets {
		if versionSet.Status == "Preferred" {
			return versionSet.Version, nil
		}
	}

	return "", errors.New("no preferred version found")
}

func DescribeLatestVersion(ctx context.Context, token, serviceID, productTierID string) (*openapiclient.TierVersionSet, error) {
	res, err := listVersionsWithPayload(ctx, token, serviceID, productTierID)
	if err != nil {
		return nil, err
	}

	if len(res.TierVersionSets) == 0 {
		return nil, errors.New("no version found")
	}

	return &res.TierVersionSets[0], nil
}

func listVersionsWithPayload(ctx context.Context, token, serviceID, productTierID string) (*openapiclient.ListTierVersionSetsResult, error) {
	requestURL := fmt.Sprintf(
		"%s://%s/2022-09-01-00/service/%s/productTier/%s/version-set",
		config.GetHostScheme(),
		config.GetHost(),
		url.PathEscape(serviceID),
		url.PathEscape(productTierID),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, bytes.NewBufferString(`{}`))
	if err != nil {
		return nil, fmt.Errorf("failed to create version-set list request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, application/vnd.goa.error")
	request.Header.Set("User-Agent", config.GetUserAgent())

	response, err := getRetryableHttpClient().Do(request) //nolint:gosec // the CLI intentionally targets the configured Omnistrate API host
	if err != nil {
		return nil, fmt.Errorf("version-set list request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read version-set list response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, formatV1RawHTTPError(response.StatusCode, body)
	}

	var result openapiclient.ListTierVersionSetsResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse version-set list response: %w", err)
	}

	return &result, nil
}

func formatV1RawHTTPError(statusCode int, body []byte) error {
	var apiError openapiclient.Error
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Name != "" {
		return fmt.Errorf("%s\nDetail: %s", apiError.Name, apiError.Message)
	}

	if trimmedBody := strings.TrimSpace(string(body)); trimmedBody != "" {
		return fmt.Errorf("request failed (HTTP %d): %s", statusCode, trimmedBody)
	}

	return fmt.Errorf("request failed (HTTP %d)", statusCode)
}

func DescribeVersionSet(ctx context.Context, token, serviceID, productTierID, version string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiDescribeTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}

func SetDefaultServicePlan(ctx context.Context, token, serviceID, productTierID, version string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiPromoteTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}

func UpdateVersionSetName(ctx context.Context, token, serviceID, productTierID, version, newName string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	updateRequest := openapiclient.NewUpdateTierVersionSetRequest2(newName)

	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiUpdateTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).UpdateTierVersionSetRequest2(*updateRequest).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}
