package dataaccess

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func LoginWithPassword(ctx context.Context, email string, pass string) (string, error) {
	request := *openapiclient.NewSigninRequest(email)
	request.Password = utils.ToPtr(pass)

	apiClient := getV1Client()
	resp, r, err := apiClient.SigninApiAPI.SigninApiSignin(ctx).SigninRequest(request).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	r.Body.Close()
	return resp.JwtToken, nil
}

func LoginWithIdentityProvider(ctx context.Context, deviceCode, identityProviderName string) (string, error) {
	request := *openapiclient.NewLoginWithIdentityProviderRequest(identityProviderName)
	request.DeviceCode = utils.ToPtr(deviceCode)

	apiClient := getV1Client()
	resp, r, err := apiClient.SigninApiAPI.SigninApiLoginWithIdentityProvider(ctx).LoginWithIdentityProviderRequest(request).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	r.Body.Close()
	return resp.JwtToken, nil
}
