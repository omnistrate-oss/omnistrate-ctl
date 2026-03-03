package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"gopkg.in/yaml.v3"
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

// Tenancy type constants
const (
	TenancyTypeOmnistrateMulti     = "OMNISTRATE_MULTI_TENANCY"
	TenancyTypeOmnistrateDedicated = "OMNISTRATE_DEDICATED_TENANCY"
	TenancyTypeCustom              = "CUSTOM_TENANCY"
)

// Deployment model constants
const (
	DeploymentModelHosted        = "HOSTED"
	DeploymentModelBYOA          = "BYOA"
	DeploymentModelOnPrem        = "ON_PREM"
	DeploymentModelOnPremCopilot = "ON_PREM_COPILOT"
)

// ServicePlanSpecInfo holds information extracted from a service plan spec YAML
type ServicePlanSpecInfo struct {
	ProductTierName        string
	TenancyType            string // defaults to "CUSTOM_TENANCY" for service plan spec
	DeploymentModelType    string
	Features               map[string]interface{}
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

// MatchedAccountConfigs holds matched account configurations for each cloud provider
type MatchedAccountConfigs struct {
	AWS   *AccountMatchResult
	GCP   *AccountMatchResult
	Azure *AccountMatchResult
	OCI   *AccountMatchResult
}

// AccountMatchResult holds the result of matching an account config
type AccountMatchResult struct {
	AccountConfigID string
	CloudProvider   string
	Matched         bool
	Error           string
}

// ParseServicePlanSpec parses the service plan spec YAML and extracts relevant information
func ParseServicePlanSpec(fileData []byte) (*ServicePlanSpecInfo, error) {
	var specData map[string]interface{}
	if err := yaml.Unmarshal(fileData, &specData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service plan spec: %w", err)
	}

	info := &ServicePlanSpecInfo{
		TenancyType: TenancyTypeCustom, // defaults for service plan spec
		Features:    make(map[string]interface{}),
	}

	// Extract top-level name (product tier name)
	if name, ok := specData["name"].(string); ok {
		info.ProductTierName = name
	}

	// Extract features if present
	if features, ok := specData["features"].(map[string]interface{}); ok {
		info.Features = extractFeatures(features)
	}

	// Extract deployment information
	if deployment, ok := specData["deployment"].(map[string]interface{}); ok {
		extractDeploymentInfo(deployment, info)
	}

	return info, nil
}

// extractFeatures extracts the features map from the spec
func extractFeatures(features map[string]interface{}) map[string]interface{} {
	extracted := make(map[string]interface{})
	for key, value := range features {
		extracted[key] = value
	}
	return extracted
}

// extractDeploymentInfo extracts deployment model and cloud provider account information
func extractDeploymentInfo(deployment map[string]interface{}, info *ServicePlanSpecInfo) {
	// Check for different deployment types
	if hostedDeployment, ok := deployment["hostedDeployment"].(map[string]interface{}); ok {
		info.DeploymentModelType = DeploymentModelHosted
		extractAccountFromMap(hostedDeployment, info)
	} else if byoaDeployment, ok := deployment["byoaDeployment"].(map[string]interface{}); ok {
		info.DeploymentModelType = DeploymentModelBYOA
		extractAccountFromMap(byoaDeployment, info)
	} else if onPremDeployment, ok := deployment["onPremDeployment"].(map[string]interface{}); ok {
		info.DeploymentModelType = DeploymentModelOnPrem
		extractAccountFromMap(onPremDeployment, info)
	} else if onPremCopilotDeployment, ok := deployment["onPremCopilotDeployment"].(map[string]interface{}); ok {
		info.DeploymentModelType = DeploymentModelOnPremCopilot
		extractAccountFromMap(onPremCopilotDeployment, info)
	}
}

// extractAccountFromMap extracts cloud provider account information from a deployment map
func extractAccountFromMap(deploymentMap map[string]interface{}, info *ServicePlanSpecInfo) {
	// Extract AWS account info
	if awsAccountID, ok := deploymentMap["AwsAccountId"].(string); ok {
		info.AwsAccountID = awsAccountID
	}
	if awsBootstrapRoleARN, ok := deploymentMap["AwsBootstrapRoleAccountArn"].(string); ok {
		info.AwsBootstrapRoleARN = awsBootstrapRoleARN
	}

	// Extract GCP account info
	if gcpProjectID, ok := deploymentMap["GcpProjectId"].(string); ok {
		info.GcpProjectID = gcpProjectID
	}
	if gcpProjectNumber, ok := deploymentMap["GcpProjectNumber"].(string); ok {
		info.GcpProjectNumber = gcpProjectNumber
	}
	if gcpServiceAccountEmail, ok := deploymentMap["GcpServiceAccountEmail"].(string); ok {
		info.GcpServiceAccountEmail = gcpServiceAccountEmail
	}

	// Extract Azure account info
	if azureSubscriptionID, ok := deploymentMap["AzureSubscriptionId"].(string); ok {
		info.AzureSubscriptionID = azureSubscriptionID
	}
	if azureTenantID, ok := deploymentMap["AzureTenantId"].(string); ok {
		info.AzureTenantID = azureTenantID
	}

	// Extract OCI account info
	if ociTenancyID, ok := deploymentMap["OCITenancyId"].(string); ok {
		info.OCITenancyID = ociTenancyID
	}
	if ociDomainID, ok := deploymentMap["OCIDomainId"].(string); ok {
		info.OCIDomainID = ociDomainID
	}
}

// FindMatchingAccountConfigs finds existing account configs that match the spec requirements
func FindMatchingAccountConfigs(ctx context.Context, token string, specInfo *ServicePlanSpecInfo) (*MatchedAccountConfigs, error) {
	matched := &MatchedAccountConfigs{}

	// Get all account configs from Omnistrate
	allAccounts, err := dataaccess.ListAllAccountConfigs(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to list account configs: %w", err)
	}

	// Match AWS account
	if specInfo.AwsAccountID != "" {
		matched.AWS = matchAWSAccount(allAccounts.AccountConfigs, specInfo)
	}

	// Match GCP account
	if specInfo.GcpProjectID != "" || specInfo.GcpProjectNumber != "" {
		matched.GCP = matchGCPAccount(allAccounts.AccountConfigs, specInfo)
	}

	// Match Azure account
	if specInfo.AzureSubscriptionID != "" {
		matched.Azure = matchAzureAccount(allAccounts.AccountConfigs, specInfo)
	}

	// Match OCI account
	if specInfo.OCITenancyID != "" {
		matched.OCI = matchOCIAccount(allAccounts.AccountConfigs, specInfo)
	}

	return matched, nil
}

// matchAWSAccount finds a matching AWS account config
func matchAWSAccount(accounts []openapiclientfleet.FleetDescribeAccountConfigResult, specInfo *ServicePlanSpecInfo) *AccountMatchResult {
	for _, account := range accounts {
		if strings.ToLower(account.CloudProviderId) == "aws" {
			// Check if account details match
			if account.AwsAccountID != nil && *account.AwsAccountID == specInfo.AwsAccountID {
				return &AccountMatchResult{
					AccountConfigID: account.Id,
					CloudProvider:   "aws",
					Matched:         true,
				}
			}
		}
	}
	return &AccountMatchResult{
		CloudProvider: "aws",
		Matched:       false,
		Error:         fmt.Sprintf("No matching AWS account config found for account ID: %s", specInfo.AwsAccountID),
	}
}

// matchGCPAccount finds a matching GCP account config
func matchGCPAccount(accounts []openapiclientfleet.FleetDescribeAccountConfigResult, specInfo *ServicePlanSpecInfo) *AccountMatchResult {
	for _, account := range accounts {
		if strings.ToLower(account.CloudProviderId) == "gcp" {
			// Match by project ID or project number
			if (account.GcpProjectID != nil && *account.GcpProjectID == specInfo.GcpProjectID) ||
				(account.GcpProjectNumber != nil && *account.GcpProjectNumber == specInfo.GcpProjectNumber) {
				return &AccountMatchResult{
					AccountConfigID: account.Id,
					CloudProvider:   "gcp",
					Matched:         true,
				}
			}
		}
	}
	return &AccountMatchResult{
		CloudProvider: "gcp",
		Matched:       false,
		Error:         fmt.Sprintf("No matching GCP account config found for project ID: %s or project number: %s", specInfo.GcpProjectID, specInfo.GcpProjectNumber),
	}
}

// matchAzureAccount finds a matching Azure account config
func matchAzureAccount(accounts []openapiclientfleet.FleetDescribeAccountConfigResult, specInfo *ServicePlanSpecInfo) *AccountMatchResult {
	for _, account := range accounts {
		if strings.ToLower(account.CloudProviderId) == "azure" {
			if account.AzureSubscriptionID != nil && *account.AzureSubscriptionID == specInfo.AzureSubscriptionID {
				return &AccountMatchResult{
					AccountConfigID: account.Id,
					CloudProvider:   "azure",
					Matched:         true,
				}
			}
		}
	}
	return &AccountMatchResult{
		CloudProvider: "azure",
		Matched:       false,
		Error:         fmt.Sprintf("No matching Azure account config found for subscription ID: %s", specInfo.AzureSubscriptionID),
	}
}

// matchOCIAccount finds a matching OCI account config
func matchOCIAccount(accounts []openapiclientfleet.FleetDescribeAccountConfigResult, specInfo *ServicePlanSpecInfo) *AccountMatchResult {
	for _, account := range accounts {
		if strings.ToLower(account.CloudProviderId) == "oci" {
			// Note: The SDK may not have OCI fields yet, so this might need updating
			// when OCI support is fully added to the SDK
			// For now, return not matched
			// TODO: Update when SDK adds OCI fields to FleetDescribeAccountConfigResult
			return &AccountMatchResult{
				CloudProvider: "oci",
				Matched:       false,
				Error:         "OCI account matching not yet supported in SDK",
			}
		}
	}
	return &AccountMatchResult{
		CloudProvider: "oci",
		Matched:       false,
		Error:         fmt.Sprintf("No matching OCI account config found for tenancy ID: %s", specInfo.OCITenancyID),
	}
}
