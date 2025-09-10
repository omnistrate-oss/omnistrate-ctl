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
	PlanName     string                                           `json:"planName"`
	PlanID       string                                           `json:"planID"`
	ModelType    string                                           `json:"modelType"`
	Accounts     []openapiclientfleet.FleetDescribeAccountConfigResult `json:"accounts"`
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


// GetServiceAccountInfo gets account information for Customer hosted plans only, grouped by cloud provider and plan
func GetServiceAccountInfo(ctx context.Context, token string, service *openapiclient.DescribeServiceResult) (*ServiceAccountInfo, error) {
	result := &ServiceAccountInfo{
		ServiceID:          service.Id,
		ServiceName:        service.Name,
		AccountsByProvider: []AccountsByCloudProviderAndPlan{},
	}

	// Get all account configs first
	allAccountConfigs, err := ListAllAccountConfigs(ctx, token)
	if err != nil {
		return nil, err
	}

	// Create a map for quick account config lookup
	accountConfigMap := make(map[string]openapiclientfleet.FleetDescribeAccountConfigResult)
	for _, account := range allAccountConfigs.AccountConfigs {
		accountConfigMap[account.Id] = account
	}

	// Group plans by cloud provider
	cloudProviderPlans := make(map[string][]PlanWithAccounts)

	// Process each service environment
	for _, env := range service.ServiceEnvironments {
		// Process each service plan
		for _, plan := range env.ServicePlans {
			// Only process Customer hosted plans (skip Omnistrate hosted and BYOA)
			if !isCustomerHosted(plan.ModelType) {
				continue
			}

			// Get product tier details to find service model ID
			productTier, err := DescribeProductTier(ctx, token, service.Id, plan.ProductTierID)
			if err != nil {
				// Skip this plan if we can't get product tier details
				continue
			}

			// Get service model details to find account configs
			serviceModel, err := DescribeServiceModel(ctx, token, service.Id, productTier.ServiceModelId)
			if err != nil {
				// Skip this plan if we can't get service model details
				continue
			}

			// Get account configs for this service model
			var planAccounts []openapiclientfleet.FleetDescribeAccountConfigResult
			for _, accountID := range serviceModel.AccountConfigIds {
				if account, exists := accountConfigMap[accountID]; exists {
					planAccounts = append(planAccounts, account)
				}
			}

			// Group accounts by cloud provider
			cloudProviderAccountsMap := make(map[string][]openapiclientfleet.FleetDescribeAccountConfigResult)
			for _, account := range planAccounts {
				provider := getCloudProviderName(account.CloudProviderId)
				cloudProviderAccountsMap[provider] = append(cloudProviderAccountsMap[provider], account)
			}

			// Add plan to each cloud provider group
			for provider, accounts := range cloudProviderAccountsMap {
				planWithAccounts := PlanWithAccounts{
					PlanName:  plan.Name,
					PlanID:    plan.ProductTierID,
					ModelType: plan.ModelType,
					Accounts:  accounts,
				}

				cloudProviderPlans[provider] = append(cloudProviderPlans[provider], planWithAccounts)
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

// isCustomerHosted checks if a model type represents Customer hosted deployment
func isCustomerHosted(modelType string) bool {
	// Customer hosted plans typically have modelType that includes "Customer" or similar
	// This may need to be adjusted based on actual values
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