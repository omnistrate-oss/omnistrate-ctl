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

// ByocOnPremInstallKitFileName returns the default archive file name for a
// BYOC On-Premise account configuration install kit.
func ByocOnPremInstallKitFileName(accountConfigID string) string {
	return fmt.Sprintf("byoc-onprem-install-kit-%s.tar", strings.TrimSpace(accountConfigID))
}

// DownloadByocOnPremInstallKit fetches the install kit tar archive for a
// BYOC On-Premise account configuration from the consumption service and
// streams it into writer.
func DownloadByocOnPremInstallKit(ctx context.Context, token string, accountConfigID string, writer io.Writer) error {
	accountConfigID = strings.TrimSpace(accountConfigID)
	if accountConfigID == "" {
		return fmt.Errorf("account config ID is required")
	}
	if writer == nil {
		return fmt.Errorf("install kit writer is required")
	}

	kitURL := fmt.Sprintf("%s://%s/2022-09-01-00/account-setup/byoc-onprem?account_config_id=%s",
		config.GetHostScheme(), config.GetHost(), url.QueryEscape(accountConfigID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kitURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create install kit download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", config.GetUserAgent())

	client := getRetryableHttpClient()
	resp, err := client.Do(req) //nolint:gosec // CLI targets the configured Omnistrate API host.
	if err != nil {
		return fmt.Errorf("install kit download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("install kit download returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("failed to write install kit response: %w", err)
	}
	return nil
}
