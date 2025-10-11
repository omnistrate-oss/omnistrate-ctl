package dataaccess

import (
	"context"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
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

func convertTemplateToOpenAPIFormat(deploymentConfig model.DeploymentCellTemplate, cloudProvider string) (openapiclient.DeploymentCellConfigurations, error) {
	apiModel := openapiclient.DeploymentCellConfigurations{}
	configPerCloudProvider := make(map[string]openapiclient.DeploymentCellConfiguration)

	var amenitiesAPI []openapiclient.Amenity
	for _, amenity := range deploymentConfig.ManagedAmenities {
		apiAmenity := openapiclient.Amenity{
			Name:        utils.ToPtr(amenity.Name),
			Description: amenity.Description,
			Type:        amenity.Type,
			Properties:  amenity.Properties,
			IsManaged:   utils.ToPtr(true),
		}
		amenitiesAPI = append(amenitiesAPI, apiAmenity)
	}
	for _, amenity := range deploymentConfig.CustomAmenities {
		apiAmenity := openapiclient.Amenity{
			Name:        utils.ToPtr(amenity.Name),
			Description: amenity.Description,
			Type:        amenity.Type,
			Properties:  amenity.Properties,
			IsManaged:   utils.ToPtr(false),
		}
		amenitiesAPI = append(amenitiesAPI, apiAmenity)
	}
	configPerCloudProvider[cloudProvider] = openapiclient.DeploymentCellConfiguration{
		Amenities: amenitiesAPI,
	}

	apiModel.DeploymentCellConfigurationPerCloudProvider = utils.ToPtr(configPerCloudProvider)

	return apiModel, nil
}

func UpdateServiceProviderOrganization(ctx context.Context, token string, deploymentConfig model.DeploymentCellTemplate, envType string, cloudProvider string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	apiModel, err := convertTemplateToOpenAPIFormat(deploymentConfig, cloudProvider)
	if err != nil {
		return
	}

	configMap := map[string]openapiclient.DeploymentCellConfigurations{
		envType: apiModel,
	}

	req := apiClient.SpOrganizationApiAPI.SpOrganizationApiModifyServiceProviderOrganization(ctxWithToken)
	spOrg := openapiclient.ModifyServiceProviderOrganizationRequest2{
		DeploymentCellConfigurations: utils.ToPtr(configMap),
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
