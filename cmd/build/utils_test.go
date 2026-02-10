package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServicePlanSpec_ServicePlanSpecType_ExtractsProductTierName(t *testing.T) {
	yamlContent := `
name: Terraform
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Equal(t, "Terraform", info.ProductTierName)
	assert.Equal(t, TenancyTypeCustom, info.TenancyType)
}

func TestParseServicePlanSpec_ServicePlanSpecType_ExtractsAWSAccountConfig(t *testing.T) {
	yamlContent := `
name: TestPlan
deployment:
  hostedDeployment:
    AwsAccountId: "333333333333"
    AwsBootstrapRoleAccountArn: arn:aws:iam::333333333333:role/test-bootstrap-role
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Equal(t, "333333333333", info.AwsAccountID)
	assert.Equal(t, "arn:aws:iam::333333333333:role/test-bootstrap-role", info.AwsBootstrapRoleARN)
	// hostedDeployment with account info is CUSTOMER_HOSTED
	assert.Equal(t, DeploymentModelCustomerHosted, info.DeploymentModelType)
}

func TestParseServicePlanSpec_ServicePlanSpecType_ExtractsGCPAccountConfig(t *testing.T) {
	yamlContent := `
name: TestPlan
deployment:
  hostedDeployment:
    GcpProjectId: 'test-gcp-project-dev'
    GcpProjectNumber: '12345678901'
    GcpServiceAccountEmail: 'test-sa@test-gcp-project-dev.iam.gserviceaccount.com'
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Equal(t, "test-gcp-project-dev", info.GcpProjectID)
	assert.Equal(t, "12345678901", info.GcpProjectNumber)
	assert.Equal(t, "test-sa@test-gcp-project-dev.iam.gserviceaccount.com", info.GcpServiceAccountEmail)
}

func TestParseServicePlanSpec_ServicePlanSpecType_ExtractsOCIAccountConfig(t *testing.T) {
	yamlContent := `
name: TestPlan
deployment:
  hostedDeployment:
    AwsAccountId: "444444444444"
    AWSBootstrapRoleAccountArn: arn:aws:iam::444444444444:role/test-bootstrap-role
    OCITenancyId: "ocid1.tenancy.oc1..aaaaaaaa1111111111111111111111111111111111111111111111111111"
    OCIDomainId: "ocid1.domain.oc1..aaaaaaaa2222222222222222222222222222222222222222222222222222"
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Equal(t, "ocid1.tenancy.oc1..aaaaaaaa1111111111111111111111111111111111111111111111111111", info.OCITenancyID)
	assert.Equal(t, "ocid1.domain.oc1..aaaaaaaa2222222222222222222222222222222222222222222222222222", info.OCIDomainID)
}

func TestParseServicePlanSpec_ServicePlanSpecType_HandlesByoaDeployment(t *testing.T) {
	yamlContent := `
name: TestPlan
deployment:
  byoaDeployment:
    AwsAccountId: "555555555555"
    OCITenancyId: "ocid1.tenancy.oc1..testbyoa"
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Equal(t, "555555555555", info.AwsAccountID)
	assert.Equal(t, "ocid1.tenancy.oc1..testbyoa", info.OCITenancyID)
	assert.Equal(t, DeploymentModelBYOA, info.DeploymentModelType)
}

func TestMatchedAccountConfigs_HasAnyAccountConfigID(t *testing.T) {
	tests := []struct {
		name     string
		matched  MatchedAccountConfigs
		expected bool
	}{
		{
			name:     "empty",
			matched:  MatchedAccountConfigs{},
			expected: false,
		},
		{
			name:     "aws only",
			matched:  MatchedAccountConfigs{AwsAccountConfigID: "acc-123"},
			expected: true,
		},
		{
			name:     "gcp only",
			matched:  MatchedAccountConfigs{GcpAccountConfigID: "acc-456"},
			expected: true,
		},
		{
			name:     "azure only",
			matched:  MatchedAccountConfigs{AzureAccountConfigID: "acc-789"},
			expected: true,
		},
		{
			name:     "oci only",
			matched:  MatchedAccountConfigs{OciAccountConfigID: "acc-oci"},
			expected: true,
		},
		{
			name: "all providers",
			matched: MatchedAccountConfigs{
				AwsAccountConfigID:   "acc-aws",
				GcpAccountConfigID:   "acc-gcp",
				AzureAccountConfigID: "acc-azure",
				OciAccountConfigID:   "acc-oci",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.matched.HasAnyAccountConfigID())
		})
	}
}

func TestMatchedAccountConfigs_ToSlice(t *testing.T) {
	tests := []struct {
		name     string
		matched  MatchedAccountConfigs
		expected []string
	}{
		{
			name:     "empty",
			matched:  MatchedAccountConfigs{},
			expected: nil,
		},
		{
			name:     "aws only",
			matched:  MatchedAccountConfigs{AwsAccountConfigID: "acc-aws"},
			expected: []string{"acc-aws"},
		},
		{
			name: "all providers",
			matched: MatchedAccountConfigs{
				AwsAccountConfigID:   "acc-aws",
				GcpAccountConfigID:   "acc-gcp",
				AzureAccountConfigID: "acc-azure",
				OciAccountConfigID:   "acc-oci",
			},
			expected: []string{"acc-aws", "acc-gcp", "acc-azure", "acc-oci"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.matched.ToSlice())
		})
	}
}

func TestDetectSpecType_TerraformSpec(t *testing.T) {
	yamlContent := map[string]interface{}{
		"name": "Terraform",
		"services": []interface{}{
			map[string]interface{}{
				"name": "terraformResource",
				"type": "terraform",
				"terraformConfigurations": map[string]interface{}{
					"configurationPerCloudProvider": map[string]interface{}{
						"aws": map[string]interface{}{
							"terraformPath": "/",
						},
					},
				},
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, ServicePlanSpecType, specType)
}

func TestDetectSpecType_HelmSpec(t *testing.T) {
	yamlContent := map[string]interface{}{
		"name": "Redis",
		"services": []interface{}{
			map[string]interface{}{
				"name": "redis",
				"helmChartConfiguration": map[string]interface{}{
					"chartName": "redis",
				},
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, ServicePlanSpecType, specType)
}

func TestDetectSpecType_DockerCompose(t *testing.T) {
	yamlContent := map[string]interface{}{
		"services": map[string]interface{}{
			"web": map[string]interface{}{
				"image": "nginx",
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, DockerComposeSpecType, specType)
}

func TestContainsOmnistrateKey(t *testing.T) {
	tests := []struct {
		name     string
		content  map[string]interface{}
		expected bool
	}{
		{
			name: "has x-omnistrate-service-plan",
			content: map[string]interface{}{
				"x-omnistrate-service-plan": map[string]interface{}{
					"name": "test",
				},
			},
			expected: true,
		},
		{
			name: "has x-omnistrate-hosted",
			content: map[string]interface{}{
				"x-omnistrate-hosted": map[string]interface{}{
					"AwsAccountId": "123",
				},
			},
			expected: true,
		},
		{
			name: "no omnistrate keys",
			content: map[string]interface{}{
				"services": map[string]interface{}{
					"web": map[string]interface{}{
						"image": "nginx",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ContainsOmnistrateKey(tt.content))
		})
	}
}
