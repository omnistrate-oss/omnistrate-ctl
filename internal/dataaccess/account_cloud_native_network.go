package dataaccess

import (
	"context"

	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// SyncAccountConfigCloudNativeNetworks triggers cloud-native network discovery for an account configuration.
func SyncAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiSyncAccountConfigCloudNativeNetworks(
		ctxWithToken,
		accountConfigID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// ListAccountConfigCloudNativeNetworks lists registered cloud-native networks for an account configuration.
func ListAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiListAccountConfigCloudNativeNetworks(
		ctxWithToken,
		accountConfigID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// ImportAccountConfigCloudNativeNetwork marks a cloud-native network as READY for deployments.
func ImportAccountConfigCloudNativeNetwork(ctx context.Context, token, accountConfigID, networkID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiImportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		networkID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// UnimportAccountConfigCloudNativeNetwork reverts a cloud-native network back to AVAILABLE.
func UnimportAccountConfigCloudNativeNetwork(ctx context.Context, token, accountConfigID, networkID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiUnimportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		networkID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// BulkImportAccountConfigCloudNativeNetworks imports multiple cloud-native networks in a single request.
func BulkImportAccountConfigCloudNativeNetworks(ctx context.Context, token, accountConfigID string, networkIDs []string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	ops := make([]openapiclientv1.AccountConfigCloudNativeNetworkOperation, len(networkIDs))
	for i, id := range networkIDs {
		ops[i] = openapiclientv1.AccountConfigCloudNativeNetworkOperation{
			CloudNativeNetworkId: id,
			Import:               true,
		}
	}
	body := openapiclientv1.BulkImportAccountConfigCloudNativeNetworksRequest2{
		CloudNativeNetworks: ops,
	}

	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiBulkImportAccountConfigCloudNativeNetworks(
		ctxWithToken,
		accountConfigID,
	).BulkImportAccountConfigCloudNativeNetworksRequest2(body).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}
