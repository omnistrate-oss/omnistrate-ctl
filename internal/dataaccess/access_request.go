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
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

const accessRequestAPIPath = "/2022-09-01-00/fleet/access-requests"

// The access-request API contract is newer than the currently released
// omnistrate-sdk-go client, so this file calls the public REST routes directly.

// CreateAccessRequestRequest contains the inputs for creating or reusing an access request.
type CreateAccessRequestRequest struct {
	DeploymentCellID string  `json:"deploymentCellId"`
	Role             string  `json:"role"`
	Reason           *string `json:"reason,omitempty"`
}

// ListAccessRequestsOptions contains optional access-request filters.
type ListAccessRequestsOptions struct {
	Statuses       []string
	IncludeDeleted bool
}

// ApproveAccessRequestRequest contains optional manual approval settings.
type ApproveAccessRequestRequest struct {
	DurationMinutes *int `json:"durationMinutes,omitempty"`
}

func accessRequestURL(pathSegments ...string) string {
	requestURL := fmt.Sprintf("%s://%s%s", config.GetHostScheme(), config.GetHost(), accessRequestAPIPath)
	for _, segment := range pathSegments {
		requestURL += "/" + url.PathEscape(segment)
	}
	return requestURL
}

func doAccessRequest(ctx context.Context, token, method, requestURL string, body, result any) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal access request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create access request API request: %w", err)
	}
	authorization := token
	if !strings.Contains(authorization, " ") {
		authorization = "Bearer " + authorization
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", config.GetUserAgent())
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := getRetryableHttpClient().Do(request) //nolint:gosec // CLI intentionally targets the configured Omnistrate API host.
	if err != nil {
		return fmt.Errorf("access request API request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read access request API response: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		var apiError struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		}
		if json.Unmarshal(responseBody, &apiError) == nil && apiError.Message != "" {
			if apiError.Name != "" {
				return fmt.Errorf("%s\nDetail: %s", apiError.Name, apiError.Message)
			}
			return fmt.Errorf("access request API returned %d: %s", response.StatusCode, apiError.Message)
		}
		return fmt.Errorf("access request API returned %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if result != nil {
		trimmedBody := bytes.TrimSpace(responseBody)
		if len(trimmedBody) == 0 || bytes.Equal(trimmedBody, []byte("null")) {
			return fmt.Errorf("access request API returned %d without a JSON result", response.StatusCode)
		}
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("failed to decode access request API response: %w", err)
		}
	}
	return nil
}

// CreateAccessRequest creates or reuses an access request for a deployment cell.
func CreateAccessRequest(ctx context.Context, token string, request CreateAccessRequestRequest) (*model.AccessRequest, error) {
	var result model.AccessRequest
	if err := doAccessRequest(ctx, token, http.MethodPost, accessRequestURL(), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DescribeAccessRequest returns one access request by ID.
func DescribeAccessRequest(ctx context.Context, token, accessRequestID string) (*model.AccessRequest, error) {
	var result model.AccessRequest
	if err := doAccessRequest(ctx, token, http.MethodGet, accessRequestURL(accessRequestID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAccessRequests lists access requests visible to the authenticated organization.
func ListAccessRequests(ctx context.Context, token string, options ListAccessRequestsOptions) (*model.AccessRequestList, error) {
	requestURL, err := url.Parse(accessRequestURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse access request list URL: %w", err)
	}
	query := requestURL.Query()
	for _, status := range options.Statuses {
		query.Add("statuses", status)
	}
	if options.IncludeDeleted {
		query.Set("includeDeleted", "true")
	}
	requestURL.RawQuery = query.Encode()

	var result model.AccessRequestList
	if err := doAccessRequest(ctx, token, http.MethodGet, requestURL.String(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveAccessRequest approves a pending request, optionally overriding the default duration.
func ApproveAccessRequest(ctx context.Context, token, accessRequestID string, request ApproveAccessRequestRequest) (*model.AccessRequest, error) {
	var result model.AccessRequest
	if err := doAccessRequest(ctx, token, http.MethodPost, accessRequestURL(accessRequestID, "approve"), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DenyAccessRequest denies a pending access request.
func DenyAccessRequest(ctx context.Context, token, accessRequestID string) (*model.AccessRequest, error) {
	var result model.AccessRequest
	if err := doAccessRequest(ctx, token, http.MethodPost, accessRequestURL(accessRequestID, "deny"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeAccessRequest revokes an approved access request.
func RevokeAccessRequest(ctx context.Context, token, accessRequestID string) (*model.AccessRequest, error) {
	var result model.AccessRequest
	if err := doAccessRequest(ctx, token, http.MethodPost, accessRequestURL(accessRequestID, "revoke"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IsKubeConfigAccessRequestError reports whether kubeconfig issuance was blocked
// by access-request state rather than by an unrelated authorization failure.
func IsKubeConfigAccessRequestError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "access request")
}

// IsKubeConfigAccessRequestPendingError reports whether kubeconfig issuance is
// temporarily blocked while an automatic access request finishes approval.
func IsKubeConfigAccessRequestPendingError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "access request") && strings.Contains(message, "not approved")
}

// IsManualAccessReasonRequiredError reports whether creating a MANUAL request
// requires the caller to provide a nonblank reason.
func IsManualAccessReasonRequiredError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "manual access requires a nonblank reason")
}
