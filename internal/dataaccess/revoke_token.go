package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

type revokeTokenRequest struct {
	RefreshToken string `json:"refreshToken"` //nolint:gosec
}

// RevokeToken invalidates the given refresh token server-side. The
// backend deletes the token hash so it can never be used again.
// Uses raw HTTP because the SDK does not yet include this endpoint.
func RevokeToken(ctx context.Context, refreshToken string) error {
	reqBody, err := json.Marshal(revokeTokenRequest{RefreshToken: refreshToken}) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to marshal revoke request: %w", err)
	}

	url := fmt.Sprintf("%s://%s/2022-09-01-00/revoke-token", config.GetHostScheme(), config.GetHost())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.GetUserAgent())

	client := getRetryableHttpClient()
	resp, err := client.Do(req) //nolint:gosec // the CLI intentionally targets the configured Omnistrate API host
	if err != nil {
		return fmt.Errorf("revoke token request failed: %w", err)
	}
	defer resp.Body.Close()

	// Backend returns 204 No Content on success; treat 2xx as success.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("revoke token failed (HTTP %d)", resp.StatusCode)
}
