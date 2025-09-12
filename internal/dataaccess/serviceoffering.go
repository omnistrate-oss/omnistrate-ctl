package dataaccess

import (
	"context"
	"net/http"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func ListServiceOfferings(ctx context.Context, token, orgID string) (inventory *openapiclientfleet.InventoryListServiceOfferingsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListServiceOfferings(ctxWithToken)
	req = req.OrgId(orgID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	inventory, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return inventory, nil
}

func DescribeServiceOfferingResource(ctx context.Context, token, serviceID, resourceID, instanceID, productTierID, productTierVersion string) (res *openapiclientfleet.InventoryDescribeServiceOfferingResourceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeServiceOfferingResource(ctxWithToken, serviceID, resourceID, instanceID)
	req = req.ProductTierId(productTierID)
	req = req.ProductTierVersion(productTierVersion)
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

	return res, nil
}

func DescribeServiceOffering(ctx context.Context, token, serviceID, productTierID, productTierVersion string) (res *openapiclientfleet.InventoryDescribeServiceOfferingResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeServiceOffering(ctxWithToken, serviceID)
	if productTierID != "" {
		req = req.ProductTierId(productTierID)
	}
	if productTierVersion != "" {
		req = req.ProductTierVersion(productTierVersion)
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

	return res, nil
}

func DescribeServiceOfferingResourceV1(ctx context.Context, token, serviceID, resourceID, instanceID, productTierID, productTierVersion string) (res *openapiclientv1.DescribeServiceOfferingResourceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	req := apiClient.ServiceOfferingApiAPI.ServiceOfferingApiDescribeServiceOfferingResource(ctxWithToken, serviceID, resourceID, instanceID)
	if productTierID != "" {
		req = req.ProductTierId(productTierID)
	}
	if productTierVersion != "" {
		req = req.ProductTierVersion(productTierVersion)
	}
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	return res, nil
}

func ListInputParameters(ctx context.Context, token, serviceID, resourceID, productTierID, productTierVersion string) (res *openapiclientv1.ListInputParametersResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	req := apiClient.InputParameterApiAPI.InputParameterApiListInputParameter(ctxWithToken, serviceID, resourceID)
	if productTierID != "" {
		req = req.ProductTierId(productTierID)
	}
	if productTierVersion != "" {
		req = req.ProductTierVersion(productTierVersion)
	}
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	return res, nil
}
