package dataaccess

import (
	"context"
	"net/http"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func CreateServicesOrchestration(
	ctx context.Context,
	token string,
	orchestrationCreateDSL string,
) (
	res *openapiclientfleet.FleetCreateServicesOrchestrationResult,
	err error,
) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	request := openapiclientfleet.FleetCreateServicesOrchestrationRequest2{
		OrchestrationCreateDSL: orchestrationCreateDSL,
	}

	req := apiClient.InventoryApiAPI.InventoryApiCreateServicesOrchestration(
		ctxWithToken,
	).FleetCreateServicesOrchestrationRequest2(request)

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

func DeleteServicesOrchestration(ctx context.Context, token, id string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDeleteServicesOrchestration(
		ctxWithToken,
		id,
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

func DescribeServicesOrchestration(ctx context.Context, token string, id string) (resp *openapiclientfleet.FleetDescribeServicesOrchestrationResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeServicesOrchestration(
		ctxWithToken,
		id,
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

func ListServicesOrchestration(ctx context.Context, token string, environmentType string) (resp []openapiclientfleet.FleetDescribeServicesOrchestrationResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListServicesOrchestrations(
		ctxWithToken,
	)
	req = req.EnvironmentType(strings.ToUpper(environmentType))

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

func ModifyServicesOrchestration(
	ctx context.Context,
	token string,
	id string,
	orchestrationModifyDSL string,
) (
	err error,
) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	request := openapiclientfleet.FleetModifyServicesOrchestrationRequest2{
		OrchestrationModifyDSL: orchestrationModifyDSL,
	}

	req := apiClient.InventoryApiAPI.InventoryApiModifyServicesOrchestration(
		ctxWithToken,
		id,
	).FleetModifyServicesOrchestrationRequest2(request)

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
