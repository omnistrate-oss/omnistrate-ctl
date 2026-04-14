package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// AccountConfigVPC represents a VPC registered with an account configuration.
type AccountConfigVPC struct {
	ID                        string         `json:"id"`
	AccountConfigID           string         `json:"accountConfigId"`
	VPCID                     string         `json:"vpcId"`
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

// SubnetDetail represents a subnet within a VPC.
type SubnetDetail struct {
	SubnetID         string `json:"subnetId"`
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	CIDR             string `json:"cidr,omitempty"`
}

// ListAccountConfigVPCsResult is the response for VPC list/import/unimport/sync operations.
type ListAccountConfigVPCsResult struct {
	VPCs []AccountConfigVPC `json:"vpcs"`
}

// BulkImportVPCsRequestBody is the request body for bulk import.
type BulkImportVPCsRequestBody struct {
	VPCIDs []string `json:"vpcIds"`
}

// vpcBaseURL returns the base URL for VPC API calls.
func vpcBaseURL(accountConfigID string) string {
	return fmt.Sprintf("%s://%s/2022-09-01-00/accountconfig/%s/vpcs",
		config.GetHostScheme(), config.GetHost(), accountConfigID)
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

	httpClient := getRetryableHttpClient()
	resp, err := httpClient.Do(req)
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
	url := fmt.Sprintf("%s/%s/import", vpcBaseURL(accountConfigID), vpcID)
	return doVPCRequest(ctxWithToken, http.MethodPost, url, nil)
}

// UnimportAccountConfigVPC reverts a VPC back to AVAILABLE.
func UnimportAccountConfigVPC(ctx context.Context, token, accountConfigID, vpcID string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := fmt.Sprintf("%s/%s/unimport", vpcBaseURL(accountConfigID), vpcID)
	return doVPCRequest(ctxWithToken, http.MethodPost, url, nil)
}

// BulkImportAccountConfigVPCs imports multiple VPCs at once.
func BulkImportAccountConfigVPCs(ctx context.Context, token, accountConfigID string, vpcIDs []string) (*ListAccountConfigVPCsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	url := vpcBaseURL(accountConfigID) + "/import"
	body := BulkImportVPCsRequestBody{VPCIDs: vpcIDs}
	return doVPCRequest(ctxWithToken, http.MethodPost, url, body)
}
