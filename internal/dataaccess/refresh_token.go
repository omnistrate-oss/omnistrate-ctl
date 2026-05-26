package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// RefreshToken exchanges a refresh token for a new JWT + refresh token pair.
func RefreshToken(ctx context.Context, refreshToken string) (LoginResult, error) {
	request := *openapiclient.NewRefreshTokenRequest()
	request.SetRefreshToken(refreshToken)

	apiClient := getV1Client()
	resp, r, err := apiClient.SigninApiAPI.SigninApiRefreshToken(ctx).RefreshTokenRequest(request).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return LoginResult{}, handleV1Error(err)
	}

	return LoginResult{
		JWTToken:     resp.JwtToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}
