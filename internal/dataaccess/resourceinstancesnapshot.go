package dataaccess

import (
	"context"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

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

func ListResourceInstanceSnapshots(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.FleetListInstanceSnapshotResult, err error) {
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

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func DescribeResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, instanceID, snapshotID string) (res *openapiclientfleet.FleetDescribeInstanceSnapshotResult, err error) {
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

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func CopyResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, instanceID, sourceSnapshotID, targetRegion string) (res *openapiclientfleet.FleetCopyResourceInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	reqBody := openapiclientfleet.FleetCopyResourceInstanceSnapshotRequest2{
		TargetRegion: targetRegion,
	}

	if sourceSnapshotID != "" {
		reqBody.SetSourceSnapshotId(sourceSnapshotID)
	}

	req := apiClient.InventoryApiAPI.InventoryApiCopyResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetCopyResourceInstanceSnapshotRequest2(reqBody)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func DeleteResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, snapshotID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDeleteResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		snapshotID,
	)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func RestoreResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, snapshotID string, formattedParams map[string]any, tierVersionOverride string, networkType string) (res *openapiclientfleet.FleetRestoreResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	if networkType == "" {
		networkType = "PUBLIC"
	}

	reqBody := openapiclientfleet.FleetRestoreResourceInstanceFromSnapshotRequest2{
		InputParametersOverride: formattedParams,
		NetworkType:             utils.ToPtr(networkType),
	}

	if tierVersionOverride != "" {
		reqBody.ProductTierVersionOverride = &tierVersionOverride
	}

	req := apiClient.InventoryApiAPI.InventoryApiRestoreResourceInstanceFromSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		snapshotID,
	).FleetRestoreResourceInstanceFromSnapshotRequest2(reqBody)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func TriggerResourceInstanceAutoBackup(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.FleetAutomaticInstanceSnapshotCreationResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiTriggerAutomaticResourceInstanceSnapshotCreation(
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

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}
