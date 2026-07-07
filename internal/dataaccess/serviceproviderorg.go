package dataaccess

import (
	"context"
	"encoding/json"
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

func convertTemplateToOpenAPIFormat(deploymentConfig model.DeploymentCellTemplate, cloudProvider string) openapiclient.DeploymentCellConfigurations {
	apiModel := openapiclient.DeploymentCellConfigurations{}
	configPerCloudProvider := make(map[string]openapiclient.DeploymentCellConfiguration)

	var amenitiesAPI []openapiclient.Amenity
	for _, amenity := range deploymentConfig.ManagedAmenities {
		apiAmenity := openapiclient.Amenity{
			Name:        utils.ToPtr(amenity.Name),
			Description: amenity.Description,
			Disable:     amenity.Disable,
			Type:        amenity.Type,
			DependsOn:   amenity.DependsOn,
			Properties:  amenity.Properties,
			IsManaged:   utils.ToPtr(true),
		}
		amenitiesAPI = append(amenitiesAPI, apiAmenity)
	}
	for _, amenity := range deploymentConfig.CustomAmenities {
		apiAmenity := openapiclient.Amenity{
			Name:        utils.ToPtr(amenity.Name),
			Description: amenity.Description,
			Disable:     amenity.Disable,
			Type:        amenity.Type,
			DependsOn:   amenity.DependsOn,
			Properties:  amenity.Properties,
			IsManaged:   utils.ToPtr(false),
		}
		amenitiesAPI = append(amenitiesAPI, apiAmenity)
	}
	configPerCloudProvider[cloudProvider] = openapiclient.DeploymentCellConfiguration{
		Amenities:          amenitiesAPI,
		WorkloadIdentities: ConvertManagedWorkloadIdentitiesToOpenAPI(deploymentConfig.ManagedIdentities),
	}

	apiModel.DeploymentCellConfigurationPerCloudProvider = utils.ToPtr(configPerCloudProvider)

	return apiModel
}

func ConvertManagedWorkloadIdentitiesToOpenAPI(identities []model.ManagedWorkloadIdentity) []openapiclient.ManagedWorkloadIdentity {
	if len(identities) == 0 {
		return nil
	}

	result := make([]openapiclient.ManagedWorkloadIdentity, 0, len(identities))
	for _, identity := range identities {
		apiIdentity := openapiclient.ManagedWorkloadIdentity{
			Identifier:  identity.Identifier,
			Description: identity.Description,
			Bindings:    convertManagedWorkloadIdentityBindingsToOpenAPI(identity.Bindings),
		}
		if identity.Permissions != nil {
			apiIdentity.Permissions = convertManagedWorkloadIdentityPermissionsToOpenAPI(identity.Permissions)
		}
		result = append(result, apiIdentity)
	}

	return result
}

func ConvertManagedWorkloadIdentitiesFromOpenAPI(identities []openapiclient.ManagedWorkloadIdentity) []model.ManagedWorkloadIdentity {
	if len(identities) == 0 {
		return nil
	}

	result := make([]model.ManagedWorkloadIdentity, 0, len(identities))
	for _, identity := range identities {
		modelIdentity := model.ManagedWorkloadIdentity{
			Identifier:  identity.Identifier,
			Description: identity.Description,
			Bindings:    convertManagedWorkloadIdentityBindingsFromOpenAPI(identity.Bindings),
		}
		if identity.Permissions != nil {
			modelIdentity.Permissions = convertManagedWorkloadIdentityPermissionsFromOpenAPI(identity.Permissions)
		}
		result = append(result, modelIdentity)
	}

	return result
}

func ConvertToInternalManagedWorkloadIdentityList(data interface{}) ([]model.ManagedWorkloadIdentity, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var rawObject map[string]json.RawMessage
	if err = json.Unmarshal(jsonBytes, &rawObject); err == nil {
		if rawIdentities, ok := rawObject["WorkloadIdentities"]; ok {
			var identities []openapiclient.ManagedWorkloadIdentity
			if err = json.Unmarshal(rawIdentities, &identities); err != nil {
				return nil, err
			}
			return ConvertManagedWorkloadIdentitiesFromOpenAPI(identities), nil
		}
		return nil, nil
	}

	var identities []openapiclient.ManagedWorkloadIdentity
	if err = json.Unmarshal(jsonBytes, &identities); err != nil {
		return nil, err
	}

	return ConvertManagedWorkloadIdentitiesFromOpenAPI(identities), nil
}

func convertManagedWorkloadIdentityBindingsToOpenAPI(bindings []model.ManagedWorkloadIdentityBinding) []openapiclient.ManagedWorkloadIdentityBinding {
	if len(bindings) == 0 {
		return nil
	}

	result := make([]openapiclient.ManagedWorkloadIdentityBinding, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, openapiclient.ManagedWorkloadIdentityBinding{
			ServiceAccount: openapiclient.ManagedWorkloadIdentityServiceAccount{
				Namespace: binding.ServiceAccount.Namespace,
				Name:      binding.ServiceAccount.Name,
			},
		})
	}
	return result
}

func convertManagedWorkloadIdentityBindingsFromOpenAPI(bindings []openapiclient.ManagedWorkloadIdentityBinding) []model.ManagedWorkloadIdentityBinding {
	if len(bindings) == 0 {
		return nil
	}

	result := make([]model.ManagedWorkloadIdentityBinding, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, model.ManagedWorkloadIdentityBinding{
			ServiceAccount: model.ManagedWorkloadIdentityServiceAccount{
				Namespace: binding.ServiceAccount.Namespace,
				Name:      binding.ServiceAccount.Name,
			},
		})
	}
	return result
}

func convertManagedWorkloadIdentityPermissionsToOpenAPI(permissions *model.ManagedWorkloadIdentityPermissions) *openapiclient.ManagedWorkloadIdentityPermissions {
	result := &openapiclient.ManagedWorkloadIdentityPermissions{}
	if permissions.Policies != nil {
		policies := permissions.Policies
		result.Policies = &policies
	}
	if permissions.Permissions != nil {
		permissionStatements := permissions.Permissions
		result.Permissions = &permissionStatements
	}
	if permissions.Roles != nil {
		roles := make(map[string][]openapiclient.ManagedWorkloadIdentityRole, len(permissions.Roles))
		for provider, providerRoles := range permissions.Roles {
			apiRoles := make([]openapiclient.ManagedWorkloadIdentityRole, 0, len(providerRoles))
			for _, role := range providerRoles {
				apiRoles = append(apiRoles, openapiclient.ManagedWorkloadIdentityRole{
					Name: role.Name,
					Type: role.Type,
				})
			}
			roles[provider] = apiRoles
		}
		result.Roles = &roles
	}
	return result
}

func convertManagedWorkloadIdentityPermissionsFromOpenAPI(permissions *openapiclient.ManagedWorkloadIdentityPermissions) *model.ManagedWorkloadIdentityPermissions {
	result := &model.ManagedWorkloadIdentityPermissions{}
	if permissions.Policies != nil {
		result.Policies = *permissions.Policies
	}
	if permissions.Permissions != nil {
		result.Permissions = *permissions.Permissions
	}
	if permissions.Roles != nil {
		roles := make(map[string][]model.ManagedWorkloadIdentityRole, len(*permissions.Roles))
		for provider, providerRoles := range *permissions.Roles {
			modelRoles := make([]model.ManagedWorkloadIdentityRole, 0, len(providerRoles))
			for _, role := range providerRoles {
				modelRoles = append(modelRoles, model.ManagedWorkloadIdentityRole{
					Name: role.Name,
					Type: role.Type,
				})
			}
			roles[provider] = modelRoles
		}
		result.Roles = roles
	}
	return result
}

func UpdateServiceProviderOrganization(ctx context.Context, token string, deploymentConfig model.DeploymentCellTemplate, envType string, cloudProvider string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	apiModel := convertTemplateToOpenAPIFormat(deploymentConfig, cloudProvider)

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
