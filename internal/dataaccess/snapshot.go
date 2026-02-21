package dataaccess

import (
	"context"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

// ListAllSnapshots lists all snapshots for a service environment (no instance ID required).
func ListAllSnapshots(ctx context.Context, token, serviceID, environmentID string) (res *openapiclientfleet.FleetListInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListAllResourceInstanceSnapshots(
		ctxWithToken,
		serviceID,
		environmentID,
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

// DeleteSnapshot deletes a snapshot (no instance ID required).
func DeleteSnapshot(ctx context.Context, token, serviceID, environmentID, snapshotID string) (err error) {
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

// RestoreSnapshot restores a snapshot to a new instance (no instance ID required).
func RestoreSnapshot(ctx context.Context, token, serviceID, environmentID, snapshotID string, formattedParams map[string]any, tierVersionOverride string, networkType string) (res *openapiclientfleet.FleetRestoreResourceInstanceResult, err error) {
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
