package instance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe an instance deployment
omnistrate-ctl instance describe instance-abcd1234

# Get compact deployment status information
omnistrate-ctl instance describe instance-abcd1234 --deployment-status

# Get deployment status for specific resource only  
omnistrate-ctl instance describe instance-abcd1234 --deployment-status --resource-key mydb`
)

type InstanceStatusType string

var InstanceStatus InstanceStatusType

const (
	InstanceStatusRunning   InstanceStatusType = "RUNNING"
	InstanceStatusStopped   InstanceStatusType = "STOPPED"
	InstanceStatusFailed    InstanceStatusType = "FAILED"
	InstanceStatusCancelled InstanceStatusType = "CANCELLED"
	InstanceStatusUnknown   InstanceStatusType = "UNKNOWN"
)

// InstanceDeploymentStatus represents a compact view of instance deployment status
type InstanceDeploymentStatus struct {
	InstanceID               string                     `json:"instanceId"`
	ServiceID                string                     `json:"serviceId"`
	EnvironmentID            string                     `json:"environmentId"`
	Status                   string                     `json:"status"`
	ProductTierID            string                     `json:"productTierId"`
	TierVersion              string                     `json:"tierVersion"`
	CreationTime             string                     `json:"creationTime,omitempty"`
	LastModifiedTime         string                     `json:"lastModifiedTime,omitempty"`
	ResourceDeploymentStatus []ResourceDeploymentStatus `json:"resourceDeploymentStatus"`
	AppliedFilters           map[string]interface{}     `json:"appliedFilters,omitempty"`
	FilteringStats           map[string]interface{}     `json:"filteringStats,omitempty"`
}

// ResourceDeploymentStatus represents compact deployment status for a single resource
type ResourceDeploymentStatus struct {
	ResourceID       string                 `json:"resourceId,omitempty"`
	ResourceName     string                 `json:"resourceName,omitempty"`
	Version          string                 `json:"version,omitempty"`
	LatestVersion    string                 `json:"latestVersion,omitempty"`
	PodStatus        map[string]string      `json:"podStatus,omitempty"`
	DeploymentErrors string                 `json:"deploymentErrors,omitempty"`
	DeploymentType   string                 `json:"deploymentType,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

var describeCmd = &cobra.Command{
	Use:          "describe [instance-id]",
	Short:        "Describe an instance deployment for your service",
	Long:         `This command helps you describe the instance for your service.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")
	describeCmd.Flags().String("resource-id", "", "Filter results by resource ID")
	describeCmd.Flags().String("resource-key", "", "Filter results by resource key")
	describeCmd.Flags().Bool("deployment-status", false, "Return compact deployment status information instead of full instance details")
	describeCmd.Flags().Bool("detail", false, "Include detailed information in the response")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	resourceID, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	resourceKey, err := cmd.Flags().GetString("resource-key")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	deploymentStatus, err := cmd.Flags().GetBool("deployment-status")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	detail, err := cmd.Flags().GetBool("detail")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate output flag
	if output != "json" {
		err = errors.New("only json output is supported")
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Describing instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe instance
	var instance *openapiclientfleet.ResourceInstance
	instance, err = dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, detail)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully described instance")
	if instance.ConsumptionResourceInstanceResult.Status != nil {
		InstanceStatus = InstanceStatusType(*instance.ConsumptionResourceInstanceResult.Status)
	} else {
		InstanceStatus = InstanceStatusUnknown
	}

	// If deployment-status flag is set, return compact deployment status
	if deploymentStatus {
		status, err := createInstanceDeploymentStatus(cmd.Context(), token, instance, serviceID, environmentID, instanceID, resourceID, resourceKey)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to create deployment status: %w", err))
			return err
		}

		// Print compact status
		err = utils.PrintTextTableJsonOutput(output, status)
		if err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	// Apply resource filtering if specified (for full instance response)
	if resourceID != "" || resourceKey != "" {
		filteredInstance, err := filterInstanceByResource(cmd.Context(), token, instance, serviceID, resourceID, resourceKey)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to apply resource filter: %w", err))
			return err
		}
		instance = filteredInstance
	}

	// Replace the kubectl config instructions with the omnistrate-ctl update-kubeconfig for better MCP reference
	if instance.DeploymentCellID != nil {
		instance.InstanceDebugCommands = []string{
			fmt.Sprintf("omnistrate-ctl deployment-cell update-kubeconfig %s --role cluster-admin --kubeconfig /tmp/kubeconfig", *instance.DeploymentCellID),
			fmt.Sprintf("KUBECONFIG=/tmp/kubeconfig kubectl get pods -n instance-%s", *instance.ConsumptionResourceInstanceResult.Id),
		}
	}

	// Print full instance output
	err = utils.PrintTextTableJsonOutput(output, instance)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// Helper functions

func getInstance(ctx context.Context, token, instanceID string) (serviceID, environmentID, productTierID, resourceID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return
	}

	var found bool
	for _, instance := range searchRes.ResourceInstanceResults {
		if instance.Id == instanceID {
			serviceID = instance.ServiceId
			environmentID = instance.ServiceEnvironmentId
			productTierID = instance.ProductTierId
			if instance.ResourceId != nil {
				resourceID = *instance.ResourceId
			}
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("%s not found. Please check the instance ID and try again", instanceID)
		return
	}

	return
}

func getResourceFromInstance(ctx context.Context, token string, instanceID string, resourceName string) (resourceID, resourceType string, err error) {
	// Check if instance exists
	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return
	}

	// Retrieve resource ID
	instanceDes, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID, true)
	if err != nil {
		return
	}

	versionSetDes, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, instanceDes.ProductTierId, instanceDes.TierVersion)
	if err != nil {
		return
	}

	for _, resource := range versionSetDes.Resources {
		if resource.Name == resourceName {
			resourceID = resource.Id
			if resource.ManagedResourceType != nil {
				resourceType = strings.ToLower(*resource.ManagedResourceType)
			}
		}
	}

	return
}

func filterInstanceByResource(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance, serviceID, resourceID, resourceKey string) (*openapiclientfleet.ResourceInstance, error) {
	// The ResourceInstance has complex nested structure with resource-specific data
	// For filtering, we'll focus on the most relevant parts that contain resource information

	// Make a copy of the instance to avoid modifying the original
	filteredInstance := *instance

	// Get version set to map resource names to IDs if needed
	var keyToID, idToKey map[string]string
	if resourceID != "" || resourceKey != "" {
		versionSetDes, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, instance.ProductTierId, instance.TierVersion)
		if err != nil {
			return instance, err // Return original if we can't get version set
		}

		// Create mapping from resource key to resource ID and vice versa
		keyToID = make(map[string]string)
		idToKey = make(map[string]string)
		for _, resource := range versionSetDes.Resources {
			keyToID[resource.Name] = resource.Id
			idToKey[resource.Id] = resource.Name
		}
	}

	// Filter ResourceVersionSummaries array
	if len(instance.ResourceVersionSummaries) > 0 {
		var filteredSummaries []openapiclientfleet.ResourceVersionSummary

		for _, summary := range instance.ResourceVersionSummaries {
			includeResource := false

			if resourceID != "" && summary.ResourceId != nil {
				// If filtering by resource ID, check if it matches
				if *summary.ResourceId == resourceID {
					includeResource = true
				}
			}

			if resourceKey != "" && summary.ResourceName != nil {
				// If filtering by resource key, check if it matches resource name
				if *summary.ResourceName == resourceKey {
					includeResource = true
				}
			}

			// Also check cross-mapping: if filtering by resource ID but summary has resource name, check mapping
			if resourceID != "" && summary.ResourceName != nil && !includeResource {
				if keyToID[*summary.ResourceName] == resourceID {
					includeResource = true
				}
			}

			// Also check cross-mapping: if filtering by resource key but summary has resource ID, check mapping
			if resourceKey != "" && summary.ResourceId != nil && !includeResource {
				if idToKey[*summary.ResourceId] == resourceKey {
					includeResource = true
				}
			}

			if includeResource {
				filteredSummaries = append(filteredSummaries, summary)
			}
		}

		filteredInstance.ResourceVersionSummaries = filteredSummaries
	}

	// Filter DetailedNetworkTopology if present - this contains resource-specific network information
	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology != nil {
		filteredTopology := make(map[string]openapiclientfleet.ResourceNetworkTopologyResult)

		// Filter the topology based on resourceID or resourceKey
		for key, value := range *instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology {
			includeResource := false

			if resourceID != "" {
				// If filtering by resource ID, check if key matches resource ID or if key is resource name that maps to resource ID
				if key == resourceID || keyToID[key] == resourceID {
					includeResource = true
				}
			}

			if resourceKey != "" {
				// If filtering by resource key, check if key matches resource key or if key is resource ID that maps to resource key
				if key == resourceKey || idToKey[key] == resourceKey {
					includeResource = true
				}
			}

			if includeResource {
				filteredTopology[key] = value
			}
		}

		filteredInstance.ConsumptionResourceInstanceResult.DetailedNetworkTopology = utils.ToPtr(filteredTopology)
	}

	// Add filtered resource information as additional metadata
	if filteredInstance.AdditionalProperties == nil {
		filteredInstance.AdditionalProperties = make(map[string]interface{})
	}

	// Add filter information to the response
	filterInfo := map[string]interface{}{}
	if resourceID != "" {
		filterInfo["resourceId"] = resourceID
	}
	if resourceKey != "" {
		filterInfo["resourceKey"] = resourceKey
	}
	filteredInstance.AdditionalProperties["appliedFilters"] = filterInfo

	// Add count information for filtered resources
	countInfo := map[string]interface{}{
		"totalResourceVersionSummaries":    len(instance.ResourceVersionSummaries),
		"filteredResourceVersionSummaries": len(filteredInstance.ResourceVersionSummaries),
	}

	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology != nil {
		countInfo["totalNetworkTopologyEntries"] = len(*instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology)
	}

	if filteredInstance.ConsumptionResourceInstanceResult.DetailedNetworkTopology != nil {
		countInfo["filteredNetworkTopologyEntries"] = len(*filteredInstance.ConsumptionResourceInstanceResult.DetailedNetworkTopology)
	}

	filteredInstance.AdditionalProperties["filteringStats"] = countInfo

	return &filteredInstance, nil
}

func createInstanceDeploymentStatus(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance, serviceID, environmentID, instanceID, resourceID, resourceKey string) (*InstanceDeploymentStatus, error) {
	status := &InstanceDeploymentStatus{
		InstanceID:    instanceID,
		ServiceID:     serviceID,
		EnvironmentID: environmentID,
		ProductTierID: instance.ProductTierId,
		TierVersion:   instance.TierVersion,
	}

	// Set instance status
	if instance.ConsumptionResourceInstanceResult.Status != nil {
		status.Status = *instance.ConsumptionResourceInstanceResult.Status
	} else {
		status.Status = "UNKNOWN"
	}

	// Set timestamps if available
	if instance.ConsumptionResourceInstanceResult.CreatedAt != nil {
		status.CreationTime = *instance.ConsumptionResourceInstanceResult.CreatedAt
	}
	if instance.ConsumptionResourceInstanceResult.LastModifiedAt != nil {
		status.LastModifiedTime = *instance.ConsumptionResourceInstanceResult.LastModifiedAt
	}

	// Extract compact deployment status from resource version summaries
	var deploymentStatuses []ResourceDeploymentStatus
	var filteredSummaries []openapiclientfleet.ResourceVersionSummary
	var err error

	// Apply resource filtering if specified
	if resourceID != "" || resourceKey != "" {
		var filterInfo, countInfo map[string]interface{}
		filteredSummaries, filterInfo, countInfo, err = filterResourceVersionSummariesForStatus(ctx, token, instance.ResourceVersionSummaries, serviceID, instance.ProductTierId, instance.TierVersion, resourceID, resourceKey)
		if err != nil {
			return nil, fmt.Errorf("failed to apply resource filter: %w", err)
		}
		status.AppliedFilters = filterInfo
		status.FilteringStats = countInfo
	} else {
		filteredSummaries = instance.ResourceVersionSummaries
	}

	// Create compact deployment status for each resource
	for _, summary := range filteredSummaries {
		deploymentStatus := createResourceDeploymentStatus(summary)
		deploymentStatuses = append(deploymentStatuses, deploymentStatus)
	}

	status.ResourceDeploymentStatus = deploymentStatuses

	return status, nil
}

func filterResourceVersionSummariesForStatus(ctx context.Context, token string, summaries []openapiclientfleet.ResourceVersionSummary, serviceID, productTierId, tierVersion, resourceID, resourceKey string) ([]openapiclientfleet.ResourceVersionSummary, map[string]interface{}, map[string]interface{}, error) {
	var filteredSummaries []openapiclientfleet.ResourceVersionSummary

	// Get version set to map resource names to IDs if needed
	var keyToID, idToKey map[string]string
	if resourceID != "" || resourceKey != "" {
		versionSetDes, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, productTierId, tierVersion)
		if err != nil {
			return summaries, nil, nil, err // Return original if we can't get version set
		}

		// Create mapping from resource key to resource ID and vice versa
		keyToID = make(map[string]string)
		idToKey = make(map[string]string)
		for _, resource := range versionSetDes.Resources {
			keyToID[resource.Name] = resource.Id
			idToKey[resource.Id] = resource.Name
		}
	}

	// Filter summaries
	for _, summary := range summaries {
		includeResource := false

		if resourceID != "" && summary.ResourceId != nil {
			// If filtering by resource ID, check if it matches
			if *summary.ResourceId == resourceID {
				includeResource = true
			}
		}

		if resourceKey != "" && summary.ResourceName != nil {
			// If filtering by resource key, check if it matches resource name
			if *summary.ResourceName == resourceKey {
				includeResource = true
			}
		}

		// Also check cross-mapping: if filtering by resource ID but summary has resource name, check mapping
		if resourceID != "" && summary.ResourceName != nil && !includeResource {
			if keyToID[*summary.ResourceName] == resourceID {
				includeResource = true
			}
		}

		// Also check cross-mapping: if filtering by resource key but summary has resource ID, check mapping
		if resourceKey != "" && summary.ResourceId != nil && !includeResource {
			if idToKey[*summary.ResourceId] == resourceKey {
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

	return filteredSummaries, filterInfo, countInfo, nil
}

func createResourceDeploymentStatus(summary openapiclientfleet.ResourceVersionSummary) ResourceDeploymentStatus {
	status := ResourceDeploymentStatus{}

	// Basic resource information
	if summary.ResourceId != nil {
		status.ResourceID = *summary.ResourceId
	}
	if summary.ResourceName != nil {
		status.ResourceName = *summary.ResourceName
	}
	if summary.Version != nil {
		status.Version = *summary.Version
	}
	if summary.LatestVersion != nil {
		status.LatestVersion = *summary.LatestVersion
	}

	// Extract deployment-specific information based on configuration type
	if summary.GenericResourceDeploymentConfiguration != nil {
		status.DeploymentType = "Generic"
		generic := summary.GenericResourceDeploymentConfiguration

		// Extract pod status
		if generic.PodStatus != nil {
			status.PodStatus = *generic.PodStatus
		}

		// Add additional info for generic deployment
		additionalInfo := make(map[string]interface{})
		if generic.Image != nil {
			additionalInfo["image"] = *generic.Image
		}
		if generic.PodToHostMapping != nil {
			additionalInfo["podToHostMapping"] = *generic.PodToHostMapping
		}
		if len(additionalInfo) > 0 {
			status.AdditionalInfo = additionalInfo
		}
	}

	if summary.HelmDeploymentConfiguration != nil {
		status.DeploymentType = "Helm"
		helm := summary.HelmDeploymentConfiguration

		// Extract deployment errors
		if helm.DeploymentErrors != nil {
			status.DeploymentErrors = *helm.DeploymentErrors
		}

		// Extract pod status if available
		if helm.PodStatus != nil {
			status.PodStatus = *helm.PodStatus
		}

		// Add helm-specific info
		additionalInfo := make(map[string]interface{})
		additionalInfo["chartName"] = helm.ChartName
		additionalInfo["chartVersion"] = helm.ChartVersion
		additionalInfo["releaseName"] = helm.ReleaseName
		additionalInfo["releaseNamespace"] = helm.ReleaseNamespace
		additionalInfo["releaseStatus"] = helm.ReleaseStatus
		additionalInfo["repositoryURL"] = helm.RepositoryURL
		if helm.PodToHostMapping != nil {
			additionalInfo["podToHostMapping"] = *helm.PodToHostMapping
		}
		status.AdditionalInfo = additionalInfo
	}

	if summary.KustomizeDeploymentConfiguration != nil {
		status.DeploymentType = "Kustomize"
		kustomize := summary.KustomizeDeploymentConfiguration

		// Extract deployment errors
		if kustomize.DeploymentErrors != nil {
			status.DeploymentErrors = *kustomize.DeploymentErrors
		}

		// Add kustomize-specific info
		additionalInfo := make(map[string]interface{})
		additionalInfo["basePath"] = kustomize.BasePath
		additionalInfo["overlays"] = kustomize.Overlays
		status.AdditionalInfo = additionalInfo
	}

	if summary.TerraformDeploymentConfiguration != nil {
		status.DeploymentType = "Terraform"
		terraform := summary.TerraformDeploymentConfiguration

		// Extract deployment errors
		if terraform.DeploymentErrors != nil {
			status.DeploymentErrors = *terraform.DeploymentErrors
		}

		// Add terraform-specific info
		additionalInfo := make(map[string]interface{})
		if terraform.ConfigurationFiles != nil {
			additionalInfo["configurationFiles"] = *terraform.ConfigurationFiles
		}
		if len(additionalInfo) > 0 {
			status.AdditionalInfo = additionalInfo
		}
	}

	return status
}
