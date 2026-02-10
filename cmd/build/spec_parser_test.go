package build

import (
	"testing"
)

func TestParseServicePlanSpec(t *testing.T) {
	specYAML := `
name: TestService
deployment:
  hostedDeployment:
    AwsAccountId: "123456789012"
    AwsBootstrapRoleAccountArn: "arn:aws:iam::123456789012:role/test-role"
    GcpProjectId: "test-project"
    GcpProjectNumber: "123456789"
    AzureSubscriptionId: "azure-sub-id"
    AzureTenantId: "azure-tenant-id"
    OCITenancyId: "oci-tenancy-id"
    OCIDomainId: "oci-domain-id"
features:
  CUSTOM_DEPLOYMENT_CELL_PLACEMENT:
    maximumDeploymentsPerCell: -1
services:
  - name: testResource
    type: terraform
`

	info, err := ParseServicePlanSpec([]byte(specYAML))
	if err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Verify parsed values
	if info.ProductTierName != "TestService" {
		t.Errorf("Expected ProductTierName 'TestService', got '%s'", info.ProductTierName)
	}

	if info.TenancyType != TenancyTypeCustom {
		t.Errorf("Expected TenancyType '%s', got '%s'", TenancyTypeCustom, info.TenancyType)
	}

	if info.DeploymentModelType != DeploymentModelHosted {
		t.Errorf("Expected DeploymentModelType '%s', got '%s'", DeploymentModelHosted, info.DeploymentModelType)
	}

	// Verify AWS account info
	if info.AwsAccountID != "123456789012" {
		t.Errorf("Expected AwsAccountID '123456789012', got '%s'", info.AwsAccountID)
	}

	if info.AwsBootstrapRoleARN != "arn:aws:iam::123456789012:role/test-role" {
		t.Errorf("Expected AwsBootstrapRoleARN 'arn:aws:iam::123456789012:role/test-role', got '%s'", info.AwsBootstrapRoleARN)
	}

	// Verify GCP account info
	if info.GcpProjectID != "test-project" {
		t.Errorf("Expected GcpProjectID 'test-project', got '%s'", info.GcpProjectID)
	}

	if info.GcpProjectNumber != "123456789" {
		t.Errorf("Expected GcpProjectNumber '123456789', got '%s'", info.GcpProjectNumber)
	}

	// Verify Azure account info
	if info.AzureSubscriptionID != "azure-sub-id" {
		t.Errorf("Expected AzureSubscriptionID 'azure-sub-id', got '%s'", info.AzureSubscriptionID)
	}

	if info.AzureTenantID != "azure-tenant-id" {
		t.Errorf("Expected AzureTenantID 'azure-tenant-id', got '%s'", info.AzureTenantID)
	}

	// Verify OCI account info
	if info.OCITenancyID != "oci-tenancy-id" {
		t.Errorf("Expected OCITenancyID 'oci-tenancy-id', got '%s'", info.OCITenancyID)
	}

	if info.OCIDomainID != "oci-domain-id" {
		t.Errorf("Expected OCIDomainID 'oci-domain-id', got '%s'", info.OCIDomainID)
	}

	// Verify features
	if len(info.Features) == 0 {
		t.Error("Expected features to be parsed, got empty map")
	}

	if _, ok := info.Features["CUSTOM_DEPLOYMENT_CELL_PLACEMENT"]; !ok {
		t.Error("Expected CUSTOM_DEPLOYMENT_CELL_PLACEMENT feature to be present")
	}
}

func TestParseServicePlanSpecByoaDeployment(t *testing.T) {
	specYAML := `
name: ByoaService
deployment:
  byoaDeployment:
    AwsAccountId: "987654321098"
services:
  - name: byoaResource
    type: helm
`

	info, err := ParseServicePlanSpec([]byte(specYAML))
	if err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	if info.DeploymentModelType != DeploymentModelBYOA {
		t.Errorf("Expected DeploymentModelType '%s', got '%s'", DeploymentModelBYOA, info.DeploymentModelType)
	}

	if info.AwsAccountID != "987654321098" {
		t.Errorf("Expected AwsAccountID '987654321098', got '%s'", info.AwsAccountID)
	}
}
