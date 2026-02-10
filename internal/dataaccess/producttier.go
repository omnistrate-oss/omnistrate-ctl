package dataaccess

import (
	"context"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func DeleteProductTier(ctx context.Context, token, serviceID, productTierID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)

	apiClient := getV1Client()
	r, err := apiClient.ProductTierApiAPI.ProductTierApiDeleteProductTier(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}

	r.Body.Close()
	return nil
}

// CreateProductTier creates a new product tier with tier type
// tierType: OMNISTRATE_MULTI_TENANCY, OMNISTRATE_DEDICATED_TENANCY, CUSTOM_TENANCY
// serviceModelID: the service model ID this product tier belongs to
func CreateProductTier(ctx context.Context, token, serviceID, serviceModelID, name, description string, tierType string) (productTierID string, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)

	apiClient := getV1Client()

	req := openapiclientv1.CreateProductTierRequest2{
		Name:           name,
		Description:    description,
		ServiceModelId: serviceModelID,
	}

	if tierType != "" {
		req.TierType = tierType
	}

	resp, r, err := apiClient.ProductTierApiAPI.ProductTierApiCreateProductTier(
		ctxWithToken,
		serviceID,
	).CreateProductTierRequest2(req).Execute()

	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	// Clean up the response ID (remove surrounding quotes and newlines)
	return strings.Trim(resp, "\"\n\t "), nil
}

func DescribeProductTier(ctx context.Context, token, serviceID, productTierID string) (productTier *openapiclientv1.DescribeProductTierResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.ProductTierApiAPI.ProductTierApiDescribeProductTier(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func ReleaseServicePlan(ctx context.Context, token, serviceID, serviceAPIID, productTierID string, versionSetName *string, isPreferred, dryrun bool) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.ServiceApiApiAPI.ServiceApiApiReleaseServiceAPI(ctxWithToken, serviceID, serviceAPIID).
		ReleaseServiceAPIRequest2(openapiclientv1.ReleaseServiceAPIRequest2{
			ProductTierId:  utils.ToPtr(productTierID),
			VersionSetName: versionSetName,
			VersionSetType: utils.ToPtr("Major"),
			IsPreferred:    utils.ToPtr(isPreferred),
			DryRun:         utils.ToPtr(dryrun),
		}).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleV1Error(err)
	}
	return nil
}

func DescribePendingChanges(ctx context.Context, token, serviceID, serviceAPIID, productTierID string) (*openapiclientv1.DescribePendingChangesResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiApiAPI.ServiceApiApiDescribePendingChanges(ctxWithToken, serviceID, serviceAPIID).
		ProductTierId(productTierID).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return resp, nil
}

func ListProductTiers(ctx context.Context, token, serviceID, serviceModelID string) (*openapiclientv1.ListProductTiersResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ProductTierApiAPI.ProductTierApiListProductTier(
		ctxWithToken,
		serviceID,
		serviceModelID,
	).Execute()

	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
