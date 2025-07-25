package dataaccess

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const (
	NextStepsAfterBuildMsgTemplate = `
Next steps:
- Customize domain name for SaaS offer: check 'omnistrate-ctl create domain' command
- Update the service configuration: check 'omnistrate-ctl build' command`
)

func PrintNextStepsAfterBuildMsg() {
	fmt.Println(NextStepsAfterBuildMsgTemplate)
}

func ListServices(ctx context.Context, token string) (*openapiclient.ListServiceResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiAPI.ServiceApiListService(ctxWithToken).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func DescribeService(ctx context.Context, token, serviceID string) (*openapiclient.DescribeServiceResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiAPI.ServiceApiDescribeService(ctxWithToken, serviceID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func DeleteService(ctx context.Context, token, serviceID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	r, err := apiClient.ServiceApiAPI.ServiceApiDeleteService(ctxWithToken, serviceID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}
	r.Body.Close()

	return nil
}

func BuildServiceFromServicePlanSpec(ctx context.Context, token string, request openapiclient.BuildServiceFromServicePlanSpecRequest2) (*openapiclient.BuildServiceFromServicePlanSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiBuildServiceFromServicePlanSpec(ctxWithToken).
		BuildServiceFromServicePlanSpecRequest2(request).
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

func BuildServiceFromComposeSpec(ctx context.Context, token string, request openapiclient.BuildServiceFromComposeSpecRequest2) (*openapiclient.BuildServiceFromComposeSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiBuildServiceFromComposeSpec(ctxWithToken).
		BuildServiceFromComposeSpecRequest2(request).
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

// InitializeOrganizationAmenitiesConfiguration initializes the organization-level amenities configuration
// This is a placeholder implementation - the actual API endpoint may not exist yet
func InitializeOrganizationAmenitiesConfiguration(ctx context.Context, token string, environment string, configTemplate map[string]interface{}) (*model.AmenitiesConfiguration, error) {
	// TODO: Replace with actual API call once backend is available
	// For now, return a mock response to enable testing of the CLI interface
	
	config := &model.AmenitiesConfiguration{
		OrganizationID:        "", // Will be determined from token/credentials
		Environment:          environment,
		ConfigurationTemplate: configTemplate,
		Version:              "1.0.0",
	}
	
	return config, nil
}

// UpdateOrganizationAmenitiesConfiguration updates the amenities configuration for a target environment
// This is a placeholder implementation - the actual API endpoint may not exist yet
func UpdateOrganizationAmenitiesConfiguration(ctx context.Context, token string, environment string, configTemplate map[string]interface{}) (*model.AmenitiesConfiguration, error) {
	// TODO: Replace with actual API call once backend is available
	
	config := &model.AmenitiesConfiguration{
		OrganizationID:        "", // Will be determined from token/credentials
		Environment:          environment,
		ConfigurationTemplate: configTemplate,
		Version:              "1.1.0",
	}
	
	return config, nil
}

// GetOrganizationAmenitiesConfiguration retrieves the amenities configuration for an organization and environment
// This is a placeholder implementation - the actual API endpoint may not exist yet
func GetOrganizationAmenitiesConfiguration(ctx context.Context, token string, environment string) (*model.AmenitiesConfiguration, error) {
	// TODO: Replace with actual API call once backend is available
	
	// Mock configuration for testing
	mockConfig := map[string]interface{}{
		"logging": map[string]interface{}{
			"level":    "INFO",
			"rotation": "daily",
		},
		"monitoring": map[string]interface{}{
			"enabled":    true,
			"prometheus": true,
			"grafana":    true,
		},
		"security": map[string]interface{}{
			"encryption": "AES256",
			"tls_version": "1.3",
		},
	}
	
	config := &model.AmenitiesConfiguration{
		OrganizationID:        "", // Will be determined from token/credentials
		Environment:          environment,
		ConfigurationTemplate: mockConfig,
		Version:              "1.0.0",
	}
	
	return config, nil
}

// ListAvailableEnvironments lists available environments for amenities configuration
// This is a placeholder implementation - the actual API endpoint may not exist yet
func ListAvailableEnvironments(ctx context.Context, token string) ([]model.AmenitiesEnvironment, error) {
	// TODO: Replace with actual API call once backend is available
	
	// Mock environments
	environments := []model.AmenitiesEnvironment{
		{
			Name:        "production",
			DisplayName: "Production",
			Description: "Production environment configuration",
		},
		{
			Name:        "staging",
			DisplayName: "Staging",
			Description: "Staging environment configuration",
		},
		{
			Name:        "development",
			DisplayName: "Development",
			Description: "Development environment configuration",
		},
	}
	
	return environments, nil
}

// ValidateAmenitiesConfiguration validates the provided amenities configuration
func ValidateAmenitiesConfiguration(configTemplate map[string]interface{}) error {
	if configTemplate == nil {
		return fmt.Errorf("configuration template cannot be nil")
	}
	
	if len(configTemplate) == 0 {
		return fmt.Errorf("configuration template cannot be empty")
	}
	
	// Add more validation logic as needed based on the actual configuration schema
	
	return nil
}
