package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const (
	AccessApprovalModeAuto   = "AUTO"
	AccessApprovalModeManual = "MANUAL"
	AccessEnvironmentGlobal  = "GLOBAL"
	AccessEnvironmentProd    = "PROD"
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

// GetOrganizationUsers returns users in the authenticated organization.
func GetOrganizationUsers(ctx context.Context, token string) (users []openapiclient.OrgUsers, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	result, response, err := apiClient.UsersApiAPI.UsersApiDescribeUsersByOrg(ctxWithToken).Execute()
	if err = handleV1Error(err); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("organization users API returned no result")
	}
	return result.OrgUsers, nil
}

// ResolveAccessRequestEnvironment returns an explicit environment or selects
// GLOBAL/PROD from the organization's deployment-cell mode.
func ResolveAccessRequestEnvironment(
	organization *openapiclient.DescribeServiceProviderOrganizationResult,
	environment string,
) (string, error) {
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "" {
		if environment != AccessEnvironmentGlobal && environment != AccessEnvironmentProd {
			return "", fmt.Errorf("environment type must be GLOBAL or PROD")
		}
		return environment, nil
	}
	if organization == nil || organization.IsPerEnvClusterEnabled == nil {
		return "", fmt.Errorf("unable to determine the default environment type; specify --environment-type GLOBAL or PROD")
	}
	if *organization.IsPerEnvClusterEnabled {
		return AccessEnvironmentProd, nil
	}
	return AccessEnvironmentGlobal, nil
}

type rawAccessRequestApprovalConfiguration struct {
	ApprovalMode  string `json:"approvalMode"`
	Defaulted     bool   `json:"defaulted"`
	ApproverUsers []struct {
		UserID string `json:"userId"`
	} `json:"approverUsers"`
}

// AccessRequestApprovalConfiguration reads the effective JIT configuration.
// Missing configurations are treated as defaulted AUTO without writing them.
func AccessRequestApprovalConfiguration(
	organization *openapiclient.DescribeServiceProviderOrganizationResult,
	users []openapiclient.OrgUsers,
	environment string,
) (*model.AccessRequestApprovalConfiguration, error) {
	configuration := rawAccessRequestApprovalConfiguration{
		ApprovalMode: AccessApprovalModeAuto,
		Defaulted:    true,
	}
	if organization != nil && organization.AdditionalProperties != nil {
		if configurations, ok := organization.AdditionalProperties["justInTimeAccessConfigurations"]; ok {
			encoded, err := json.Marshal(configurations)
			if err != nil {
				return nil, fmt.Errorf("failed to read JIT access configurations: %w", err)
			}
			var byEnvironment map[string]rawAccessRequestApprovalConfiguration
			if err = json.Unmarshal(encoded, &byEnvironment); err != nil {
				return nil, fmt.Errorf("failed to decode JIT access configurations: %w", err)
			}
			if configured, found := byEnvironment[environment]; found {
				configuration = configured
			}
		}
	}

	if configuration.ApprovalMode == "" {
		configuration.ApprovalMode = AccessApprovalModeAuto
	}
	usersByID := make(map[string]openapiclient.OrgUsers, len(users))
	for _, user := range users {
		usersByID[user.UserId] = user
	}
	approvers := make([]model.AccessRequestApprover, 0, len(configuration.ApproverUsers))
	for _, configured := range configuration.ApproverUsers {
		approver := model.AccessRequestApprover{UserID: configured.UserID}
		if user, found := usersByID[configured.UserID]; found {
			approver.Email = user.Email
			approver.Name = user.Name
		}
		approvers = append(approvers, approver)
	}
	return &model.AccessRequestApprovalConfiguration{
		EnvironmentType: environment,
		ApprovalMode:    strings.ToUpper(configuration.ApprovalMode),
		Defaulted:       configuration.Defaulted,
		Approvers:       approvers,
	}, nil
}

// ResolveAccessRequestApproverEmails resolves unique approver emails to user IDs.
func ResolveAccessRequestApproverEmails(users []openapiclient.OrgUsers, emails []string) ([]string, error) {
	usersByEmail := make(map[string][]openapiclient.OrgUsers, len(users))
	for _, user := range users {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email != "" {
			usersByEmail[email] = append(usersByEmail[email], user)
		}
	}

	userIDs := make([]string, 0, len(emails))
	seenEmails := make(map[string]bool, len(emails))
	for _, input := range emails {
		email := strings.ToLower(strings.TrimSpace(input))
		if email == "" || seenEmails[email] {
			continue
		}
		seenEmails[email] = true
		matches := usersByEmail[email]
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("no organization user found for approver email %s", strings.TrimSpace(input))
		case 1:
			userIDs = append(userIDs, matches[0].UserId)
		default:
			return nil, fmt.Errorf("multiple organization users found for approver email %s", strings.TrimSpace(input))
		}
	}
	return userIDs, nil
}

// GetAccessRequestApprovalConfiguration loads the effective configuration and
// resolves configured approvers to organization user details.
func GetAccessRequestApprovalConfiguration(
	ctx context.Context,
	token string,
	environment string,
) (*model.AccessRequestApprovalConfiguration, error) {
	organization, err := GetServiceProviderOrganization(ctx, token)
	if err != nil {
		return nil, err
	}
	environment, err = ResolveAccessRequestEnvironment(organization, environment)
	if err != nil {
		return nil, err
	}
	configuration, err := AccessRequestApprovalConfiguration(organization, nil, environment)
	if err != nil {
		return nil, err
	}
	if len(configuration.Approvers) == 0 {
		return configuration, nil
	}
	users, err := GetOrganizationUsers(ctx, token)
	if err != nil {
		return nil, err
	}
	return AccessRequestApprovalConfiguration(organization, users, environment)
}

// GetConfiguredAccessRequestApprovers returns configured approver emails,
// falling back to user IDs for stale configuration entries.
func GetConfiguredAccessRequestApprovers(ctx context.Context, token, environment string) ([]string, error) {
	configuration, err := GetAccessRequestApprovalConfiguration(ctx, token, environment)
	if err != nil {
		return nil, err
	}
	approvers := make([]string, 0, len(configuration.Approvers))
	seen := make(map[string]bool, len(configuration.Approvers))
	for _, approver := range configuration.Approvers {
		contact := strings.TrimSpace(approver.Email)
		if contact == "" {
			contact = strings.TrimSpace(approver.UserID)
		}
		normalized := strings.ToLower(contact)
		if contact != "" && !seen[normalized] {
			seen[normalized] = true
			approvers = append(approvers, contact)
		}
	}
	return approvers, nil
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

// ConfigureAccessRequestApproval updates one environment's just-in-time access
// approval mode and optional inline approver list.
func ConfigureAccessRequestApproval(
	ctx context.Context,
	token string,
	environment string,
	approvalMode string,
	approverUserIDs []string,
) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	type approverUser struct {
		UserID string `json:"userId"`
	}
	type approvalConfiguration struct {
		ApprovalMode  string         `json:"approvalMode"`
		ApproverUsers []approverUser `json:"approverUsers"`
	}

	approvers := make([]approverUser, 0, len(approverUserIDs))
	for _, userID := range approverUserIDs {
		approvers = append(approvers, approverUser{UserID: userID})
	}
	configurations := map[string]approvalConfiguration{
		environment: {
			ApprovalMode:  approvalMode,
			ApproverUsers: approvers,
		},
	}

	req := apiClient.SpOrganizationApiAPI.SpOrganizationApiModifyServiceProviderOrganization(ctxWithToken)
	req = req.ModifyServiceProviderOrganizationRequest2(openapiclient.ModifyServiceProviderOrganizationRequest2{
		AdditionalProperties: map[string]interface{}{
			"justInTimeAccessConfigurations": configurations,
		},
	})
	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = req.Execute()
	return handleV1Error(err)
}
