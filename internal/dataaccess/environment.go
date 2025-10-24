package dataaccess

import (
	"context"
	"net/http"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found")
)

func CreateServiceEnvironment(ctx context.Context,
	token string,
	name, description, serviceID string,
	visibility, environmentType string,
	sourceEnvID *string,
	deploymentConfigID string,
	autoApproveSubscription bool,
	serviceAuthPublicKey *string,
) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiCreateServiceEnvironment(ctxWithToken, serviceID).
		CreateServiceEnvironmentRequest2(openapiclientv1.CreateServiceEnvironmentRequest2{
			Name:                    name,
			Description:             description,
			Visibility:              utils.ToPtr(visibility),
			Type:                    utils.ToPtr(environmentType),
			SourceEnvironmentId:     sourceEnvID,
			DeploymentConfigId:      deploymentConfigID,
			AutoApproveSubscription: utils.ToPtr(autoApproveSubscription),
			ServiceAuthPublicKey:    serviceAuthPublicKey,
		}).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return "", handleV1Error(err)
	}
	return cleanupId(resp), nil // remove surrounding quotes and newlines
}

func cleanupId(resp string) string {
	return strings.Trim(resp, "\"\n\t ")
}

func DescribeServiceEnvironment(ctx context.Context, token, serviceID, serviceEnvironmentID string) (*openapiclientv1.DescribeServiceEnvironmentResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiDescribeServiceEnvironment(ctxWithToken, serviceID, serviceEnvironmentID).Execute()
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

func ListServiceEnvironments(ctx context.Context, token, serviceID string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiListServiceEnvironment(ctxWithToken, serviceID).Execute()
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

func PromoteServiceEnvironment(ctx context.Context, token, serviceID, serviceEnvironmentID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiPromoteServiceEnvironment(ctxWithToken, serviceID, serviceEnvironmentID).
		PromoteServiceEnvironmentRequest2(
			openapiclientv1.PromoteServiceEnvironmentRequest2{},
		).Execute()

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


func PromoteServiceEnvironmentStatus(ctx context.Context, token, serviceID, serviceEnvironmentID string) (resp []openapiclientv1.EnvironmentPromotionStatus, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	var r *http.Response
	resp, r, err = apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiPromoteServiceEnvironmentStatus(ctxWithToken, serviceID, serviceEnvironmentID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	err = handleV1Error(err)
	if err != nil {
		return
	}
	return resp, nil
}

func DeleteServiceEnvironment(ctx context.Context, token, serviceID, serviceEnvironmentID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.ServiceEnvironmentApiAPI.ServiceEnvironmentApiDeleteServiceEnvironment(ctxWithToken, serviceID, serviceEnvironmentID).Execute()
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

func FindEnvironment(ctx context.Context, token, serviceID, environmentType string) (*openapiclientv1.DescribeServiceEnvironmentResult, error) {
	listRes, err := ListServiceEnvironments(ctx, token, serviceID)
	if err != nil {
		return nil, err
	}

	for _, id := range listRes.Ids {
		descRes, err := DescribeServiceEnvironment(ctx, token, serviceID, id)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(descRes.GetType(), environmentType) {
			return descRes, nil
		}
	}

	return nil, ErrEnvironmentNotFound
}
