package dataaccess

import (
	"context"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

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

func CreateInstanceSnapshot(ctx context.Context, token, serviceID, environmentID, instanceID string) (resp *openapiclientfleet.FleetCreateInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiCreateResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetCreateInstanceSnapshotRequest2(*openapiclientfleet.NewFleetCreateInstanceSnapshotRequest2())

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

// ListAllSnapshotsOptions controls optional filters for service environment snapshot listing.
type ListAllSnapshotsOptions struct {
	ProductTierID string
	SnapshotType  string
}

// ListAllSnapshots lists snapshots for a service environment (no instance ID required).
func ListAllSnapshots(ctx context.Context, token, serviceID, environmentID string, opts ListAllSnapshotsOptions) (res *openapiclientfleet.FleetListInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListAllResourceInstanceSnapshots(
		ctxWithToken,
		serviceID,
		environmentID,
	)
	if opts.ProductTierID != "" {
		req = req.ProductTierId(opts.ProductTierID)
	}
	if opts.SnapshotType != "" {
		req = req.SnapshotType(opts.SnapshotType)
	}

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

// RestoreSnapshotOptions controls optional restore behavior.
type RestoreSnapshotOptions struct {
	InputParametersOverride    map[string]any
	ProductTierVersionOverride string
	NetworkType                string
	CustomNetworkID            string
	SubscriptionID             string
	RestoreToSource            bool
}

// RestoreSnapshot restores a snapshot either to a new instance or, when RestoreToSource is true, to the original source instance.
func RestoreSnapshot(ctx context.Context, token, serviceID, environmentID, snapshotID string, opts RestoreSnapshotOptions) (res *openapiclientfleet.FleetRestoreResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	networkType := opts.NetworkType
	if networkType == "" {
		networkType = "PUBLIC"
	}

	reqBody := buildRestoreSnapshotRequestBody(opts)
	reqBody.NetworkType = utils.ToPtr(networkType)

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

func buildRestoreSnapshotRequestBody(opts RestoreSnapshotOptions) openapiclientfleet.FleetRestoreResourceInstanceFromSnapshotRequest2 {
	reqBody := openapiclientfleet.FleetRestoreResourceInstanceFromSnapshotRequest2{
		InputParametersOverride: opts.InputParametersOverride,
	}

	if opts.ProductTierVersionOverride != "" {
		reqBody.ProductTierVersionOverride = &opts.ProductTierVersionOverride
	}

	if opts.CustomNetworkID != "" {
		reqBody.CustomNetworkId = &opts.CustomNetworkID
	}

	if opts.SubscriptionID != "" {
		reqBody.SubscriptionId = &opts.SubscriptionID
	}

	if opts.RestoreToSource {
		reqBody.RestoreToSourceInstance = utils.ToPtr(true)
	}

	return reqBody
}
