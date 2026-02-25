package build

import (
	"encoding/base64"
	"os"
	"path/filepath"
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

func TestParseServicePlanSpec_ExtractsArtifactPaths_FromTerraform(t *testing.T) {
	yamlContent := `
name: Terraform
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /
          artifactsLocalPath: /path/to/aws/artifacts
        gcp:
          terraformPath: /
          artifactsLocalPath: /path/to/gcp/artifacts
        azure:
          terraformPath: /
          artifactsLocalPath: /path/to/azure/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 3)
	assert.Contains(t, info.ArtifactPaths, "/path/to/aws/artifacts")
	assert.Contains(t, info.ArtifactPaths, "/path/to/gcp/artifacts")
	assert.Contains(t, info.ArtifactPaths, "/path/to/azure/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_Deduplicates(t *testing.T) {
	yamlContent := `
name: Terraform
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /
          artifactsLocalPath: /shared/artifacts
        gcp:
          terraformPath: /
          artifactsLocalPath: /shared/artifacts
        azure:
          terraformPath: /
          artifactsLocalPath: /shared/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	// Same path should be deduplicated
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/shared/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_FromMultipleServices(t *testing.T) {
	yamlContent := `
name: MultiService
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformService
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /
          artifactsLocalPath: /terraform/artifacts
  - name: helmService
    helmChartConfiguration:
      chartName: redis
      artifactsLocalPath: /helm/artifacts
  - name: kustomizeService
    kustomizeConfiguration:
      kustomizePath: /
      artifactsLocalPath: /kustomize/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 3)
	assert.Contains(t, info.ArtifactPaths, "/terraform/artifacts")
	assert.Contains(t, info.ArtifactPaths, "/helm/artifacts")
	assert.Contains(t, info.ArtifactPaths, "/kustomize/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_FromHelm(t *testing.T) {
	yamlContent := `
name: HelmService
deployment:
  hostedDeployment: {}
services:
  - name: redis
    helmChartConfiguration:
      chartName: redis
      artifactsLocalPath: /helm/redis/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/helm/redis/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_FromKustomize(t *testing.T) {
	yamlContent := `
name: KustomizeService
deployment:
  hostedDeployment: {}
services:
  - name: app
    kustomizeConfiguration:
      kustomizePath: /overlays/prod
      artifactsLocalPath: /kustomize/app/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/kustomize/app/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_FromOperator(t *testing.T) {
	yamlContent := `
name: OperatorService
deployment:
  hostedDeployment: {}
services:
  - name: operator
    operatorCRDConfiguration:
      crdName: myoperator
      artifactsLocalPath: /operator/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/operator/artifacts")
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_FallbackToTerraformPath(t *testing.T) {
	yamlContent := `
name: NoArtifacts
deployment:
  hostedDeployment: {}
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	// When artifactsLocalPath is not specified, terraformPath is used as fallback
	assert.NotNil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/")
	// Also check artifact uploads
	assert.Len(t, info.ArtifactUploads, 1)
	assert.Equal(t, "/", info.ArtifactUploads[0].Path)
	assert.Equal(t, "aws", info.ArtifactUploads[0].CloudProvider)
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_NoServicesSection(t *testing.T) {
	yamlContent := `
name: NoServices
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.Nil(t, info.ArtifactPaths)
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_MixedTerraformFallback(t *testing.T) {
	yamlContent := `
name: MixedTerraform
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /aws/tf
          artifactsLocalPath: /aws/artifacts
        gcp:
          terraformPath: /gcp/tf
        azure:
          terraformPath: /azure/tf
          artifactsLocalPath: /azure/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	// aws uses artifactsLocalPath, gcp falls back to terraformPath, azure uses artifactsLocalPath
	assert.Len(t, info.ArtifactPaths, 3)
	assert.Contains(t, info.ArtifactPaths, "/aws/artifacts")
	assert.Contains(t, info.ArtifactPaths, "/gcp/tf")
	assert.Contains(t, info.ArtifactPaths, "/azure/artifacts")
	// Artifact uploads should have 3 entries
	assert.Len(t, info.ArtifactUploads, 3)
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_TerraformFallbackDeduplicates(t *testing.T) {
	yamlContent := `
name: TerraformDedup
deployment:
  hostedDeployment: {}
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /shared
        gcp:
          terraformPath: /shared
        azure:
          terraformPath: /shared
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	// Same terraformPath should be deduplicated in ArtifactPaths
	assert.Len(t, info.ArtifactPaths, 1)
	assert.Contains(t, info.ArtifactPaths, "/shared")
	// But ArtifactUploads should have 3 entries (one per cloud provider)
	assert.Len(t, info.ArtifactUploads, 3)
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_SkipFallbackWithGitConfig(t *testing.T) {
	yamlContent := `
name: TerraformGit
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform-spec
          gitConfiguration:
            reference: refs/tags/2.3
            repositoryUrl: https://github.com/example/repo.git
        gcp:
          terraformPath: /terraform-spec
          gitConfiguration:
            reference: refs/tags/2.3
            repositoryUrl: https://github.com/example/repo.git
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	// gitConfiguration is present, so terraformPath should NOT be used as fallback
	assert.Nil(t, info.ArtifactPaths)
	assert.Len(t, info.ArtifactUploads, 0)
}

func TestParseServicePlanSpec_ExtractsArtifactPaths_MixedGitAndLocal(t *testing.T) {
	yamlContent := `
name: TerraformMixed
deployment:
  hostedDeployment:
    AwsAccountId: "111111111111"
services:
  - name: terraformResource
    type: terraform
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform-spec
          gitConfiguration:
            reference: refs/tags/2.3
            repositoryUrl: https://github.com/example/repo.git
        gcp:
          terraformPath: /local/path
        azure:
          terraformPath: /azure/tf
          artifactsLocalPath: /azure/artifacts
`
	info, err := ParseServicePlanSpec([]byte(yamlContent))
	require.NoError(t, err)
	assert.NotNil(t, info.ArtifactPaths)
	// aws has gitConfiguration so skipped, gcp falls back to terraformPath, azure uses artifactsLocalPath
	assert.Len(t, info.ArtifactPaths, 2)
	assert.Contains(t, info.ArtifactPaths, "/local/path")
	assert.Contains(t, info.ArtifactPaths, "/azure/artifacts")
	assert.Len(t, info.ArtifactUploads, 2)
}

func TestArchiveArtifactPaths_CreatesBase64Archive(t *testing.T) {
	// Create a temporary source directory with some files
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a subdirectory to archive
	artifactDir := filepath.Join(sourceDir, "artifacts")
	err = os.MkdirAll(artifactDir, 0755)
	require.NoError(t, err)

	// Create some test files
	err = os.WriteFile(filepath.Join(artifactDir, "test.txt"), []byte("test content"), 0644)
	require.NoError(t, err)

	// Archive the directory
	result, err := ArchiveArtifactPaths(sourceDir, []string{"artifacts"})
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "artifacts")

	// Verify the base64 content is valid
	base64Content := result["artifacts"]
	assert.NotEmpty(t, base64Content)

	// Verify it can be decoded
	decoded, err := base64.StdEncoding.DecodeString(base64Content)
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}

func TestArchiveArtifactPaths_MultipleDirectories(t *testing.T) {
	// Create a temporary source directory
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create subdirectories
	err = os.MkdirAll(filepath.Join(sourceDir, "dir1"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "dir1", "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(sourceDir, "dir2"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "dir2", "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	// Archive multiple directories
	result, err := ArchiveArtifactPaths(sourceDir, []string{"dir1", "dir2"})
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "dir1")
	assert.Contains(t, result, "dir2")
}

func TestArchiveArtifactPaths_EmptyPaths(t *testing.T) {
	result, err := ArchiveArtifactPaths("/tmp", []string{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArchiveArtifactPaths_NilPaths(t *testing.T) {
	result, err := ArchiveArtifactPaths("/tmp", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArchiveArtifactPaths_NonExistentPath(t *testing.T) {
	_, err := ArchiveArtifactPaths("/tmp", []string{"/non/existent/path/that/does/not/exist"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestArchiveArtifactPaths_FileNotDirectory(t *testing.T) {
	// Create a temporary file (not a directory)
	tmpFile, err := os.CreateTemp("", "test-file-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = ArchiveArtifactPaths(filepath.Dir(tmpFile.Name()), []string{filepath.Base(tmpFile.Name())})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestArchiveArtifactPaths_NestedDirectories(t *testing.T) {
	// Create a temporary source directory with nested structure
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a nested directory structure
	nestedDir := filepath.Join(sourceDir, "artifacts", "subdir1", "subdir2")
	err = os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create files at different levels
	err = os.WriteFile(filepath.Join(sourceDir, "artifacts", "root.txt"), []byte("root"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "artifacts", "subdir1", "level1.txt"), []byte("level1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nestedDir, "level2.txt"), []byte("level2"), 0644)
	require.NoError(t, err)

	// Archive the directory
	result, err := ArchiveArtifactPaths(sourceDir, []string{"artifacts"})
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "artifacts")

	// Verify the content is valid base64
	decoded, err := base64.StdEncoding.DecodeString(result["artifacts"])
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}
