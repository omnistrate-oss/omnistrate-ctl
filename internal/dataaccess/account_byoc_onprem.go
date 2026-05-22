package dataaccess

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

// DownloadByocOnPremInstallKit fetches the install kit tar archive for a
// BYOC On-Premise account configuration from the consumption service.
func DownloadByocOnPremInstallKit(ctx context.Context, token string, accountConfigID string) ([]byte, string, error) {
	accountConfigID = strings.TrimSpace(accountConfigID)
	if accountConfigID == "" {
		return nil, "", fmt.Errorf("account config ID is required")
	}

	kitURL := fmt.Sprintf("%s://%s/2022-09-01-00/account-setup/byoc-onprem?account_config_id=%s",
		config.GetHostScheme(), config.GetHost(), url.QueryEscape(accountConfigID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kitURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create install kit download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", config.GetUserAgent())

	client := getRetryableHttpClient()
	resp, err := client.Do(req) //nolint:gosec // CLI targets the configured Omnistrate API host.
	if err != nil {
		return nil, "", fmt.Errorf("install kit download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("install kit download returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read install kit response: %w", err)
	}

	fileName := fmt.Sprintf("byoc-onprem-install-kit-%s.tar", accountConfigID)
	return data, fileName, nil
}
