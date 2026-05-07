package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// RevokeToken invalidates the given refresh token server-side. The
// backend deletes the token hash so it can never be used again.
func RevokeToken(ctx context.Context, refreshToken string) error {
	request := *openapiclient.NewRevokeTokenRequest()
	request.SetRefreshToken(refreshToken)

	apiClient := getV1Client()
	r, err := apiClient.SigninApiAPI.SigninApiRevokeToken(ctx).RevokeTokenRequest(request).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleV1Error(err)
	}

	return nil
}
