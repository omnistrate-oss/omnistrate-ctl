package dataaccess

import (
	"context"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"net/http"
)

func FleetDescribeCustomNetwork(
	ctx context.Context, token string, id string) (
	customNetwork *openapiclientfleet.FleetCustomNetwork, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetCustomNetworkApiAPI.FleetCustomNetworkApiDescribeCustomNetwork(ctxWithToken, id)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	customNetwork, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return
}

func FleetUpdateCustomNetwork(
	ctx context.Context, token string, id string, updatedName *string) (
	customNetwork *openapiclientfleet.FleetCustomNetwork, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	updates := openapiclientfleet.FleetUpdateCustomNetworkRequest2{
		Name: updatedName,
	}

	req := apiClient.FleetCustomNetworkApiAPI.FleetCustomNetworkApiUpdateCustomNetwork(ctxWithToken, id).
		FleetUpdateCustomNetworkRequest2(updates)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	customNetwork, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return
}

func FleetListCustomNetworks(
	ctx context.Context, token string, cloudProviderName *string, cloudProviderRegion *string) (
	customNetworks *openapiclientfleet.FleetListCustomNetworksResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetCustomNetworkApiAPI.FleetCustomNetworkApiListCustomNetworks(ctxWithToken)
	if cloudProviderName != nil {
		req = req.CloudProviderName(*cloudProviderName)
	}
	if cloudProviderRegion != nil {
		req = req.CloudProviderRegion(*cloudProviderRegion)
	}

	req = req.CustomNetworksOnly(false)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	customNetworks, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return
}

func FleetCreateCustomNetwork(
	ctx context.Context, token string, cloudProviderName string, cloudProviderRegion string, cidr string, name *string) (
	customNetwork *openapiclientfleet.FleetCustomNetwork, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetCustomNetworkApiAPI.FleetCustomNetworkApiCreateCustomNetwork(ctxWithToken)
	reqNetwork := openapiclientfleet.FleetCreateCustomNetworkRequest2{
		Cidr:                cidr,
		CloudProviderName:   cloudProviderName,
		CloudProviderRegion: cloudProviderRegion,
		Name:                name,
	}
	req = req.FleetCreateCustomNetworkRequest2(reqNetwork)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	customNetwork, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return
}

func FleetDeleteCustomNetwork(
	ctx context.Context, token string, customNetworkId string) (
	err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.FleetCustomNetworkApiAPI.FleetCustomNetworkApiDeleteCustomNetwork(ctxWithToken, customNetworkId)

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
