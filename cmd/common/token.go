package common

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/pkg/errors"
)

func GetTokenWithLogin() (token string, err error) {
	token, err = config.GetToken()
	if err != nil && !errors.Is(err, config.ErrAuthConfigNotFound) && !errors.Is(err, config.ErrConfigFileNotFound) {
		return
	}

	// If token is present, validate it by calling the user API
	if token != "" {
		// Validate token by making an API call
		_, err = dataaccess.DescribeUser(context.Background(), token)
		if err != nil {
			// Token is invalid, remove it and prompt for login
			_ = config.RemoveAuthConfig()
			token = ""
		} else {
			// Token is valid, return it
			return
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
