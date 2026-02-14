package dataaccess

import (
	"context"
	"net/http"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func DescribeServiceModel(ctx context.Context, token, serviceID, serviceModelID string) (serviceModel *openapiclient.DescribeServiceModelResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err := apiClient.ServiceModelApiAPI.ServiceModelApiDescribeServiceModel(ctxWithToken, serviceID, serviceModelID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func EnableServiceModelFeature(ctx context.Context, token, serviceID, serviceModelID, featureName string, featureConfiguration map[string]any) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	req := apiClient.ServiceModelApiAPI.ServiceModelApiEnableServiceModelFeature(ctxWithToken, serviceID, serviceModelID)
	req = req.EnableServiceModelFeatureRequest2(openapiclient.EnableServiceModelFeatureRequest2{
		Feature:       featureName,
		Configuration: featureConfiguration,
	})

	r, err = req.Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}
	return
}

func DisableServiceModelFeature(ctx context.Context, token, serviceID, serviceModelID, featureName string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	req := apiClient.ServiceModelApiAPI.ServiceModelApiDisableServiceModelFeature(ctxWithToken, serviceID, serviceModelID)
	req = req.DisableServiceModelFeatureRequest2(openapiclient.DisableServiceModelFeatureRequest2{
		Feature: featureName,
	})

	r, err = req.Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}

	return
}

// CreateServiceModel creates a new service model with model type and account config IDs
// serviceApiID: the service API ID this model belongs to
// modelType: CUSTOMER_HOSTED, OMNISTRATE_HOSTED, OMNISTRATE_DEDICATED, etc.
func CreateServiceModel(ctx context.Context, token, serviceID, serviceApiID, name, description, modelType string, accountConfigIDs []string) (serviceModelID string, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	req := openapiclient.CreateServiceModelRequest2{
		Name:         name,
		Description:  description,
		ServiceApiId: serviceApiID,
	}

	// Set model type if provided
	if modelType != "" {
		req.ModelType = modelType
	}

	// Set account config IDs if provided
	if len(accountConfigIDs) > 0 {
		req.AccountConfigIds = accountConfigIDs
	}

	resp, r, err := apiClient.ServiceModelApiAPI.ServiceModelApiCreateServiceModel(ctxWithToken, serviceID).
		CreateServiceModelRequest2(req).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	// Clean up the response ID (remove surrounding quotes and newlines)
	return strings.Trim(resp, "\"\n\t "), nil
}

func DeleteServiceModel(ctx context.Context, token, serviceID, serviceModelID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err := apiClient.ServiceModelApiAPI.ServiceModelApiDeleteServiceModel(ctxWithToken, serviceID, serviceModelID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}

	return nil
}
