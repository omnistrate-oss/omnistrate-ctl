package dataaccess

import (
	"context"
	"time"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// CreateAPIKey provisions a new org-bounded API key with the supplied
// display name and role and returns the plaintext value (returned by
// the platform exactly once) together with the persisted metadata.
//
// description and expiresAt are optional; pass nil when they should be
// omitted from the request.
func CreateAPIKey(
	ctx context.Context,
	token string,
	name string,
	roleType string,
	description *string,
	expiresAt *time.Time,
) (*openapiclient.CreateAPIKeyResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	req := openapiclient.NewCreateAPIKeyRequest2(name, roleType)
	if description != nil {
		req.Description = description
	}
	if expiresAt != nil {
		req.ExpiresAt = expiresAt
	}

	apiClient := getV1Client()
	res, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiCreateAPIKey(ctxWithToken).
		CreateAPIKeyRequest2(*req).
		Execute()
	if r != nil {
		defer r.Body.Close()
	}
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	return res, nil
}

// ListAPIKeys returns the metadata for every API key visible to the
// caller's org. Plaintext is never included.
func ListAPIKeys(ctx context.Context, token string) (*openapiclient.ListAPIKeysResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiListAPIKeys(ctxWithToken).Execute()
	if r != nil {
		defer r.Body.Close()
	}
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	return res, nil
}

// DescribeAPIKey returns the metadata for the API key identified by id.
func DescribeAPIKey(ctx context.Context, token, id string) (*openapiclient.DescribeAPIKeyResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiDescribeAPIKey(ctxWithToken, id).Execute()
	if r != nil {
		defer r.Body.Close()
	}
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	return res, nil
}

// UpdateAPIKeyMetadata edits the mutable metadata (name, description)
// of the key identified by id. Either pointer may be nil to leave the
// corresponding field unchanged.
func UpdateAPIKeyMetadata(
	ctx context.Context,
	token string,
	id string,
	name *string,
	description *string,
) (*openapiclient.UpdateAPIKeyMetadataResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	req := openapiclient.NewUpdateAPIKeyMetadataRequest2()
	if name != nil {
		req.Name = name
	}
	if description != nil {
		req.Description = description
	}

	apiClient := getV1Client()
	res, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiUpdateAPIKeyMetadata(ctxWithToken, id).
		UpdateAPIKeyMetadataRequest2(*req).
		Execute()
	if r != nil {
		defer r.Body.Close()
	}
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	return res, nil
}

// RevokeAPIKey marks the key identified by id as revoked. The key
// remains in the org listing with status=revoked until DeleteAPIKey
// is called.
func RevokeAPIKey(ctx context.Context, token, id string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	_, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiRevokeAPIKey(ctxWithToken, id).Execute()
	if r != nil {
		defer r.Body.Close()
	}
	return handleV1Error(err)
}

// DeleteAPIKey permanently removes the key identified by id and its
// backing user. This is irreversible.
func DeleteAPIKey(ctx context.Context, token, id string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	_, r, err := apiClient.ApiKeyApiAPI.ApiKeyApiDeleteAPIKey(ctxWithToken, id).Execute()
	if r != nil {
		defer r.Body.Close()
	}
	return handleV1Error(err)
}
