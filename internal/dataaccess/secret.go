package dataaccess

import (
	"context"

	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// SetSecret sets a secret for the given environment type
func SetSecret(ctx context.Context, token, environmentType, name, value string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.SecretsApiAPI.SecretsApiSetSecret(ctxWithToken, environmentType, name).
		SetSecretRequest2(openapiclientv1.SetSecretRequest2{
			Value: value,
		}).Execute()
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

// GetSecret retrieves a secret for the given environment type and name
func GetSecret(ctx context.Context, token, environmentType, name string) (*openapiclientv1.GetSecretResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.SecretsApiAPI.SecretsApiGetSecret(ctxWithToken, environmentType, name).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return resp, nil
}

// ListSecrets lists all secrets for the given environment type
func ListSecrets(ctx context.Context, token, environmentType string) (*openapiclientv1.ListSecretsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.SecretsApiAPI.SecretsApiListSecrets(ctxWithToken, environmentType).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return resp, nil
}

// DeleteSecret deletes a secret for the given environment type and name
func DeleteSecret(ctx context.Context, token, environmentType, name string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.SecretsApiAPI.SecretsApiDeleteSecret(ctxWithToken, environmentType, name).Execute()
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
