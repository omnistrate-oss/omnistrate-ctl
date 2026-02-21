package dataaccess

import (
	"context"
	"net/http"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func SearchInventory(ctx context.Context, token, query string) (*openapiclientfleet.SearchInventoryResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	req := openapiclientfleet.SearchInventoryRequest2{
		Query: query,
	}

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.
		InventoryApiSearchInventory(ctxWithToken).
		SearchInventoryRequest2(req).
		Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func ListInstanceSnapshots(ctx context.Context, token, serviceID, environmentID, instanceID string) (resp *openapiclientfleet.FleetListInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListResourceInstanceSnapshots(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func DescribeInstanceSnapshot(ctx context.Context, token, serviceID, environmentID, instanceID, snapshotID string) (resp *openapiclientfleet.FleetDescribeInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
		snapshotID,
	)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func CreateInstanceSnapshot(ctx context.Context, token, serviceID, environmentID, instanceID, targetRegion string) (resp *openapiclientfleet.FleetCreateInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiCreateResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

	reqBody := *openapiclientfleet.NewFleetCreateInstanceSnapshotRequest2()
	if targetRegion != "" {
		reqBody.SetTargetRegion(targetRegion)
	}
	req = req.FleetCreateInstanceSnapshotRequest2(reqBody)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}
