package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// AccountConfigVPC represents a cloud native network registered with an account configuration.
type AccountConfigVPC struct {
	ID                        string         `json:"id"`
	AccountConfigID           string         `json:"accountConfigId"`
	CloudNativeNetworkID      string         `json:"cloudNativeNetworkId"`
	Region                    string         `json:"region"`
	Name                      string         `json:"name,omitempty"`
	CIDR                      string         `json:"cidr,omitempty"`
	Status                    string         `json:"status"`
	StatusMessage             string         `json:"statusMessage,omitempty"`
	PrivateSubnets            []SubnetDetail `json:"privateSubnets,omitempty"`
	PublicSubnets             []SubnetDetail `json:"publicSubnets,omitempty"`
	SupportsPrivateDeployment *bool          `json:"supportsPrivateDeployment,omitempty"`
	SupportsPublicDeployment  *bool          `json:"supportsPublicDeployment,omitempty"`
	CreatedAt                 string         `json:"createdAt"`
	UpdatedAt                 string         `json:"updatedAt"`
}

// SubnetDetail represents a subnet within a cloud native network.
type SubnetDetail struct {
	ID       string `json:"id"`
	AZ       string `json:"az,omitempty"`
	CIDR     string `json:"cidr,omitempty"`
	IsPublic bool   `json:"isPublic"`
	IsTagged bool   `json:"isTagged"`
}

// ListAccountConfigVPCsResult is the response for cloud native network list/import/unimport/sync operations.
type ListAccountConfigVPCsResult struct {
	CloudNativeNetworks []AccountConfigVPC `json:"cloudNativeNetworks"`
}

// CloudNativeNetworkOperation is a single import/unimport operation for bulk requests.
type CloudNativeNetworkOperation struct {
	CloudNativeNetworkID string `json:"cloudNativeNetworkId"`
	Import               bool   `json:"import"`
}

// BulkImportVPCsRequestBody is the request body for bulk import.
type BulkImportVPCsRequestBody struct {
	CloudNativeNetworks []CloudNativeNetworkOperation `json:"cloudNativeNetworks"`
}

// vpcBaseURL returns the base URL for VPC API calls.
func vpcBaseURL(accountConfigID string) string {
	return fmt.Sprintf("%s://%s/2022-09-01-00/accountconfig/%s/cloud-native-networks",
		config.GetHostScheme(), config.GetHost(), url.PathEscape(accountConfigID))
}

// doVPCRequest performs an authenticated HTTP request and decodes the response.
func doVPCRequest(ctx context.Context, method, url string, body any) (*ListAccountConfigVPCsResult, error) {
	token, ok := ctx.Value(openapiclient.ContextAccessToken).(string)
	if !ok || token == "" {
		return nil, fmt.Errorf("authentication token not found")
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.GetUserAgent())

	httpClient := getRetryableHttpClient()
	resp, err := httpClient.Do(req) //nolint:gosec // the CLI intentionally targets the configured Omnistrate API host
	if err != nil {
		return nil, fmt.Errorf("VPC request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("VPC API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result ListAccountConfigVPCsResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(respBody))
	}

	return &result, nil
}

// SyncAccountConfigVPCs triggers VPC discovery for an account config.
func SyncAccountConfigVPCs(ctx context.Context, token, accountConfigID string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := vpcBaseURL(accountConfigID) + "/sync"
	return doVPCRequest(ctxWithToken, http.MethodPost, url, nil)
}

// ListAccountConfigVPCs lists all registered VPCs for an account config.
func ListAccountConfigVPCs(ctx context.Context, token, accountConfigID string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := vpcBaseURL(accountConfigID)
	return doVPCRequest(ctxWithToken, http.MethodGet, url, nil)
}

// ImportAccountConfigVPC imports a VPC, setting its status to READY.
func ImportAccountConfigVPC(ctx context.Context, token, accountConfigID, vpcID string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := fmt.Sprintf("%s/%s/import", vpcBaseURL(accountConfigID), url.PathEscape(vpcID))
	return doVPCRequest(ctxWithToken, http.MethodPost, url, nil)
}

// UnimportAccountConfigVPC reverts a VPC back to AVAILABLE.
func UnimportAccountConfigVPC(ctx context.Context, token, accountConfigID, vpcID string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := fmt.Sprintf("%s/%s/unimport", vpcBaseURL(accountConfigID), url.PathEscape(vpcID))
	return doVPCRequest(ctxWithToken, http.MethodPost, url, nil)
}

// BulkImportAccountConfigVPCs imports multiple VPCs at once.
func BulkImportAccountConfigVPCs(ctx context.Context, token, accountConfigID string, vpcIDs []string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := vpcBaseURL(accountConfigID) + "/import"
	ops := make([]CloudNativeNetworkOperation, len(vpcIDs))
	for i, id := range vpcIDs {
		ops[i] = CloudNativeNetworkOperation{CloudNativeNetworkID: id, Import: true}
	}
	body := BulkImportVPCsRequestBody{CloudNativeNetworks: ops}
	return doVPCRequest(ctxWithToken, http.MethodPost, url, body)
}
