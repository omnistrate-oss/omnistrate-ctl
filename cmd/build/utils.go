package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Tenancy type constants
const (
	TenancyTypeCustom = "CUSTOM_TENANCY"
)

// Deployment model type constants
const (
	DeploymentModelHosted         = "hostedDeployment"
	DeploymentModelCustomerHosted = "customerHostedDeployment"
	DeploymentModelBYOA           = "byoaDeployment"
	DeploymentModelOnPrem         = "onPremDeployment"
	DeploymentModelOnPremCopilot  = "onPremCopilotDeployment"
)

// Service model type constants
const (
	ServiceModelTypeOmnistrateHosted = "OMNISTRATE_HOSTED"
	ServiceModelTypeCustomerHosted   = "CUSTOMER_HOSTED"
	ServiceModelTypeBYOA             = "BYOA"
	ServiceModelTypeOnPremCopilot    = "ON_PREM_COPILOT"
	ServiceModelTypeOnPrem           = "ON_PREM"
)

// DetectSpecType analyzes YAML content to determine if it contains service plan specifications
// Returns ServicePlanSpecType if plan-specific keys are found, otherwise DockerComposeSpecType
func DetectSpecType(yamlContent map[string]interface{}) string {
	// Improved: Recursively check for plan spec keys at any level
	planKeyGroups := [][]string{
		{"helm", "helmChart", "helmChartConfiguration"},
		{"operator", "operatorCRDConfiguration"},
		{"terraform", "terraformConfigurations"},
		{"kustomize", "kustomizeConfiguration"},
	}

	// Check if any plan-specific keys are found
	for _, keys := range planKeyGroups {
		if ContainsAnyKey(yamlContent, keys) {
			return ServicePlanSpecType
		}
	}

	return DockerComposeSpecType
}

// ContainsOmnistrateKey recursively searches for any x-omnistrate key in a map
func ContainsOmnistrateKey(m map[string]interface{}) bool {
	for k, v := range m {
		// Check for any x-omnistrate key
		if strings.HasPrefix(k, "x-omnistrate-") {
			return true
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]interface{}); ok {
			if ContainsOmnistrateKey(sub) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]interface{}); ok {
					if ContainsOmnistrateKey(subm) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ContainsAnyKey recursively searches for any key in keys in a map
func ContainsAnyKey(m map[string]interface{}, keys []string) bool {
	for k, v := range m {
		for _, key := range keys {
			if k == key {
				return true
			}
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]interface{}); ok {
			if ContainsAnyKey(sub, keys) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]interface{}); ok {
					if ContainsAnyKey(subm, keys) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ServicePlanSpecInfo holds information extracted from a service plan spec YAML
type ServicePlanSpecInfo struct {
	ProductTierName        string
	TenancyType            string // "OMNISTRATE_MULTI_TENANCY", "OMNISTRATE_DEDICATED_TENANCY", "CUSTOM_TENANCY"
	DeploymentModelType    string // "hostedDeployment", "byoaDeployment", "onPremDeployment", "onPremCopilotDeployment"
	AwsAccountID           string
	AwsBootstrapRoleARN    string
	GcpProjectID           string
	GcpProjectNumber       string
	GcpServiceAccountEmail string
	AzureSubscriptionID    string
	AzureTenantID          string
	OCITenancyID           string
	OCIDomainID            string
}

// ParseServicePlanSpec parses the service plan spec YAML and extracts relevant information
// This function only supports ServicePlanSpecType
func ParseServicePlanSpec(fileData []byte) (*ServicePlanSpecInfo, error) {
	var yamlContent map[string]interface{}
	if err := yaml.Unmarshal(fileData, &yamlContent); err != nil {
		return nil, fmt.Errorf("failed to parse YAML content: %w", err)
	}

	info := &ServicePlanSpecInfo{}

	// Service plan spec format: top-level fields
	if name, ok := yamlContent["name"].(string); ok {
		info.ProductTierName = name
	}
	// Service plan spec defaults to CUSTOM_TENANCY
	info.TenancyType = TenancyTypeCustom
	// Extract deployment model type and account configs from top-level deployment section
	extractDeploymentInfo(yamlContent, info)

	return info, nil
}

// extractDeploymentInfo extracts deployment model type and account configs from deployment section
func extractDeploymentInfo(yamlContent map[string]interface{}, info *ServicePlanSpecInfo) {
	if deployment, exists := yamlContent["deployment"]; exists {
		if depMap, ok := deployment.(map[string]interface{}); ok {
			if hosted, exists := depMap["hostedDeployment"]; exists {
				if hostedMap, ok := hosted.(map[string]interface{}); ok {
					extractAccountFromMap(hostedMap, info)
					// If hostedDeployment has cloud account info, it's CUSTOMER_HOSTED
					// Only if there's no account info, it's OMNISTRATE_HOSTED
					if info.AwsAccountID != "" || info.GcpProjectID != "" || info.AzureSubscriptionID != "" || info.OCITenancyID != "" {
						info.DeploymentModelType = DeploymentModelCustomerHosted
					} else {
						info.DeploymentModelType = DeploymentModelHosted
					}
				} else {
					// hostedDeployment is empty or not a map, treat as OMNISTRATE_HOSTED
					info.DeploymentModelType = DeploymentModelHosted
				}
			}
			if byoa, exists := depMap["byoaDeployment"]; exists {
				info.DeploymentModelType = DeploymentModelBYOA
				if byoaMap, ok := byoa.(map[string]interface{}); ok {
					extractAccountFromMap(byoaMap, info)
				}
			}
			if onPrem, exists := depMap["onPremDeployment"]; exists {
				info.DeploymentModelType = DeploymentModelOnPrem
				if onPremMap, ok := onPrem.(map[string]interface{}); ok {
					extractAccountFromMap(onPremMap, info)
				}
			}
			if onPremCopilot, exists := depMap["onPremCopilotDeployment"]; exists {
				info.DeploymentModelType = DeploymentModelOnPremCopilot
				if onPremCopilotMap, ok := onPremCopilot.(map[string]interface{}); ok {
					extractAccountFromMap(onPremCopilotMap, info)
				}
			}
		}
	}

	// Default to BYOA if no deployment model type was set
	if info.DeploymentModelType == "" {
		info.DeploymentModelType = DeploymentModelBYOA
	}
}

// extractAccountFromMap extracts cloud provider account information from a map
func extractAccountFromMap(m map[string]interface{}, info *ServicePlanSpecInfo) {
	getFirstString := func(m map[string]interface{}, keys ...string) string {
		for _, key := range keys {
			if v, ok := m[key].(string); ok && v != "" {
				return v
			}
			if v, ok := m[key].(int); ok {
				return fmt.Sprintf("%d", v)
			}
		}
		return ""
	}

	if info.AwsAccountID == "" {
		info.AwsAccountID = getFirstString(m, "AwsAccountId", "awsAccountId", "awsAccountID", "AwsAccountID")
	}
	if info.AwsBootstrapRoleARN == "" {
		info.AwsBootstrapRoleARN = getFirstString(m, "AwsBootstrapRoleAccountArn", "awsBootstrapRoleAccountArn", "awsBootstrapRoleARN", "AwsBootstrapRoleARN", "AWSBootstrapRoleAccountArn")
	}
	if info.GcpProjectID == "" {
		info.GcpProjectID = getFirstString(m, "GcpProjectId", "gcpProjectId", "gcpProjectID", "GcpProjectID")
	}
	if info.GcpProjectNumber == "" {
		info.GcpProjectNumber = getFirstString(m, "GcpProjectNumber", "gcpProjectNumber")
	}
	if info.GcpServiceAccountEmail == "" {
		info.GcpServiceAccountEmail = getFirstString(m, "GcpServiceAccountEmail", "gcpServiceAccountEmail")
	}
	if info.AzureSubscriptionID == "" {
		info.AzureSubscriptionID = getFirstString(m, "AzureSubscriptionId", "azureSubscriptionId", "azureSubscriptionID", "AzureSubscriptionID")
	}
	if info.AzureTenantID == "" {
		info.AzureTenantID = getFirstString(m, "AzureTenantId", "azureTenantId", "azureTenantID", "AzureTenantID")
	}
	if info.OCITenancyID == "" {
		info.OCITenancyID = getFirstString(m, "OCITenancyId", "ociTenancyId", "ociTenancyID", "OCITenancyID")
	}
	if info.OCIDomainID == "" {
		info.OCIDomainID = getFirstString(m, "OCIDomainId", "ociDomainId", "ociDomainID", "OCIDomainID")
	}
}

// MatchedAccountConfigs holds the matched account config IDs for each cloud provider
type MatchedAccountConfigs struct {
	AwsAccountConfigID   string
	GcpAccountConfigID   string
	AzureAccountConfigID string
	OciAccountConfigID   string
}

// AccountMatchResult holds the result of matching accounts from spec with existing accounts
type AccountMatchResult struct {
	Matched    MatchedAccountConfigs
	Missing    []string // List of missing account descriptions
	Unverified []string // List of unverified account descriptions
}

// FindMatchingAccountConfigs finds existing account configs that match the spec requirements
func FindMatchingAccountConfigs(ctx context.Context, token string, specInfo *ServicePlanSpecInfo) (*AccountMatchResult, error) {
	result := &AccountMatchResult{
		Missing:    []string{},
		Unverified: []string{},
	}

	// Check which cloud providers are specified in the spec
	needsAws := specInfo.AwsAccountID != ""
	needsGcp := specInfo.GcpProjectID != ""
	needsAzure := specInfo.AzureSubscriptionID != ""
	needsOci := specInfo.OCITenancyID != ""

	if !needsAws && !needsGcp && !needsAzure && !needsOci {
		// No account configs specified in spec, nothing to match
		return result, nil
	}

	// Fetch all accounts
	cloudProviders := []string{}
	if needsAws {
		cloudProviders = append(cloudProviders, "aws")
	}
	if needsGcp {
		cloudProviders = append(cloudProviders, "gcp")
	}
	if needsAzure {
		cloudProviders = append(cloudProviders, "azure")
	}
	if needsOci {
		cloudProviders = append(cloudProviders, "oci")
	}

	allAccounts := []*openapiclient.DescribeAccountConfigResult{}
	for _, cp := range cloudProviders {
		accounts, err := dataaccess.ListAccounts(ctx, token, cp)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s accounts: %w", cp, err)
		}
		for i := range accounts.AccountConfigs {
			allAccounts = append(allAccounts, &accounts.AccountConfigs[i])
		}
	}

	// Match AWS account
	if needsAws {
		found := false
		for _, acc := range allAccounts {
			if acc.AwsAccountID != nil && *acc.AwsAccountID == specInfo.AwsAccountID {
				found = true
				if acc.Status == "READY" {
					result.Matched.AwsAccountConfigID = acc.Id
				} else {
					result.Unverified = append(result.Unverified,
						fmt.Sprintf("AWS account %s (ID: %s) is in status %s", specInfo.AwsAccountID, acc.Id, acc.Status))
				}
				break
			}
		}
		if !found {
			result.Missing = append(result.Missing,
				fmt.Sprintf("AWS account %s is not linked. Please link it using 'omnistrate-ctl account create'", specInfo.AwsAccountID))
		}
	}

	// Match GCP account
	if needsGcp {
		found := false
		for _, acc := range allAccounts {
			if acc.GcpProjectID != nil && *acc.GcpProjectID == specInfo.GcpProjectID {
				// Also check project number if specified
				if specInfo.GcpProjectNumber != "" && acc.GcpProjectNumber != nil {
					if *acc.GcpProjectNumber != specInfo.GcpProjectNumber {
						continue
					}
				}
				found = true
				if acc.Status == "READY" {
					result.Matched.GcpAccountConfigID = acc.Id
				} else {
					result.Unverified = append(result.Unverified,
						fmt.Sprintf("GCP project %s (ID: %s) is in status %s", specInfo.GcpProjectID, acc.Id, acc.Status))
				}
				break
			}
		}
		if !found {
			result.Missing = append(result.Missing,
				fmt.Sprintf("GCP project %s is not linked. Please link it using 'omnistrate-ctl account create'", specInfo.GcpProjectID))
		}
	}

	// Match Azure account
	if needsAzure {
		found := false
		for _, acc := range allAccounts {
			if acc.AzureSubscriptionID != nil && *acc.AzureSubscriptionID == specInfo.AzureSubscriptionID {
				// Also check tenant ID if specified
				if specInfo.AzureTenantID != "" && acc.AzureTenantID != nil {
					if *acc.AzureTenantID != specInfo.AzureTenantID {
						continue
					}
				}
				found = true
				if acc.Status == "READY" {
					result.Matched.AzureAccountConfigID = acc.Id
				} else {
					result.Unverified = append(result.Unverified,
						fmt.Sprintf("Azure subscription %s (ID: %s) is in status %s", specInfo.AzureSubscriptionID, acc.Id, acc.Status))
				}
				break
			}
		}
		if !found {
			result.Missing = append(result.Missing,
				fmt.Sprintf("Azure subscription %s is not linked. Please link it using 'omnistrate-ctl account create'", specInfo.AzureSubscriptionID))
		}
	}

	// Match OCI account
	if needsOci {
		found := false
		for _, acc := range allAccounts {
			if acc.OciTenancyID != nil && *acc.OciTenancyID == specInfo.OCITenancyID {
				found = true
				if acc.Status == "READY" {
					result.Matched.OciAccountConfigID = acc.Id
				} else {
					result.Unverified = append(result.Unverified,
						fmt.Sprintf("OCI tenancy %s (ID: %s) is in status %s", specInfo.OCITenancyID, acc.Id, acc.Status))
				}
				break
			}
		}
		if !found {
			result.Missing = append(result.Missing,
				fmt.Sprintf("OCI tenancy %s is not linked. Please link it using 'omnistrate-ctl account create'", specInfo.OCITenancyID))
		}
	}

	return result, nil
}

// HasAnyAccountConfigID returns true if any account config ID is set
func (m *MatchedAccountConfigs) HasAnyAccountConfigID() bool {
	return m.AwsAccountConfigID != "" || m.GcpAccountConfigID != "" || m.AzureAccountConfigID != "" || m.OciAccountConfigID != ""
}

// ToSlice returns the account config IDs as a slice (non-empty ones only)
func (m *MatchedAccountConfigs) ToSlice() []string {
	var ids []string
	if m.AwsAccountConfigID != "" {
		ids = append(ids, m.AwsAccountConfigID)
	}
	if m.GcpAccountConfigID != "" {
		ids = append(ids, m.GcpAccountConfigID)
	}
	if m.AzureAccountConfigID != "" {
		ids = append(ids, m.AzureAccountConfigID)
	}
	if m.OciAccountConfigID != "" {
		ids = append(ids, m.OciAccountConfigID)
	}
	return ids
}

// ServiceHierarchyResult holds the result of finding or creating the service hierarchy
type ServiceHierarchyResult struct {
	ServiceID      string
	EnvironmentID  string
	ServiceAPIID   string
	ServiceModelID string
	ProductTierID  string
	IsNewService   bool
	IsNewTier      bool
}

// FindOrCreateServiceHierarchy finds or creates the service hierarchy for a service plan spec
// It returns the IDs of the service, environment, service API, service model, and product tier
// environmentName and environmentType are optional - if nil, defaults to "Development" and "DEV" respectively
// deploymentModelType is used to determine the service model type (OMNISTRATE_HOSTED or CUSTOMER_HOSTED)
func FindOrCreateServiceHierarchy(
	ctx context.Context,
	token string,
	serviceName string,
	productTierName string,
	description string,
	environmentName *string,
	environmentType *string,
	tenancyType string,
	deploymentModelType string,
	accountConfigIDs []string,
) (*ServiceHierarchyResult, error) {
	result := &ServiceHierarchyResult{}

	// Set defaults for environment name and type if not provided
	envName := "Development"
	if environmentName != nil && *environmentName != "" {
		envName = *environmentName
	}

	envType := "DEV"
	if environmentType != nil && *environmentType != "" {
		envType = strings.ToUpper(*environmentType)
	}

	// Convert deployment model type to service model type
	serviceModelType := DeploymentModelToServiceModelType(deploymentModelType)

	// Step 1: Find or create service by name
	serviceID, isNewService, err := findOrCreateService(ctx, token, serviceName, description)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create service: %w", err)
	}
	result.ServiceID = serviceID
	result.IsNewService = isNewService

	// Step 2: Find or create environment
	environmentID, err := findOrCreateEnvironment(ctx, token, serviceID, envName, envType)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create environment: %w", err)
	}
	result.EnvironmentID = environmentID

	// Step 3: Find product tier by name (search through service APIs and service models)
	productTierID, serviceAPIID, serviceModelID, err := findProductTierByName(ctx, token, serviceID, environmentID, productTierName)
	if err != nil {
		return nil, fmt.Errorf("failed to find product tier: %w", err)
	}

	if productTierID != "" {
		// Product tier found, return the result
		result.ProductTierID = productTierID
		result.ServiceAPIID = serviceAPIID
		result.ServiceModelID = serviceModelID
		result.IsNewTier = false
		return result, nil
	}

	// Step 4: Product tier not found, need to create the hierarchy
	// First, find or create a service API for this environment
	serviceAPIID, err = findOrCreateServiceAPI(ctx, token, serviceID, environmentID, productTierName)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create service API: %w", err)
	}
	result.ServiceAPIID = serviceAPIID

	// Validate that CUSTOMER_HOSTED, BYOA, ON_PREM, and ON_PREM_COPILOT model types have account config IDs
	// OMNISTRATE_HOSTED does not require account config IDs
	if (serviceModelType == ServiceModelTypeCustomerHosted || serviceModelType == ServiceModelTypeBYOA || serviceModelType == ServiceModelTypeOnPrem || serviceModelType == ServiceModelTypeOnPremCopilot) && len(accountConfigIDs) == 0 {
		return nil, fmt.Errorf("%s deployment requires at least one linked cloud account. "+
			"Please ensure your spec includes a deployment section with valid cloud provider account info "+
			"(e.g., AwsAccountId, GcpProjectId, AzureSubscriptionId, or OCITenancyId) and that these accounts "+
			"are linked using 'omnistrate-ctl account create'", serviceModelType)
	}

	// Step 5: Find or create service model with model type and account configs
	serviceModelID, err = findOrCreateServiceModel(ctx, token, serviceID, serviceAPIID, productTierName, serviceModelType, accountConfigIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create service model: %w", err)
	}
	result.ServiceModelID = serviceModelID

	// Step 6: Find or create product tier with tenancy type
	// Provide default description for product tier if empty
	tierDescription := description
	if tierDescription == "" {
		tierDescription = fmt.Sprintf("Product tier for %s", productTierName)
	}
	productTierID, isNewTier, err := findOrCreateProductTier(ctx, token, serviceID, serviceModelID, productTierName, tierDescription, tenancyType)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create product tier: %w", err)
	}
	result.ProductTierID = productTierID
	result.IsNewTier = isNewTier

	return result, nil
}

// findOrCreateService finds a service by name or creates a new one
func findOrCreateService(ctx context.Context, token, serviceName, description string) (serviceID string, isNew bool, err error) {
	// List all services and find by name
	services, err := dataaccess.ListServices(ctx, token)
	if err != nil {
		return "", false, err
	}

	for _, svc := range services.Services {
		if svc.Name == serviceName {
			return svc.Id, false, nil
		}
	}

	// Provide default description if empty
	if description == "" {
		description = fmt.Sprintf("Service for %s", serviceName)
	}

	// Service not found, create a new one
	serviceID, err = dataaccess.CreateService(ctx, token, serviceName, description)
	if err != nil {
		return "", false, err
	}

	return serviceID, true, nil
}

// findOrCreateEnvironment finds an environment by type or creates a new one
func findOrCreateEnvironment(ctx context.Context, token, serviceID, environmentName, environmentType string) (environmentID string, err error) {
	// Try to find existing environment by type
	env, err := dataaccess.FindEnvironment(ctx, token, serviceID, environmentType)
	if err == nil {
		return env.Id, nil
	}

	// If not found, create a new environment
	if errors.Is(err, dataaccess.ErrEnvironmentNotFound) {
		// Get default deployment config ID
		defaultDeploymentConfigID, err := dataaccess.GetDefaultDeploymentConfigID(ctx, token)
		if err != nil {
			return "", err
		}

		environmentID, err = dataaccess.CreateServiceEnvironment(
			ctx,
			token,
			environmentName,
			fmt.Sprintf("%s environment", environmentName),
			serviceID,
			"PRIVATE",
			strings.ToUpper(environmentType),
			nil,
			defaultDeploymentConfigID,
			true,
			nil,
		)
		if err != nil {
			return "", err
		}

		return environmentID, nil
	}

	return "", err
}

// findProductTierByName searches for a product tier by name across all service APIs and service models
func findProductTierByName(ctx context.Context, token, serviceID, environmentID, productTierName string) (productTierID, serviceAPIID, serviceModelID string, err error) {
	// List service APIs for this environment
	serviceAPIs, err := dataaccess.ListServiceAPIs(ctx, token, serviceID, environmentID)
	if err != nil {
		return "", "", "", err
	}

	// Search through each service API
	for _, apiID := range serviceAPIs.Ids {
		// List service models for this service API
		serviceModels, err := dataaccess.ListServiceModels(ctx, token, serviceID, apiID)
		if err != nil {
			continue // Skip on error, try next API
		}

		// Search through each service model
		for _, modelID := range serviceModels.Ids {
			// List product tiers for this service model
			productTiers, err := dataaccess.ListProductTiers(ctx, token, serviceID, modelID)
			if err != nil {
				continue // Skip on error, try next model
			}

			// Search for the product tier by name
			for _, tierID := range productTiers.Ids {
				tier, err := dataaccess.DescribeProductTier(ctx, token, serviceID, tierID)
				if err != nil {
					continue // Skip on error, try next tier
				}

				if tier.Name == productTierName {
					return tierID, apiID, modelID, nil
				}
			}
		}
	}

	// Product tier not found
	return "", "", "", nil
}

// findOrCreateServiceAPI finds an existing service API or creates a new one
func findOrCreateServiceAPI(ctx context.Context, token, serviceID, environmentID, name string) (serviceAPIID string, err error) {
	// List existing service APIs
	serviceAPIs, err := dataaccess.ListServiceAPIs(ctx, token, serviceID, environmentID)
	if err != nil {
		return "", err
	}

	// If there's an existing service API, use it
	if len(serviceAPIs.Ids) > 0 {
		return serviceAPIs.Ids[0], nil
	}

	// No service API exists, create a new one
	serviceAPIID, err = dataaccess.CreateServiceAPI(ctx, token, serviceID, environmentID, fmt.Sprintf("Service API for %s", name))
	if err != nil {
		return "", err
	}

	return serviceAPIID, nil
}

// findOrCreateServiceModel finds an existing service model matching the model type or creates a new one
// modelType: OMNISTRATE_HOSTED, BYOA, ON_PREM, ON_PREM_COPILOT
func findOrCreateServiceModel(ctx context.Context, token, serviceID, serviceAPIID, name, modelType string, accountConfigIDs []string) (serviceModelID string, err error) {
	// If we have a service API, list its service models and find one matching the model type
	if serviceAPIID != "" {
		serviceModels, listErr := dataaccess.ListServiceModels(ctx, token, serviceID, serviceAPIID)
		if listErr == nil {
			for _, modelID := range serviceModels.Ids {
				model, descErr := dataaccess.DescribeServiceModel(ctx, token, serviceID, modelID)
				if descErr != nil {
					continue
				}
				// Check if this service model matches the requested model type
				if model.ModelType == modelType {
					return modelID, nil
				}
			}
		}
	}

	// No matching service model found, create a new one
	serviceModelID, err = dataaccess.CreateServiceModel(ctx, token, serviceID, serviceAPIID, name+" Model", "Service model for "+name, modelType, accountConfigIDs)
	if err != nil {
		return "", err
	}

	return serviceModelID, nil
}

// DeploymentModelToServiceModelType converts deployment model type to service model type
func DeploymentModelToServiceModelType(deploymentModelType string) string {
	switch deploymentModelType {
	case DeploymentModelHosted:
		return ServiceModelTypeOmnistrateHosted
	case DeploymentModelCustomerHosted:
		return ServiceModelTypeCustomerHosted
	case DeploymentModelBYOA:
		return ServiceModelTypeBYOA
	case DeploymentModelOnPrem:
		return ServiceModelTypeOnPrem
	case DeploymentModelOnPremCopilot:
		return ServiceModelTypeOnPremCopilot
	default:
		return ""
	}
}

// findOrCreateProductTier finds an existing product tier by name or creates a new one
func findOrCreateProductTier(ctx context.Context, token, serviceID, serviceModelID, name, description, tierType string) (productTierID string, isNew bool, err error) {
	// List product tiers for this service model
	productTiers, err := dataaccess.ListProductTiers(ctx, token, serviceID, serviceModelID)
	if err == nil {
		// Search for the product tier by name
		for _, tierID := range productTiers.Ids {
			tier, descErr := dataaccess.DescribeProductTier(ctx, token, serviceID, tierID)
			if descErr != nil {
				continue // Skip on error, try next tier
			}
			if tier.Name == name {
				// Found existing product tier with matching name
				return tierID, false, nil
			}
		}
	}

	// Product tier not found, create a new one
	productTierID, err = dataaccess.CreateProductTier(ctx, token, serviceID, serviceModelID, name, description, tierType)
	if err != nil {
		return "", false, err
	}

	return productTierID, true, nil
}
