package dataaccess

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// APIKeySigninEmail is the RFC 5321-shaped sentinel email value the
// signin endpoint recognizes to route a request into the api-key
// signin-exchange path. Mirrors security.APIKeySigninEmailFull in
// omnistrate/commons; redefined here because ctl is a separate module
// and cannot import the platform-internal commons package. Keep both
// constants in sync when touching either side.
const APIKeySigninEmail = "apikey@apikeys.invalid"

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

// LoginWithAPIKey exchanges an org-bounded API key plaintext for a JWT
// session. It uses the same signin endpoint as password login, with
// the sentinel APIKeySigninEmail in the email field and the API key
// plaintext in the password field; the platform recognizes the
// sentinel and routes the request into the api-key signin-exchange
// path. The returned session is bound to the api-key's backing user
// and inherits the key's role; the JWT MUST NOT be persisted any
// longer than the calling session needs it.
func LoginWithAPIKey(ctx context.Context, apiKey string) (LoginResult, error) {
	return LoginWithPassword(ctx, APIKeySigninEmail, apiKey)
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
