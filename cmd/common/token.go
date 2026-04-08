package common

import (
	"context"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func GetTokenWithLogin() (token string, err error) {
	token, err = config.GetToken()
	if err != nil && !errors.Is(err, config.ErrAuthConfigNotFound) && !errors.Is(err, config.ErrConfigFileNotFound) {
		return
	}

	// If token is present, check if it's expired locally before making a network call
	if token != "" {
		if config.IsTokenExpired(token, 30*time.Second) {
			// Token expired or about to expire — try refresh
			newToken, refreshErr := tryRefreshToken()
			if refreshErr == nil {
				return newToken, nil
			}
			log.Debug().Err(refreshErr).Msg("Token refresh failed, will re-authenticate")
			_ = config.RemoveAuthConfig()
			token = ""
		} else {
			// Token still valid locally, verify with server
			_, err = dataaccess.DescribeUser(context.Background(), token)
			if err != nil {
				// Server rejected — try refresh before prompting login
				newToken, refreshErr := tryRefreshToken()
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

// tryRefreshToken attempts to exchange the stored refresh token for a new JWT.
func tryRefreshToken() (string, error) {
	refreshToken, err := config.GetRefreshToken()
	if err != nil {
		return "", err
	}

	result, err := dataaccess.RefreshToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Persist the new token pair
	authConfig := config.AuthConfig{
		Token:        result.JWTToken,
		RefreshToken: result.RefreshToken,
	}
	if err := config.CreateOrUpdateAuthConfig(authConfig); err != nil {
		return "", err
	}

	return result.JWTToken, nil
}
