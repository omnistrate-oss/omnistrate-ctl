package dataaccess

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// LoginResult holds both the JWT and optional refresh token from a login response.
type LoginResult struct {
	JWTToken     string //nolint:gosec
	RefreshToken string //nolint:gosec
}

func LoginWithPassword(ctx context.Context, email string, pass string) (LoginResult, error) {
	request := *openapiclient.NewSigninRequest(email)
	request.Password = utils.ToPtr(pass)

	apiClient := getV1Client()
	resp, r, err := apiClient.SigninApiAPI.SigninApiSignin(ctx).SigninRequest(request).Execute()

	if r != nil {
		defer r.Body.Close()
	}

	err = handleV1Error(err)
	if err != nil {
		return LoginResult{}, err
	}

	result := LoginResult{JWTToken: resp.JwtToken, RefreshToken: utils.FromPtr(resp.RefreshToken)}
	return result, nil
}

func LoginWithIdentityProvider(ctx context.Context, deviceCode, identityProviderName string) (LoginResult, error) {
	request := *openapiclient.NewLoginWithIdentityProviderRequest(identityProviderName)
	request.DeviceCode = utils.ToPtr(deviceCode)

	apiClient := getV1Client()
	resp, r, err := apiClient.SigninApiAPI.SigninApiLoginWithIdentityProvider(ctx).LoginWithIdentityProviderRequest(request).Execute()

	if r != nil {
		defer r.Body.Close()
	}

	err = handleV1Error(err)
	if err != nil {
		return LoginResult{}, err
	}

	result := LoginResult{JWTToken: resp.JwtToken, RefreshToken: utils.FromPtr(resp.RefreshToken)}
	return result, nil
}
