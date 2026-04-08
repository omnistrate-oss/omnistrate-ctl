package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"` //nolint:gosec
}

type refreshTokenResponse struct {
	JWTToken     string `json:"jwtToken"`     //nolint:gosec
	RefreshToken string `json:"refreshToken"` //nolint:gosec
}

// RefreshToken exchanges a refresh token for a new JWT + refresh token pair.
// Uses raw HTTP because the SDK does not yet include this endpoint.
func RefreshToken(ctx context.Context, refreshToken string) (LoginResult, error) {
	reqBody, err := json.Marshal(refreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	url := fmt.Sprintf("%s://%s/2022-09-01-00/refresh-token", config.GetHostScheme(), config.GetHost())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.GetUserAgent())

	client := getRetryableHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return LoginResult{}, fmt.Errorf("refresh token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return LoginResult{}, fmt.Errorf("refresh token failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result refreshTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return LoginResult{}, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if result.JWTToken == "" {
		return LoginResult{}, fmt.Errorf("refresh response missing jwtToken")
	}

	return LoginResult(result), nil
}
