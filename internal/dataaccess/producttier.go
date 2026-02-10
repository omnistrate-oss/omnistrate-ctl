package dataaccess

import (
	"context"

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

// CreateProductTier creates a new product tier
// Note: AccountConfigIDs parameter is included for API compatibility but not yet supported
// in the current SDK CreateProductTierRequest2 schema. Account configs can be associated
// with the product tier through other APIs after creation if needed.
// ServiceModelId is left empty as it should be provided through separate service model creation.
func CreateProductTier(ctx context.Context, token, serviceID, name, description string, tierType string, accountConfigIDs []string) (productTierID string, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	// Note: CreateProductTierRequest2 requires these fields to be set
	// Using empty strings as defaults if not provided
	req := openapiclientv1.CreateProductTierRequest2{
		Name:            name,
		Description:     description,
		PlanDescription: description, // Using same as description
		ServiceModelId:  "",          // Should be provided through separate service model creation
		TierType:        tierType,
	}

	productTierID, r, err := apiClient.ProductTierApiAPI.ProductTierApiCreateProductTier(ctxWithToken, serviceID).
		CreateProductTierRequest2(req).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return "", handleV1Error(err)
	}

	return productTierID, nil
}

// ListProductTiers lists all product tiers for a service
// Note: The current implementation may need adjustment based on the actual API behavior
// as ProductTierApiListProductTier typically expects specific tier IDs
func ListProductTiers(ctx context.Context, token, serviceID, serviceModelID string) (*openapiclientv1.ListProductTiersResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	// Note: This API call may need to be updated to use a different endpoint
	// for listing all product tiers rather than querying a specific tier
	resp, r, err := apiClient.ProductTierApiAPI.ProductTierApiListProductTier(ctxWithToken, serviceID, "").
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
