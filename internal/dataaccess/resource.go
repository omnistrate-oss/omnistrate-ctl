package dataaccess

import (
	"context"
	"net/http"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func DescribeResource(ctx context.Context, token, serviceID, resourceID string, productTierID, productTierVersion *string) (resp *openapiclient.DescribeResourceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	req := apiClient.ResourceApiAPI.ResourceApiDescribeResource(
		ctxWithToken,
		serviceID,
		resourceID,
	)
	if productTierID != nil {
		req = req.ProductTierId(*productTierID)
	}
	if productTierVersion != nil {
		req = req.ProductTierVersion(*productTierVersion)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return
}

func ListResources(ctx context.Context, token, serviceID string, productTierID string, productTierVersion *string) (resp *openapiclient.ListResourcesResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	req := apiClient.ResourceApiAPI.ResourceApiListResources(
		ctxWithToken,
		serviceID,
		productTierID,
	)
	if productTierVersion != nil {
		req = req.ProductTierVersion(*productTierVersion)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return
}
