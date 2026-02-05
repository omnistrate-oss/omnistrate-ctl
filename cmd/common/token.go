package common

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/pkg/errors"
)

const maxTokenRetries = 3

func GetTokenWithLogin() (token string, err error) {
	return getTokenWithRetry(0)
}

func getTokenWithRetry(retryCount int) (token string, err error) {
	// Check if max retries exceeded
	if retryCount >= maxTokenRetries {
		return "", fmt.Errorf("maximum token validation retries (%d) exceeded, please try again later", maxTokenRetries)
	}

	token, err = config.GetToken()
	if err != nil && !errors.Is(err, config.ErrAuthConfigNotFound) && !errors.Is(err, config.ErrConfigFileNotFound) {
		return "", errors.Wrap(err, "failed to retrieve authentication token")
	}

	// If token is present, validate it by calling the user API
	if token != "" {
		// Validate token by making an API call
		_, err = dataaccess.DescribeUser(context.Background(), token)
		if err != nil {
			// Check if error is due to token expiry or authentication issues
			if errors.Is(err, config.ErrTokenExpired) || errors.Is(err, config.ErrUnauthorized) {
				// Remove expired/invalid token
				_ = config.RemoveAuthConfig()
				token = ""
				// Continue to login prompt
			} else {
				// Other API errors (network, server issues, etc.)
				return "", errors.Wrap(err, "failed to validate token")
			}
		} else {
			// Token is valid, return it
			return token, nil
		}
	}

	// Run login command (if no token or token was expired/invalid)
	err = login.RunLogin(login.LoginCmd, []string{})
	if err != nil {
		return "", errors.Wrap(err, "login failed")
	}

	token, err = config.GetToken()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve token after login")
	}

	// Validate the newly obtained token with retry
	_, err = dataaccess.DescribeUser(context.Background(), token)
	if err != nil {
		// Check if error is due to token expiry or authentication issues
		if errors.Is(err, config.ErrTokenExpired) || errors.Is(err, config.ErrUnauthorized) {
			// Token is still invalid after login, retry
			_ = config.RemoveAuthConfig()
			return getTokenWithRetry(retryCount + 1)
		}
		// Other API errors (network, server issues, etc.)
		return "", errors.Wrap(err, "failed to validate token after login")
	}

	return token, nil
}
