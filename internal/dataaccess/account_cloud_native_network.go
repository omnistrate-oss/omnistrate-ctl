package dataaccess

import (
	"context"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type CloudNativeNetworkTarget struct {
	Region    string
	NetworkID string
}

func SyncAccountConfigCloudNativeNetworks(
	ctx context.Context,
	token string,
	accountConfigID string,
	targets []CloudNativeNetworkTarget,
) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	apiTargets := make([]openapiclientfleet.FleetSyncAccountConfigCloudNativeNetworkTarget, 0, len(targets))
	for _, target := range targets {
		apiTarget := openapiclientfleet.FleetSyncAccountConfigCloudNativeNetworkTarget{
			Region: target.Region,
		}
		if target.NetworkID != "" {
			apiTarget.CloudNativeNetworkId = &target.NetworkID
		}
		apiTargets = append(apiTargets, apiTarget)
	}

	request := openapiclientfleet.FleetSyncAccountConfigCloudNativeNetworksRequest2{
		CloudNativeNetworks: apiTargets,
	}

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.InventoryApiSyncAccountConfigCloudNativeNetworks(
		ctxWithToken,
		accountConfigID,
	).FleetSyncAccountConfigCloudNativeNetworksRequest2(request).Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func SyncAccountConfigCloudNativeNetworksByTarget(
	ctx context.Context,
	token string,
	accountConfigID string,
	targets []CloudNativeNetworkTarget,
) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	return SyncAccountConfigCloudNativeNetworks(ctx, token, accountConfigID, targets)
}

func ImportAccountConfigCloudNativeNetwork(
	ctx context.Context,
	token string,
	accountConfigID string,
	cloudNativeNetworkID string,
) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.InventoryApiImportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		cloudNativeNetworkID,
	).Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func BulkImportAccountConfigCloudNativeNetworks(
	ctx context.Context,
	token string,
	accountConfigID string,
	cloudNativeNetworkIDs []string,
) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	operations := make([]openapiclientfleet.FleetAccountConfigCloudNativeNetworkOperation, 0, len(cloudNativeNetworkIDs))
	for _, id := range cloudNativeNetworkIDs {
		operations = append(operations, openapiclientfleet.FleetAccountConfigCloudNativeNetworkOperation{
			CloudNativeNetworkId: id,
			Import:               true,
		})
	}

	request := openapiclientfleet.FleetBulkImportAccountConfigCloudNativeNetworksRequest2{
		CloudNativeNetworks: operations,
	}

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.InventoryApiBulkImportAccountConfigCloudNativeNetworks(
		ctxWithToken,
		accountConfigID,
	).FleetBulkImportAccountConfigCloudNativeNetworksRequest2(request).Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func UnimportAccountConfigCloudNativeNetwork(
	ctx context.Context,
	token string,
	accountConfigID string,
	cloudNativeNetworkID string,
) (*openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.InventoryApiUnimportAccountConfigCloudNativeNetwork(
		ctxWithToken,
		accountConfigID,
		cloudNativeNetworkID,
	).Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}
