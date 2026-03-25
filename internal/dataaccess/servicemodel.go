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
