package dataaccess

import (
	"context"
	"encoding/json"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"net/http"
)

func GetServiceProviderOrganization(ctx context.Context, token string) (res *openapiclient.DescribeServiceProviderOrganizationResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = apiClient.SpOrganizationApiAPI.SpOrganizationApiDescribeServiceProviderOrganization(ctxWithToken).Execute()

	err = handleV1Error(err)
	if err != nil {
		return
	}

	return
}

func UpdateServiceProviderOrganization(ctx context.Context, token string, deploymentCellConfigurations model.DeploymentCellConfigurations, envType string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	apiModel, err := convertTemplateToOpenAPIFormat(deploymentCellConfigurations, envType)

	req := apiClient.SpOrganizationApiAPI.SpOrganizationApiModifyServiceProviderOrganization(ctxWithToken)
	spOrg := openapiclient.ModifyServiceProviderOrganizationRequest2{
		DeploymentCellConfigurations: apiModel,
	}
	req = req.ModifyServiceProviderOrganizationRequest2(spOrg)
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleV1Error(err)
	}
	return
}

func convertTemplateToOpenAPIFormat(templateConfig model.DeploymentCellConfigurations, environment string) (*map[string]openapiclient.DeploymentCellConfigurations, error) {
	// Create the map structure expected by the OpenAPI client
	// This should be a map where the key is environment type and value is DeploymentCellConfigurations
	deploymentCellConfigs := make(map[string]openapiclient.DeploymentCellConfigurations)

	// Create cloud provider configurations map
	cloudProviderConfigs := make(map[string]openapiclient.DeploymentCellConfiguration)

	// Convert each cloud provider configuration
	for cloudProvider, config := range templateConfig.DeploymentCellConfigurationPerCloudProvider {
		// Convert amenities to OpenAPI format
		var amenities []openapiclient.Amenity
		for _, amenity := range config.Amenities {
			openAPIAmenity := openapiclient.Amenity{}

			// Set the name (required field)
			openAPIAmenity.SetName(amenity.Name)

			// Set optional fields if they exist
			if amenity.Modifiable != nil {
				openAPIAmenity.SetModifiable(*amenity.Modifiable)
			}
			if amenity.Description != nil {
				openAPIAmenity.SetDescription(*amenity.Description)
			}
			if amenity.IsManaged != nil {
				openAPIAmenity.SetIsManaged(*amenity.IsManaged)
			}
			if amenity.Type != nil {
				openAPIAmenity.SetType(*amenity.Type)
			}
			if amenity.Properties != nil {
				openAPIAmenity.SetProperties(amenity.Properties)
			}

			amenities = append(amenities, openAPIAmenity)
		}

		// Create DeploymentCellConfiguration
		deploymentCellConfig := openapiclient.NewDeploymentCellConfiguration()
		deploymentCellConfig.SetAmenities(amenities)

		cloudProviderConfigs[cloudProvider] = *deploymentCellConfig
	}

	cloudProviderConfigsMap, err := structToMap(cloudProviderConfigs)
	if err != nil {
		return nil, err
	}

	// Create the DeploymentCellConfigurations structure
	deploymentCellConfiguration := openapiclient.NewDeploymentCellConfigurations()
	deploymentCellConfiguration.SetDeploymentCellConfigurationPerCloudProvider(cloudProviderConfigsMap)

	// Use the specified environment as the key
	deploymentCellConfigs[environment] = *deploymentCellConfiguration

	return &deploymentCellConfigs, nil
}

func structToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
