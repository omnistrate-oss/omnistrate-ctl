package dataaccess

import (
	"context"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

// AccountsByCloudProviderAndPlan holds account configuration information grouped by cloud provider and plan
type AccountsByCloudProviderAndPlan struct {
	CloudProvider string                                           `json:"cloudProvider"`
	Plans         []PlanWithAccounts                               `json:"plans"`
}

type PlanWithAccounts struct {
	PlanName             string                                           `json:"planName"`
	PlanID               string                                           `json:"planID"`
	ModelType            string                                           `json:"modelType"`
	ServiceModelId       string                                           `json:"serviceModelId,omitempty"`
	AccountConfigIds     []string                                         `json:"accountConfigIds,omitempty"`
	ActiveAccountConfigIds map[string]interface{}                        `json:"activeAccountConfigIds,omitempty"`
	Accounts             []openapiclientfleet.FleetDescribeAccountConfigResult `json:"accounts"`
}

type ServiceAccountInfo struct {
	ServiceID           string                                `json:"serviceId"`
	ServiceName         string                                `json:"serviceName"`
	AccountsByProvider  []AccountsByCloudProviderAndPlan      `json:"accountsByProvider"`
}

func ListAccountConfigs(ctx context.Context, token, cloudProviderName string) (*openapiclientfleet.ListAccountConfigsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	request := openapiclientfleet.FleetListAccountConfigsRequest2{
		CloudProviderName: cloudProviderName,
	}

	resp, r, err := apiClient.InventoryApiAPI.InventoryApiListAccountConfigs(ctxWithToken).
		FleetListAccountConfigsRequest2(request).
		Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func ListAllAccountConfigs(ctx context.Context, token string) (*openapiclientfleet.ListAccountConfigsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// The API requires a request object even for listing all configs
	// We'll try common cloud providers to get all configs
	allConfigs := &openapiclientfleet.ListAccountConfigsResult{
		AccountConfigs: []openapiclientfleet.FleetDescribeAccountConfigResult{},
	}

	cloudProviders := []string{"aws", "gcp", "azure"}
	
	for _, provider := range cloudProviders {
		request := openapiclientfleet.FleetListAccountConfigsRequest2{
			CloudProviderName: provider,
		}

		resp, r, err := apiClient.InventoryApiAPI.InventoryApiListAccountConfigs(ctxWithToken).
			FleetListAccountConfigsRequest2(request).
			Execute()

		if err != nil {
			// Continue with other providers if one fails
			if r != nil {
				r.Body.Close()
			}
			continue
		}

		// Append configs from this provider
		if resp != nil && resp.AccountConfigs != nil {
			allConfigs.AccountConfigs = append(allConfigs.AccountConfigs, resp.AccountConfigs...)
		}

		r.Body.Close()
	}

	return allConfigs, nil
}

func ListServiceModels(ctx context.Context, token, serviceID, serviceAPIID string) (*openapiclient.ListServiceModelsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceModelApiAPI.ServiceModelApiListServiceModel(ctxWithToken, serviceID, serviceAPIID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func ListServicePlans(ctx context.Context, token, serviceID, serviceEnvironmentID string) (*openapiclient.ListServicePlansResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServicePlanApiAPI.ServicePlanApiListServicePlans(ctxWithToken, serviceID, serviceEnvironmentID).
		SkipHasPendingChangesCheck(false).
		Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}


// GetServiceAccountInfo gets account information for Customer hosted plans only, grouped by cloud provider and plan
func GetServiceAccountInfo(ctx context.Context, token string, service *openapiclient.DescribeServiceResult) (*ServiceAccountInfo, error) {
	result := &ServiceAccountInfo{
		ServiceID:          service.Id,
		ServiceName:        service.Name,
		AccountsByProvider: []AccountsByCloudProviderAndPlan{},
	}

	// Get all account configs first for detailed account information
	allAccountConfigs, err := ListAllAccountConfigs(ctx, token)
	if err != nil {
		// Continue without detailed account info if fleet API fails
		allAccountConfigs = &openapiclientfleet.ListAccountConfigsResult{
			AccountConfigs: []openapiclientfleet.FleetDescribeAccountConfigResult{},
		}
	}

	// Create a map for quick account config lookup
	accountConfigMap := make(map[string]openapiclientfleet.FleetDescribeAccountConfigResult)
	for _, account := range allAccountConfigs.AccountConfigs {
		accountConfigMap[account.Id] = account
	}

	// Group plans by cloud provider
	cloudProviderPlans := make(map[string][]PlanWithAccounts)

	// Process each service environment using the detailed service plan API
	for _, env := range service.ServiceEnvironments {
		// Get detailed service plan information for this environment
		servicePlansResult, err := ListServicePlans(ctx, token, service.Id, env.Id)
		if err != nil {
			// Skip this environment if we can't get detailed service plan info
			continue
		}

		// Process each detailed service plan
		for _, detailedPlan := range servicePlansResult.ServicePlans {
			// Only process Customer hosted plans (skip Omnistrate hosted and BYOA)
			if !isCustomerHosted(detailedPlan.ModelType) {
				continue
			}

			// Get account details for this plan
			var planAccounts []openapiclientfleet.FleetDescribeAccountConfigResult
			for _, accountID := range detailedPlan.AccountConfigIds {
				if account, exists := accountConfigMap[accountID]; exists {
					planAccounts = append(planAccounts, account)
				}
			}

			// Create plan with detailed information from the service plan API
			planWithAccounts := PlanWithAccounts{
				PlanName:               detailedPlan.ProductTierName,
				PlanID:                 detailedPlan.ProductTierId,
				ModelType:              detailedPlan.ModelType,
				ServiceModelId:         detailedPlan.ServiceModelId,
				AccountConfigIds:       detailedPlan.AccountConfigIds,
				ActiveAccountConfigIds: detailedPlan.ActiveAccountConfigIds,
				Accounts:               planAccounts,
			}

			// If no detailed account information is available, group by active account configs
			if len(planAccounts) == 0 && len(detailedPlan.AccountConfigIds) > 0 {
				// Plan has account config IDs but we couldn't get detailed info
				cloudProviderPlans["Account Details Unavailable"] = append(cloudProviderPlans["Account Details Unavailable"], planWithAccounts)
			} else if len(planAccounts) == 0 {
				// Plan has no account configs at all
				cloudProviderPlans["No Accounts Configured"] = append(cloudProviderPlans["No Accounts Configured"], planWithAccounts)
			} else {
				// Group accounts by cloud provider
				cloudProviderAccountsMap := make(map[string][]openapiclientfleet.FleetDescribeAccountConfigResult)
				for _, account := range planAccounts {
					provider := getCloudProviderName(account.CloudProviderId)
					cloudProviderAccountsMap[provider] = append(cloudProviderAccountsMap[provider], account)
				}

				// Add plan to each cloud provider group
				for provider, accounts := range cloudProviderAccountsMap {
					planCopy := planWithAccounts
					planCopy.Accounts = accounts
					cloudProviderPlans[provider] = append(cloudProviderPlans[provider], planCopy)
				}
			}
		}
	}

	// Convert map to slice
	for provider, plans := range cloudProviderPlans {
		result.AccountsByProvider = append(result.AccountsByProvider, AccountsByCloudProviderAndPlan{
			CloudProvider: provider,
			Plans:         plans,
		})
	}

	return result, nil
}

// EnhanceServicePlansWithAccountInfo enhances the existing service plans with detailed account information
func EnhanceServicePlansWithAccountInfo(ctx context.Context, token string, service *openapiclient.DescribeServiceResult) error {
	// Get all account configs first for detailed account information
	allAccountConfigs, err := ListAllAccountConfigs(ctx, token)
	if err != nil {
		// Continue without detailed account info if fleet API fails
		allAccountConfigs = &openapiclientfleet.ListAccountConfigsResult{
			AccountConfigs: []openapiclientfleet.FleetDescribeAccountConfigResult{},
		}
	}

	// Create a map for quick account config lookup
	accountConfigMap := make(map[string]openapiclientfleet.FleetDescribeAccountConfigResult)
	for _, account := range allAccountConfigs.AccountConfigs {
		accountConfigMap[account.Id] = account
	}

	// Process each service environment
	for envIdx, env := range service.ServiceEnvironments {
		// Get detailed service plan information for this environment
		servicePlansResult, err := ListServicePlans(ctx, token, service.Id, env.Id)
		if err != nil {
			// Skip this environment if we can't get detailed service plan info
			continue
		}

		// Create a map of detailed plans by product tier ID for quick lookup
		detailedPlanMap := make(map[string]openapiclient.GetServicePlanResult)
		for _, detailedPlan := range servicePlansResult.ServicePlans {
			detailedPlanMap[detailedPlan.ProductTierId] = detailedPlan
		}

		// Enhance each existing service plan with detailed information
		for planIdx, plan := range env.ServicePlans {
			// Only enhance Customer hosted plans (skip Omnistrate hosted and BYOA)
			if !isCustomerHosted(plan.ModelType) {
				continue
			}

			// Find the corresponding detailed plan
			detailedPlan, exists := detailedPlanMap[plan.ProductTierID]
			if !exists {
				continue
			}

			// Get account details for this plan
			var planAccounts []openapiclientfleet.FleetDescribeAccountConfigResult
			for _, accountID := range detailedPlan.AccountConfigIds {
				if account, exists := accountConfigMap[accountID]; exists {
					planAccounts = append(planAccounts, account)
				}
			}

			// Group accounts by cloud provider
			accountsByProvider := make(map[string][]openapiclientfleet.FleetDescribeAccountConfigResult)
			for _, account := range planAccounts {
				provider := getCloudProviderName(account.CloudProviderId)
				accountsByProvider[provider] = append(accountsByProvider[provider], account)
			}

			// Initialize AdditionalProperties if nil
			if service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties == nil {
				service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties = make(map[string]interface{})
			}

			// Add enhanced information to the existing service plan
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["serviceModelId"] = detailedPlan.ServiceModelId
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["accountConfigIds"] = detailedPlan.AccountConfigIds
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["activeAccountConfigIds"] = detailedPlan.ActiveAccountConfigIds
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["accountsByProvider"] = accountsByProvider
			
			// Add additional detailed plan information that might be useful
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["productTierKey"] = detailedPlan.ProductTierKey
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["serviceApiId"] = detailedPlan.ServiceApiId
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["deploymentConfigId"] = detailedPlan.DeploymentConfigId
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["versionSetStatus"] = detailedPlan.VersionSetStatus
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["latestMajorVersion"] = detailedPlan.LatestMajorVersion
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["hasAccountsConfigured"] = len(detailedPlan.AccountConfigIds) > 0
			service.ServiceEnvironments[envIdx].ServicePlans[planIdx].AdditionalProperties["accountsAvailable"] = len(planAccounts) > 0
		}
	}

	return nil
}

// isCustomerHosted checks if a model type represents Customer hosted deployment
func isCustomerHosted(modelType string) bool {
	// Customer hosted plans typically have modelType that includes "Customer" or similar
	// Based on the example, it can be "CUSTOMER_HOSTED"
	modelTypeLower := strings.ToLower(modelType)
	return strings.Contains(modelTypeLower, "customer")
}

// getCloudProviderName maps cloud provider IDs to readable names
func getCloudProviderName(cloudProviderID string) string {
	// This is a simplified mapping - may need to be enhanced based on actual provider IDs
	switch strings.ToLower(cloudProviderID) {
	case "aws":
		return "AWS"
	case "gcp":
		return "GCP"  
	case "azure":
		return "Azure"
	default:
		return cloudProviderID
	}
}