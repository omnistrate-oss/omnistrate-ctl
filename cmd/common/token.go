package common

import (
	"context"
	"os"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	tokenRefreshMargin = 5 * time.Minute
	apiKeyEnv          = "OMNISTRATE_API_KEY" //nolint:gosec
)

func GetTokenWithLogin() (token string, err error) {
	token, err = config.GetToken()
	if err != nil && !errors.Is(err, config.ErrAuthConfigNotFound) && !errors.Is(err, config.ErrConfigFileNotFound) {
		return
	}

	ctx := context.Background()

	// If token is present, check if it's expired locally before making a network call
	if token != "" {
		if shouldRefreshToken(token) {
			// Token expired or about to expire — try refresh
			newToken, refreshErr := tryRefreshToken(ctx)
			if refreshErr == nil {
				return newToken, nil
			}
			log.Debug().Err(refreshErr).Msg("Token refresh failed, will re-authenticate")
			_ = config.RemoveAuthConfig()
			token = ""
			// Fall through to env-var exchange or interactive login below
		} else {
			// Token still valid locally, verify with server
			_, err = dataaccess.DescribeUser(ctx, token)
			if err != nil {
				// Server rejected — try refresh before prompting login
				newToken, refreshErr := tryRefreshToken(ctx)
				if refreshErr == nil {
					return newToken, nil
				}
				_ = config.RemoveAuthConfig()
				token = ""
			} else {
				return
			}
		}
	}

	// If OMNISTRATE_API_KEY is set, exchange it for a JWT transparently
	// without requiring an explicit `login` call.
	if envKey := os.Getenv(apiKeyEnv); envKey != "" {
		return exchangeAPIKeyEnv(ctx, envKey)
	}

	// Run login command (if no token or token was invalid)
	err = login.RunLogin(login.LoginCmd, []string{})
	if err != nil {
		return
	}

	token, err = config.GetToken()
	if err != nil {
		return
	}

	return
}

// exchangeAPIKeyEnv performs a signin-exchange using the OMNISTRATE_API_KEY
// env var and persists the resulting JWT so subsequent calls in the same
// process reuse it.
func exchangeAPIKeyEnv(ctx context.Context, apiKey string) (string, error) {
	result, err := dataaccess.LoginWithAPIKey(ctx, apiKey)
	if err != nil {
		return "", errors.Wrap(err, "OMNISTRATE_API_KEY signin-exchange failed")
	}

	authConfig := config.AuthConfig{
		Token:        result.JWTToken,
		RefreshToken: result.RefreshToken,
	}
	if err := config.CreateOrUpdateAuthConfig(authConfig); err != nil {
		return "", err
	}

	return result.JWTToken, nil
}

func shouldRefreshToken(token string) bool {
	return config.IsTokenExpired(token, tokenRefreshMargin)
}

// tryRefreshToken attempts to exchange the stored refresh token for a new JWT.
func tryRefreshToken(ctx context.Context) (string, error) {
	refreshToken, err := config.GetRefreshToken()
	if err != nil {
		return "", err
	}

	result, err := dataaccess.RefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	if result.JWTToken == "" {
		return "", errors.New("refresh response missing jwt token")
	}

	// Preserve the existing refresh token if the server didn't return a new one
	persistedRefreshToken := result.RefreshToken
	if persistedRefreshToken == "" {
		persistedRefreshToken = refreshToken
	}

	authConfig := config.AuthConfig{
		Token:        result.JWTToken,
		RefreshToken: persistedRefreshToken,
	}
	if err := config.CreateOrUpdateAuthConfig(authConfig); err != nil {
		return "", err
	}

	return result.JWTToken, nil
}
