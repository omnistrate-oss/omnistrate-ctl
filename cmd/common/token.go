package common

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/pkg/errors"
)

const (
	maxTokenRetries     = 3
	loginInstructionMsg = "Run: omnistrate-ctl login"
)

// isStdinPiped returns true when stdin is not a terminal (piped/redirected).
// This happens in MCP servers, CI/CD, shell pipes, and automation.
func isStdinPiped() bool {
	return !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// GetToken gets and validates auth token without prompting for login.
func GetToken() (string, error) {
	token, err := config.GetToken()
	if err != nil {
		if errors.Is(err, config.ErrAuthConfigNotFound) || errors.Is(err, config.ErrConfigFileNotFound) {
			return "", fmt.Errorf("authentication required: not logged in. %s", loginInstructionMsg)
		}
		return "", errors.Wrap(err, "failed to retrieve authentication token")
	}

	if token == "" {
		return "", fmt.Errorf("authentication required: not logged in. %s", loginInstructionMsg)
	}

	// Validate token with API call
	_, err = dataaccess.DescribeUser(context.Background(), token)
	if err != nil {
		if errors.Is(err, config.ErrTokenExpired) {
			return "", fmt.Errorf("authentication expired: token has expired. %s", loginInstructionMsg)
		}
		if errors.Is(err, config.ErrUnauthorized) {
			return "", fmt.Errorf("authentication failed: unauthorized access. %s", loginInstructionMsg)
		}
		return "", errors.Wrap(err, "failed to validate token")
	}

	return token, nil
}

// GetTokenWithLogin gets auth token, prompting for login only in interactive mode.
// In non-interactive contexts (MCP, automation), fails immediately to prevent hanging.
func GetTokenWithLogin() (string, error) {
	if isStdinPiped() {
		return GetToken()
	}
	return getTokenWithRetry(0)
}

func getTokenWithRetry(retryCount int) (string, error) {
	if retryCount >= maxTokenRetries {
		return "", fmt.Errorf("maximum token validation retries (%d) exceeded, please try again later", maxTokenRetries)
	}

	token, err := config.GetToken()
	if err != nil && !errors.Is(err, config.ErrAuthConfigNotFound) && !errors.Is(err, config.ErrConfigFileNotFound) {
		return "", errors.Wrap(err, "failed to retrieve authentication token")
	}

	// Validate existing token
	if token != "" {
		_, err = dataaccess.DescribeUser(context.Background(), token)
		if err != nil {
			if errors.Is(err, config.ErrTokenExpired) || errors.Is(err, config.ErrUnauthorized) {
				_ = config.RemoveAuthConfig()
				token = ""
			} else {
				return "", errors.Wrap(err, "failed to validate token")
			}
		} else {
			return token, nil
		}
	}

	// Prompt for login
	err = login.RunLogin(login.LoginCmd, []string{})
	if err != nil {
		return "", errors.Wrap(err, "login failed")
	}

	token, err = config.GetToken()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve token after login")
	}

	// Validate new token
	_, err = dataaccess.DescribeUser(context.Background(), token)
	if err != nil {
		if errors.Is(err, config.ErrTokenExpired) || errors.Is(err, config.ErrUnauthorized) {
			_ = config.RemoveAuthConfig()
			return getTokenWithRetry(retryCount + 1)
		}
		return "", errors.Wrap(err, "failed to validate token after login")
	}

	return token, nil
}
