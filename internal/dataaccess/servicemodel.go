package dataaccess

import (
	"context"
	"net/http"

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

// CreateServiceModel creates a new service model
func CreateServiceModel(ctx context.Context, token, serviceID, name, description string) (serviceModelID string, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	// Use empty string as default for description if not provided
	if description == "" {
		description = "Created by omnistrate-ctl"
	}

	req := openapiclient.CreateServiceModelRequest2{
		Name:        name,
		Description: description,
	}

	serviceModelID, r, err := apiClient.ServiceModelApiAPI.ServiceModelApiCreateServiceModel(ctxWithToken, serviceID).
		CreateServiceModelRequest2(req).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return "", handleV1Error(err)
	}

	return serviceModelID, nil
}

// DeleteServiceModel deletes a service model
func DeleteServiceModel(ctx context.Context, token, serviceID, serviceModelID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	r, err := apiClient.ServiceModelApiAPI.ServiceModelApiDeleteServiceModel(ctxWithToken, serviceID, serviceModelID).Execute()
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
