package dataaccess

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	// Try the SDK approach first
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
	
	// If we get a "missing_payload" error, try custom HTTP request with empty JSON body
	if err != nil && (strings.Contains(err.Error(), "missing_payload") || strings.Contains(err.Error(), "missing required payload")) {
		return promoteServiceEnvironmentWithPayload(ctx, token, serviceID, serviceEnvironmentID)
	}
	
	if err != nil {
		return handleV1Error(err)
	}
	return nil
}

// Custom HTTP request function that sends an empty JSON payload
func promoteServiceEnvironmentWithPayload(ctx context.Context, token, serviceID, serviceEnvironmentID string) error {
	// Get the base URL from the API client configuration
	apiClient := getV1Client()
	config := apiClient.GetConfig()
	baseURL := strings.TrimSuffix(config.Servers[0].URL, "/")
	
	// Construct the URL for the promotion endpoint
	url := fmt.Sprintf("%s/v1/service/%s/environment/%s/promote", baseURL, serviceID, serviceEnvironmentID)
	
	// Create an empty JSON payload
	payload := []byte("{}")
	
	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create promotion request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute promotion request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check the response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("promotion request failed with status %d: %s", resp.StatusCode, string(body))
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
