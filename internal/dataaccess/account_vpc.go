package dataaccess

import (
	"context"

	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// SyncAccountConfigVPCs triggers cloud-native network discovery for an account configuration.
func SyncAccountConfigVPCs(ctx context.Context, token, accountConfigID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
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

// ListAccountConfigVPCs lists registered cloud-native networks for an account configuration.
func ListAccountConfigVPCs(ctx context.Context, token, accountConfigID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
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

// ImportAccountConfigVPC marks a cloud-native network as READY for deployments.
func ImportAccountConfigVPC(ctx context.Context, token, accountConfigID, vpcID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiImportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		vpcID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// UnimportAccountConfigVPC reverts a cloud-native network back to AVAILABLE.
func UnimportAccountConfigVPC(ctx context.Context, token, accountConfigID, vpcID string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiUnimportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		vpcID,
	).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	r.Body.Close()
	return res, nil
}

// BulkImportAccountConfigVPCs imports multiple cloud-native networks in a single request.
func BulkImportAccountConfigVPCs(ctx context.Context, token, accountConfigID string, vpcIDs []string) (*openapiclientv1.ListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	ops := make([]openapiclientv1.AccountConfigCloudNativeNetworkOperation, len(vpcIDs))
	for i, id := range vpcIDs {
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
