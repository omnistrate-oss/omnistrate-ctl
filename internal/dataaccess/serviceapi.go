package dataaccess

import (
	"context"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// CreateServiceAPI creates a new service API for a service environment
func CreateServiceAPI(ctx context.Context, token, serviceID, serviceEnvironmentID, description string) (serviceAPIID string, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	req := openapiclient.CreateServiceAPIRequest2{
		Description:          description,
		ServiceEnvironmentId: serviceEnvironmentID,
	}

	resp, r, err := apiClient.ServiceApiApiAPI.ServiceApiApiCreateServiceAPI(ctxWithToken, serviceID).
		CreateServiceAPIRequest2(req).Execute()

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

func ListServiceAPIs(ctx context.Context, token, serviceID, serviceEnvironmentID string) (*openapiclient.ListServiceAPIsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiApiAPI.ServiceApiApiListServiceAPI(ctxWithToken, serviceID, serviceEnvironmentID).Execute()

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

func DescribeServiceAPI(ctx context.Context, token, serviceID, serviceAPIID string) (*openapiclient.DescribeServiceAPIResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiApiAPI.ServiceApiApiDescribeServiceAPI(ctxWithToken, serviceID, serviceAPIID).Execute()

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

func DeleteServiceAPI(ctx context.Context, token, serviceID, serviceAPIID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	r, err := apiClient.ServiceApiApiAPI.ServiceApiApiDeleteServiceAPI(ctxWithToken, serviceID, serviceAPIID).Execute()

	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	err = handleV1Error(err)
	if err != nil {
		return err
	}

	return nil
}
