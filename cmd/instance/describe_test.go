package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
)

func TestDescribeCommand(t *testing.T) {
	// Test that the command is properly registered
	assert.NotNil(t, describeCmd)
	assert.Equal(t, "describe [instance-id]", describeCmd.Use)
	assert.Contains(t, describeCmd.Short, "instance deployment")
}

func TestDescribeCommandFlags(t *testing.T) {
	// Test that describe command has expected flags
	assert.NotNil(t, describeCmd.Flag("output"))
	assert.NotNil(t, describeCmd.Flag("resource-id"))
	assert.NotNil(t, describeCmd.Flag("resource-key"))
	assert.NotNil(t, describeCmd.Flag("deployment-status"))

	// Test output flag default value
	outputFlag := describeCmd.Flag("output")
	assert.Equal(t, "json", outputFlag.DefValue, "output flag should default to json")

	// Test deployment-status flag default value
	deploymentStatusFlag := describeCmd.Flag("deployment-status")
	assert.Equal(t, "false", deploymentStatusFlag.DefValue, "deployment-status flag should default to false")

	// Test deployment-status flag type
	assert.Equal(t, "bool", deploymentStatusFlag.Value.Type(), "deployment-status flag should be bool type")
}

func TestInstanceDeploymentStatusStructure(t *testing.T) {
	// Test the InstanceDeploymentStatus structure
	status := InstanceDeploymentStatus{
		InstanceID:               "instance-123",
		ServiceID:                "service-456",
		EnvironmentID:            "env-789",
		Status:                   "RUNNING",
		ProductTierID:            "tier-abc",
		TierVersion:              "v1.0.0",
		CreationTime:             "2024-01-15T10:00:00Z",
		LastModifiedTime:         "2024-01-15T11:00:00Z",
		ResourceDeploymentStatus: []ResourceDeploymentStatus{},
		AppliedFilters:           map[string]interface{}{"resourceId": "r-123"},
		FilteringStats:           map[string]interface{}{"totalResourceVersionSummaries": 5},
	}

	assert.Equal(t, "instance-123", status.InstanceID)
	assert.Equal(t, "service-456", status.ServiceID)
	assert.Equal(t, "env-789", status.EnvironmentID)
	assert.Equal(t, "RUNNING", status.Status)
	assert.Equal(t, "tier-abc", status.ProductTierID)
	assert.Equal(t, "v1.0.0", status.TierVersion)
	assert.Equal(t, "2024-01-15T10:00:00Z", status.CreationTime)
	assert.Equal(t, "2024-01-15T11:00:00Z", status.LastModifiedTime)
	assert.NotNil(t, status.ResourceDeploymentStatus)
	assert.NotNil(t, status.AppliedFilters)
	assert.NotNil(t, status.FilteringStats)
}

func TestResourceDeploymentStatusStructure(t *testing.T) {
	// Test the ResourceDeploymentStatus structure
	podStatus := map[string]string{
		"pod-1": "Running",
		"pod-2": "Pending",
	}
	additionalInfo := map[string]interface{}{
		"image":            "nginx:latest",
		"podToHostMapping": map[string]string{"pod-1": "node-1"},
	}

	status := ResourceDeploymentStatus{
		ResourceID:       "r-123",
		ResourceName:     "database",
		Version:          "v1.2.3",
		LatestVersion:    "v1.3.0",
		PodStatus:        podStatus,
		DeploymentErrors: "Connection timeout",
		DeploymentType:   "Generic",
		AdditionalInfo:   additionalInfo,
	}

	assert.Equal(t, "r-123", status.ResourceID)
	assert.Equal(t, "database", status.ResourceName)
	assert.Equal(t, "v1.2.3", status.Version)
	assert.Equal(t, "v1.3.0", status.LatestVersion)
	assert.Equal(t, podStatus, status.PodStatus)
	assert.Equal(t, "Connection timeout", status.DeploymentErrors)
	assert.Equal(t, "Generic", status.DeploymentType)
	assert.Equal(t, additionalInfo, status.AdditionalInfo)
}

func TestCreateResourceDeploymentStatus(t *testing.T) {
	// Test creating status for Generic deployment
	resourceId := "r-123"
	resourceName := "web-server"
	version := "v1.0.0"
	latestVersion := "v1.1.0"
	image := "nginx:latest"
	podStatus := map[string]string{"web-server-pod": "Running"}
	podToHostMapping := map[string]string{"web-server-pod": "node-1"}

	genericSummary := openapiclientfleet.ResourceVersionSummary{
		ResourceId:    &resourceId,
		ResourceName:  &resourceName,
		Version:       &version,
		LatestVersion: &latestVersion,
		GenericResourceDeploymentConfiguration: &openapiclientfleet.GenericResourceDeploymentConfiguration{
			Image:            &image,
			PodStatus:        &podStatus,
			PodToHostMapping: &podToHostMapping,
		},
	}

	status := createResourceDeploymentStatus(genericSummary)

	assert.Equal(t, "r-123", status.ResourceID)
	assert.Equal(t, "web-server", status.ResourceName)
	assert.Equal(t, "v1.0.0", status.Version)
	assert.Equal(t, "v1.1.0", status.LatestVersion)
	assert.Equal(t, "Generic", status.DeploymentType)
	assert.Equal(t, podStatus, status.PodStatus)
	assert.NotNil(t, status.AdditionalInfo)
	assert.Equal(t, "nginx:latest", status.AdditionalInfo["image"])
	assert.Equal(t, podToHostMapping, status.AdditionalInfo["podToHostMapping"])

	// Test creating status for Helm deployment
	deploymentErrors := "Helm chart failed to deploy"
	chartName := "nginx"
	chartVersion := "1.2.3"
	releaseName := "my-nginx"
	releaseNamespace := "default"
	releaseStatus := "failed"
	repositoryURL := "https://charts.bitnami.com/bitnami"

	helmSummary := openapiclientfleet.ResourceVersionSummary{
		ResourceId:    &resourceId,
		ResourceName:  &resourceName,
		Version:       &version,
		LatestVersion: &latestVersion,
		HelmDeploymentConfiguration: &openapiclientfleet.HelmDeploymentConfiguration{
			ChartName:        chartName,
			ChartVersion:     chartVersion,
			DeploymentErrors: &deploymentErrors,
			PodStatus:        &podStatus,
			ReleaseName:      releaseName,
			ReleaseNamespace: releaseNamespace,
			ReleaseStatus:    releaseStatus,
			RepositoryURL:    repositoryURL,
		},
	}

	helmStatus := createResourceDeploymentStatus(helmSummary)

	assert.Equal(t, "r-123", helmStatus.ResourceID)
	assert.Equal(t, "web-server", helmStatus.ResourceName)
	assert.Equal(t, "Helm", helmStatus.DeploymentType)
	assert.Equal(t, "Helm chart failed to deploy", helmStatus.DeploymentErrors)
	assert.Equal(t, podStatus, helmStatus.PodStatus)
	assert.NotNil(t, helmStatus.AdditionalInfo)
	assert.Equal(t, "nginx", helmStatus.AdditionalInfo["chartName"])
	assert.Equal(t, "1.2.3", helmStatus.AdditionalInfo["chartVersion"])
	assert.Equal(t, "my-nginx", helmStatus.AdditionalInfo["releaseName"])
	assert.Equal(t, "failed", helmStatus.AdditionalInfo["releaseStatus"])

	// Test creating status for Terraform deployment
	terraformErrors := "Terraform apply failed"
	configFiles := map[string]string{"main.tf": "resource \"aws_instance\" {}"}

	terraformSummary := openapiclientfleet.ResourceVersionSummary{
		ResourceId:    &resourceId,
		ResourceName:  &resourceName,
		Version:       &version,
		LatestVersion: &latestVersion,
		TerraformDeploymentConfiguration: &openapiclientfleet.TerraformDeploymentConfiguration{
			DeploymentErrors:   &terraformErrors,
			ConfigurationFiles: &configFiles,
		},
	}

	terraformStatus := createResourceDeploymentStatus(terraformSummary)

	assert.Equal(t, "r-123", terraformStatus.ResourceID)
	assert.Equal(t, "web-server", terraformStatus.ResourceName)
	assert.Equal(t, "Terraform", terraformStatus.DeploymentType)
	assert.Equal(t, "Terraform apply failed", terraformStatus.DeploymentErrors)
	assert.NotNil(t, terraformStatus.AdditionalInfo)
	assert.Equal(t, configFiles, terraformStatus.AdditionalInfo["configurationFiles"])
}

func TestFilterResourceVersionSummariesForStatusLocal(t *testing.T) {
	// Create test data
	resourceId1 := "r-123"
	resourceName1 := "database"
	resourceId2 := "r-456"
	resourceName2 := "webserver"

	summaries := []openapiclientfleet.ResourceVersionSummary{
		{
			ResourceId:   &resourceId1,
			ResourceName: &resourceName1,
		},
		{
			ResourceId:   &resourceId2,
			ResourceName: &resourceName2,
		},
	}

	// Test case 1: No filtering (empty filters)
	filteredSummaries, filterInfo, countInfo := filterResourceVersionSummariesLocal(summaries, "", "")

	assert.Equal(t, len(summaries), len(filteredSummaries), "Should return all summaries when no filters applied")
	assert.Nil(t, filterInfo, "Filter info should be nil when no filters applied")
	assert.Nil(t, countInfo, "Count info should be nil when no filters applied")

	// Test case 2: Filter by resource ID
	filteredSummaries2, filterInfo2, countInfo2 := filterResourceVersionSummariesLocal(summaries, "r-123", "")

	assert.Equal(t, 1, len(filteredSummaries2), "Should return only matching resource")
	assert.Equal(t, "r-123", *filteredSummaries2[0].ResourceId, "Should return the correct resource")
	assert.NotNil(t, filterInfo2)
	assert.Equal(t, "r-123", filterInfo2["resourceId"])
	assert.NotNil(t, countInfo2)
	assert.Equal(t, 2, countInfo2["totalResourceVersionSummaries"])
	assert.Equal(t, 1, countInfo2["filteredResourceVersionSummaries"])

	// Test case 3: Filter by resource key/name
	filteredSummaries3, filterInfo3, countInfo3 := filterResourceVersionSummariesLocal(summaries, "", "webserver")

	assert.Equal(t, 1, len(filteredSummaries3), "Should return only matching resource")
	assert.Equal(t, "webserver", *filteredSummaries3[0].ResourceName, "Should return the correct resource")
	assert.NotNil(t, filterInfo3)
	assert.Equal(t, "webserver", filterInfo3["resourceKey"])
	assert.NotNil(t, countInfo3)
	assert.Equal(t, 2, countInfo3["totalResourceVersionSummaries"])
	assert.Equal(t, 1, countInfo3["filteredResourceVersionSummaries"])

	// Test case 4: Filter by non-existent resource
	filteredSummaries4, filterInfo4, countInfo4 := filterResourceVersionSummariesLocal(summaries, "r-999", "")

	assert.Equal(t, 0, len(filteredSummaries4), "Should return no resources for non-matching filter")
	assert.NotNil(t, filterInfo4)
	assert.Equal(t, "r-999", filterInfo4["resourceId"])
	assert.NotNil(t, countInfo4)
	assert.Equal(t, 2, countInfo4["totalResourceVersionSummaries"])
	assert.Equal(t, 0, countInfo4["filteredResourceVersionSummaries"])
}

// Helper function for testing filtering logic without external dependencies
func filterResourceVersionSummariesLocal(summaries []openapiclientfleet.ResourceVersionSummary, resourceID, resourceKey string) ([]openapiclientfleet.ResourceVersionSummary, map[string]interface{}, map[string]interface{}) {
	var filteredSummaries []openapiclientfleet.ResourceVersionSummary

	// If no filters, return all summaries
	if resourceID == "" && resourceKey == "" {
		return summaries, nil, nil
	}

	// Filter summaries
	for _, summary := range summaries {
		includeResource := false

		if resourceID != "" && summary.ResourceId != nil {
			if *summary.ResourceId == resourceID {
				includeResource = true
			}
		}

		if resourceKey != "" && summary.ResourceName != nil {
			if *summary.ResourceName == resourceKey {
				includeResource = true
			}
		}

		if includeResource {
			filteredSummaries = append(filteredSummaries, summary)
		}
	}

	// Create filter info
	filterInfo := map[string]interface{}{}
	if resourceID != "" {
		filterInfo["resourceId"] = resourceID
	}
	if resourceKey != "" {
		filterInfo["resourceKey"] = resourceKey
	}

	// Create count info
	countInfo := map[string]interface{}{
		"totalResourceVersionSummaries":    len(summaries),
		"filteredResourceVersionSummaries": len(filteredSummaries),
	}

	return filteredSummaries, filterInfo, countInfo
}
