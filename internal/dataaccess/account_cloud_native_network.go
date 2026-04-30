package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

// CloudNativeNetworkResult is a convenience alias for the fleet response type.
type CloudNativeNetworkResult = openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult

// The cloud-native-network endpoints are exposed by consumption-service under the
// fleet path so that ingress routes them correctly. The v1 SDK currently only
// generates the AccountConfigApi paths (/accountconfig/...), which are not
// served, so we call the fleet routes directly. Result JSON shape matches
// openapiclientv1.ListAccountConfigCloudNativeNetworksResult.
func cnnFleetURL(accountConfigID string, suffix ...string) string {
	base := fmt.Sprintf("%s://%s/2022-09-01-00/fleet/account-config/%s/cloud-native-networks",
		config.GetHostScheme(), config.GetHost(), url.PathEscape(accountConfigID))
	for _, s := range suffix {
		base += "/" + url.PathEscape(s)
	}
	return base
}

func doCNNRequest(ctx context.Context, token, method, requestURL string, body any) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.GetUserAgent())

	client := getRetryableHttpClient()
	resp, err := client.Do(req) //nolint:gosec // CLI intentionally targets the configured Omnistrate API host
	if err != nil {
		return nil, fmt.Errorf("cloud-native-network request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cloud-native-network API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(respBody))
		}
	}
	return &result, nil
}

// SyncAccountConfigCloudNativeNetworks triggers cloud-native network discovery for an account configuration.
// Optional regions narrow the discovery; when empty the platform uses all regions from the service plan.
// The request body uses the cloudNativeNetworks[{region, cloudNativeNetworkId?}] shape so individual VPC
// IDs can be re-validated. When only regions are passed we send a target per region with cloudNativeNetworkId
// omitted, which sweeps every VPC in that region.
func SyncAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string, regions []string) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	type target struct {
		Region               string `json:"region"`
		CloudNativeNetworkID string `json:"cloudNativeNetworkId,omitempty"`
	}
	body := map[string]any{}
	if len(regions) > 0 {
		targets := make([]target, len(regions))
		for i, r := range regions {
			targets[i] = target{Region: r}
		}
		body["cloudNativeNetworks"] = targets
	}
	return doCNNRequest(ctx, token, http.MethodPost, cnnFleetURL(accountConfigID, "sync"), body)
}

// ListAccountConfigCloudNativeNetworks lists registered cloud-native networks for an account configuration.
func ListAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	return doCNNRequest(ctx, token, http.MethodGet, cnnFleetURL(accountConfigID), nil)
}

// ImportAccountConfigCloudNativeNetwork marks a cloud-native network as READY for deployments.
func ImportAccountConfigCloudNativeNetwork(ctx context.Context, token, accountConfigID, networkID string) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	return doCNNRequest(ctx, token, http.MethodPost, cnnFleetURL(accountConfigID, networkID, "import"), nil)
}

// UnimportAccountConfigCloudNativeNetwork reverts a cloud-native network back to AVAILABLE.
func UnimportAccountConfigCloudNativeNetwork(ctx context.Context, token, accountConfigID, networkID string) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	return doCNNRequest(ctx, token, http.MethodPost, cnnFleetURL(accountConfigID, networkID, "unimport"), nil)
}

// BulkImportAccountConfigCloudNativeNetworks imports multiple cloud-native networks in a single request.
func BulkImportAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string, networkIDs []string) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	type op struct {
		CloudNativeNetworkID string `json:"cloudNativeNetworkId"`
		Import               bool   `json:"import"`
	}
	ops := make([]op, len(networkIDs))
	for i, id := range networkIDs {
		ops[i] = op{CloudNativeNetworkID: id, Import: true}
	}
	body := map[string]any{"cloudNativeNetworks": ops}
	return doCNNRequest(ctx, token, http.MethodPost, cnnFleetURL(accountConfigID, "import"), body)
}
