package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func DebugHostCluster(ctx context.Context, token string, hostClusterID string) (*openapiclientfleet.DebugHostClusterResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	fmt.Printf("Debugging host cluster with ID: %s\n", hostClusterID)
	req := apiClient.HostclusterApiAPI.HostclusterApiDebugHostCluster(ctxWithToken, hostClusterID).
		IncludeAmenitiesInstallationLogs(true)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	debugRes, r, err := req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return debugRes, nil
}

func DescribeHostCluster(ctx context.Context, token string, hostClusterID string) (*openapiclientfleet.HostCluster, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiDescribeHostCluster(ctxWithToken, hostClusterID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	hostCluster, r, err := req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return hostCluster, nil
}

func ListHostClusters(ctx context.Context, token string, accountConfigID *string, regionID *string) (*openapiclientfleet.ListHostClustersResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiListHostClusters(ctxWithToken)

	if accountConfigID != nil {
		req = req.AccountConfigId(*accountConfigID)
	}
	if regionID != nil {
		req = req.RegionId(*regionID)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	hostClusters, r, err := req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return hostClusters, nil
}

func ApplyPendingChangesToHostCluster(ctx context.Context, token string, hostClusterID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiApplyPendingChangesToHostCluster(ctxWithToken, hostClusterID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err := req.Execute()
	if err != nil {
		return handleFleetError(err)
	}

	return nil
}

func UpdateHostCluster(ctx context.Context, token string, hostClusterID string, pendingAmenities []openapiclientfleet.Amenity, syncWithOrgTemplate *bool) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	if len(pendingAmenities) > 0 && utils.FromPtr(syncWithOrgTemplate) {
		return fmt.Errorf("cannot set pending amenities when syncing with organization template is enabled")
	}

	updateRequest := openapiclientfleet.UpdateHostClusterRequest2{}
	// Set sync with organization template flag if provided
	if syncWithOrgTemplate != nil {
		updateRequest.SyncWithOrgTemplate = syncWithOrgTemplate
	} else {
		// If sync with organization template is not provided, we do not set it
		updateRequest.PendingAmenities = pendingAmenities
	}
	req := apiClient.HostclusterApiAPI.HostclusterApiUpdateHostCluster(ctxWithToken, hostClusterID).UpdateHostClusterRequest2(updateRequest)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err := req.Execute()
	if err != nil {
		return handleFleetError(err)
	}

	return nil
}

func AdoptHostCluster(ctx context.Context, token string, hostClusterID string, cloudProvider string, region string, description string, userEmail *string) (*openapiclientfleet.AdoptHostClusterResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	adoptRequest := openapiclientfleet.AdoptHostClusterRequest2{
		CloudProvider: cloudProvider,
		Region:        region,
		Description:   description,
		Id:            hostClusterID,
	}

	if userEmail != nil && *userEmail != "" {
		adoptRequest.CustomerEmail = userEmail
	}

	req := apiClient.HostclusterApiAPI.HostclusterApiAdoptHostCluster(ctxWithToken).AdoptHostClusterRequest2(adoptRequest)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	hostCluster, r, err := req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return hostCluster, nil
}

func DeleteHostCluster(ctx context.Context, token string, hostClusterID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiDeleteHostCluster(ctxWithToken, hostClusterID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err := req.Execute()
	if err != nil {
		return handleFleetError(err)
	}

	return nil
}

func GetKubeConfigForHostCluster(
	ctx context.Context,
	token string,
	hostClusterID string,
	role string,
) (
	*openapiclientfleet.KubeConfigHostClusterResult,
	error,
) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.HostclusterApiAPI.HostclusterApiKubeConfigHostCluster(ctxWithToken, hostClusterID)

	if role != "" {
		req = req.Role(role)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	kubeConfig, r, err := req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	return kubeConfig, nil
}

// GetOrganizationDeploymentCellTemplate retrieves the organization template for a specific environment and cloud provider
func GetOrganizationDeploymentCellTemplate(ctx context.Context, token string, environment string, cloudProvider string) (*model.DeploymentCellTemplate, error) {
	// Get the service provider organization configuration
	spOrg, err := GetServiceProviderOrganization(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get service provider organization: %w", err)
	}

	// Extract deployment cell configurations for the environment
	deploymentCellConfigs, exists := spOrg.GetDeploymentCellConfigurationsPerEnv()[environment]
	if !exists {
		return nil, fmt.Errorf("no deployment cell configurations found for environment '%s'", environment)
	}

	// Convert to map to access the DeploymentCellConfigurationPerCloudProvider
	deploymentCellConfigsMap, err := interfaceToMap(deploymentCellConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert deployment cell configurations to map: %w", err)
	}

	// Access the DeploymentCellConfigurationPerCloudProvider level
	deploymentCellConfigPerCloudProvider, exists := deploymentCellConfigsMap["DeploymentCellConfigurationPerCloudProvider"]
	if !exists {
		return nil, fmt.Errorf("no DeploymentCellConfigurationPerCloudProvider found for environment '%s'", environment)
	}

	// Convert the cloud provider configurations to map
	cloudProviderConfigsMap, err := interfaceToMap(deploymentCellConfigPerCloudProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cloud provider configurations to map: %w", err)
	}

	// Access the specific cloud provider configuration
	amenitiesPerCloudProvider, exists := cloudProviderConfigsMap[cloudProvider]
	if !exists {
		return nil, fmt.Errorf("no deployment cell configurations found for cloud provider '%s'", cloudProvider)
	}

	amenitiesInternalModel, err := ConvertToInternalAmenitiesList(amenitiesPerCloudProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amenities list: %w", err)
	}

	var managedAmenities []model.Amenity
	var customAmenities []model.Amenity
	for _, amenity := range amenitiesInternalModel {
		externalModel := model.Amenity{
			Name:        amenity.Name,
			Description: amenity.Description,
			Type:        amenity.Type,
			Properties:  amenity.Properties,
		}
		if utils.FromPtr(amenity.IsManaged) {
			managedAmenities = append(managedAmenities, externalModel)
		} else {
			customAmenities = append(customAmenities, externalModel)
		}
	}

	return &model.DeploymentCellTemplate{
		ManagedAmenities: managedAmenities,
		CustomAmenities:  customAmenities,
	}, nil
}

func ConvertToInternalAmenitiesList(data interface{}) ([]model.InternalAmenity, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var configWrapper struct {
		Amenities []model.InternalAmenity `json:"Amenities"`
	}
	err = json.Unmarshal(jsonBytes, &configWrapper)
	if err == nil && len(configWrapper.Amenities) > 0 {
		return configWrapper.Amenities, nil
	}

	// If that fails, try to unmarshal directly as an array
	var result []model.InternalAmenity
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func interfaceToMap(data interface{}) (map[string]interface{}, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Unmarshal to map[string]interface{}
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListNodepools lists all nodepools in a deployment cell
func ListNodepools(ctx context.Context, token string, hostClusterID string) ([]model.NodepoolTableView, []openapiclientfleet.Entity, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Describe the host cluster first to get the cloud provider
	hostCluster, err := DescribeHostCluster(ctx, token, hostClusterID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe host cluster: %w", err)
	}

	var req openapiclientfleet.ApiHostclusterApiListHostClusterEntitiesRequest
	switch hostCluster.GetCloudProvider() {
	case "aws":
		req = apiClient.HostclusterApiAPI.HostclusterApiListHostClusterEntities(ctxWithToken, hostClusterID, "NODE_GROUP")
	case "gcp":
		req = apiClient.HostclusterApiAPI.HostclusterApiListHostClusterEntities(ctxWithToken, hostClusterID, "NODEPOOL")
	case "azure":
		req = apiClient.HostclusterApiAPI.HostclusterApiListHostClusterEntities(ctxWithToken, hostClusterID, "AZURE_NODEPOOL")
	default:
		return nil, nil, fmt.Errorf("nodepools are not supported for cloud provider: %s", hostCluster.GetCloudProvider())
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	result, r, err := req.Execute()
	if err != nil {
		return nil, nil, handleFleetError(err)
	}

	entities := result.GetEntities()

	var nodepools []model.NodepoolTableView
	for _, entity := range entities {
		nodepool := formatNodepoolForTable(entity, hostCluster.GetCloudProvider(), false)
		nodepools = append(nodepools, nodepool)
	}

	return nodepools, entities, nil
}

// formatNodepoolForTable formats a nodepool entity for table display
func formatNodepoolForTable(entity openapiclientfleet.Entity, cloudProvider string, isDescribe bool) model.NodepoolTableView {
	identifier := entity.GetIdentifier()

	// Extract just the nodepool name (last part after /)
	name := identifier
	if lastSlash := strings.LastIndex(identifier, "/"); lastSlash != -1 {
		name = identifier[lastSlash+1:]
	}

	tableView := model.NodepoolTableView{
		Name: name,
		Type: entity.GetType(),
	}

	properties := entity.GetProperties()

	switch cloudProvider {
	case "gcp":
		var nodePoolMap map[string]interface{}
		var ok bool

		if isDescribe {
			// Describe response: properties.applyRequest.node_pool.*
			if applyRequest, applyOk := properties["applyRequest"].(map[string]interface{}); applyOk {
				nodePoolMap, ok = applyRequest["node_pool"].(map[string]interface{})
			}
		} else {
			// List response: properties.node_pool.*
			nodePoolMap, ok = properties["node_pool"].(map[string]interface{})
		}

		if ok {
			if machineType, ok := nodePoolMap["machine_type"].(string); ok {
				tableView.MachineType = machineType
			}
			if imageType, ok := nodePoolMap["image_type"].(string); ok {
				tableView.ImageType = imageType
			}

			if autoscaling, ok := nodePoolMap["autoscaling"].(map[string]interface{}); ok {
				if maxNodes, ok := autoscaling["max_node_count"].(float64); ok {
					tableView.MaxNodes = int64(maxNodes)
				}
				if minNodes, ok := autoscaling["min_node_count"].(float64); ok {
					tableView.MinNodes = int64(minNodes)
				}
			}
			if locations, ok := nodePoolMap["node_locations"].([]interface{}); ok && len(locations) > 0 {
				if loc, ok := locations[0].(string); ok {
					tableView.Location = loc
				}
			}
			if nodeManagement, ok := nodePoolMap["node_management"].(map[string]interface{}); ok {
				if autoRepair, ok := nodeManagement["auto_repair"].(bool); ok {
					tableView.AutoRepair = autoRepair
				}
				if autoUpgrade, ok := nodeManagement["auto_upgrade"].(bool); ok {
					tableView.AutoUpgrade = autoUpgrade
				}
			}
			// Extract privateSubnet from labels without storing all labels
			if labels, ok := nodePoolMap["labels"].(map[string]interface{}); ok {
				if privateSubnetStr, ok := labels["omnistrate.com/private-subnet"].(string); ok {
					tableView.PrivateSubnet = (privateSubnetStr == "true")
				}
			}
		}

	case "aws":
		var nodegroupSpec map[string]interface{}
		var ok bool

		if isDescribe {
			// Describe response: properties.applyRequest.nodegroup_spec.*
			if applyRequest, applyOk := properties["applyRequest"].(map[string]interface{}); applyOk {
				nodegroupSpec, ok = applyRequest["nodegroup_spec"].(map[string]interface{})
			}
		} else {
			// List response: properties.nodegroup_spec.*
			nodegroupSpec, ok = properties["nodegroup_spec"].(map[string]interface{})
		}

		if ok {
			if amiType, ok := nodegroupSpec["ami_type"].(string); ok {
				tableView.ImageType = amiType
			}
			if capacityType, ok := nodegroupSpec["capacity_type"].(string); ok {
				tableView.CapacityType = capacityType
			}

			if scalingConfig, ok := nodegroupSpec["scaling_config"].(map[string]interface{}); ok {
				if minSize, ok := scalingConfig["min_size"].(float64); ok {
					tableView.MinNodes = int64(minSize)
				}
				if maxSize, ok := scalingConfig["max_size"].(float64); ok {
					tableView.MaxNodes = int64(maxSize)
				}
			}

			if subnets, ok := nodegroupSpec["subnets"].([]interface{}); ok && len(subnets) > 0 {
				if subnet, ok := subnets[0].(string); ok {
					tableView.Location = subnet
				}
			}

			// Extract privateSubnet from labels without storing all labels
			if labels, ok := nodegroupSpec["labels"].(map[string]interface{}); ok {
				if privateSubnetStr, ok := labels["omnistrate.com/private-subnet"].(string); ok {
					tableView.PrivateSubnet = (privateSubnetStr == "true")
				}
			}

			// Try to extract machine type from launch template name
			if launchTemplate, ok := nodegroupSpec["launch_template"].(map[string]interface{}); ok {
				if templateName, ok := launchTemplate["name"].(string); ok {
					// Format: hc-xxx-...-<instance-type>
					// Convert dots to dashes in template name: t3.medium -> t3-medium
					parts := strings.Split(templateName, "-")
					if len(parts) >= 2 {
						// Last part is usually the instance type
						instanceType := parts[len(parts)-1]
						// Find the previous part that's part of instance type
						if len(parts) >= 3 {
							prevPart := parts[len(parts)-2]
							// Check if it's an instance family (like t3, r7i, etc.)
							if len(prevPart) <= 4 && !strings.HasPrefix(prevPart, "hc") && !strings.HasPrefix(prevPart, "pt") && !strings.HasPrefix(prevPart, "r-") {
								tableView.MachineType = prevPart + "." + instanceType
							}
						}
					}
				}
			}
		}

	case "azure":
		var nodePoolSpec map[string]interface{}
		var ok bool

		if isDescribe {
			// Describe response: properties.applyRequest.node_pool_spec.*
			if applyRequest, applyOk := properties["applyRequest"].(map[string]interface{}); applyOk {
				nodePoolSpec, ok = applyRequest["node_pool_spec"].(map[string]interface{})
			}
		} else {
			// List response: properties.node_pool_spec.*
			nodePoolSpec, ok = properties["node_pool_spec"].(map[string]interface{})
		}

		if ok {
			if vmSize, ok := nodePoolSpec["vm_size"].(string); ok {
				tableView.MachineType = vmSize
			}

			if minCount, ok := nodePoolSpec["min_count"].(float64); ok {
				tableView.MinNodes = int64(minCount)
			}
			if maxCount, ok := nodePoolSpec["max_count"].(float64); ok {
				tableView.MaxNodes = int64(maxCount)
			}

			if zones, ok := nodePoolSpec["availability_zones"].([]interface{}); ok && len(zones) > 0 {
				if zone, ok := zones[0].(string); ok {
					tableView.Location = zone
				}
			}

			// 			if enableAutoScaling, ok := nodePoolSpec["enable_auto_scaling"].(bool); ok {
			// 				tableView.AutoScaling = enableAutoScaling
			// 			}

			if enableNodePublicIP, ok := nodePoolSpec["enable_node_public_ip"].(bool); ok {
				tableView.PrivateSubnet = !enableNodePublicIP
			}
		}
	}

	return tableView
}

// DescribeNodepool describes a specific nodepool in a deployment cell
func DescribeNodepool(ctx context.Context, token string, hostClusterID string, nodepoolName string) (*model.NodepoolDescribeView, *openapiclientfleet.Entity, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Get the host cluster to determine cloud provider
	hostCluster, err := DescribeHostCluster(ctx, token, hostClusterID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe host cluster: %w", err)
	}

	cloudProvider := hostCluster.GetCloudProvider()

	var req openapiclientfleet.ApiHostclusterApiDescribeHostClusterEntityRequest
	switch cloudProvider {
	case "aws":
		req = apiClient.HostclusterApiAPI.HostclusterApiDescribeHostClusterEntity(ctxWithToken, hostClusterID, "NODE_GROUP", nodepoolName)
	case "gcp":
		req = apiClient.HostclusterApiAPI.HostclusterApiDescribeHostClusterEntity(ctxWithToken, hostClusterID, "NODEPOOL", nodepoolName)
	case "azure":
		req = apiClient.HostclusterApiAPI.HostclusterApiDescribeHostClusterEntity(ctxWithToken, hostClusterID, "AZURE_NODEPOOL", nodepoolName)
	default:
		return nil, nil, fmt.Errorf("nodepools are not supported for cloud provider: %s", cloudProvider)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	entity, r, err := req.Execute()
	if err != nil {
		fleetErr := handleFleetError(err)
		// Simplify not found errors
		errMsg := fleetErr.Error()
		if strings.Contains(errMsg, "not_found") || strings.Contains(errMsg, "Not found") || strings.Contains(errMsg, "notFound") || strings.Contains(errMsg, "404") {
			return nil, nil, fmt.Errorf("nodepool '%s' not found in deployment cell '%s'", nodepoolName, hostClusterID)
		}
		return nil, nil, fleetErr
	}

	tableView := formatNodepoolForTable(*entity, cloudProvider, true)

	// Get currentNodeCount for describe response
	var currentNodes int64
	properties := entity.GetProperties()

	if currentNodeCount, ok := properties["currentNodeCount"].(float64); ok {
		currentNodes = int64(currentNodeCount)
	}

	describeView := model.NodepoolDescribeView{
		Name:          tableView.Name,
		Type:          tableView.Type,
		MachineType:   tableView.MachineType,
		ImageType:     tableView.ImageType,
		MinNodes:      tableView.MinNodes,
		MaxNodes:      tableView.MaxNodes,
		CurrentNodes:  currentNodes,
		Location:      tableView.Location,
		AutoRepair:    tableView.AutoRepair,
		AutoUpgrade:   tableView.AutoUpgrade,
		CapacityType:  tableView.CapacityType,
		PrivateSubnet: tableView.PrivateSubnet,
	}
	return &describeView, entity, nil
}

// ConfigureNodepool configures a nodepool in a deployment cell
func ConfigureNodepool(ctx context.Context, token string, hostClusterID string, nodepoolName string, maxSize int64) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	requestBody := openapiclientfleet.NewSetNodePoolPropertyRequest2(maxSize)

	req := apiClient.HostclusterApiAPI.HostclusterApiSetNodePoolProperty(ctxWithToken, hostClusterID, nodepoolName).
		SetNodePoolPropertyRequest2(*requestBody)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err := req.Execute()
	if err != nil {
		return handleFleetError(err)
	}

	return nil
}

// DeleteNodepool deletes a nodepool from a deployment cell (can take up to 10 minutes)
func DeleteNodepool(ctx context.Context, token string, hostClusterID string, nodepoolName string) error {
	// Create a context with a 15 minute timeout for delete operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	ctxWithToken := context.WithValue(ctxWithTimeout, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Describe the host cluster first to get the cloud provider
	hostCluster, err := DescribeHostCluster(ctx, token, hostClusterID)
	if err != nil {
		return fmt.Errorf("failed to describe host cluster: %w", err)
	}

	var req openapiclientfleet.ApiHostclusterApiDeleteEntityRequest
	switch hostCluster.GetCloudProvider() {
	case "aws":
		req = apiClient.HostclusterApiAPI.HostclusterApiDeleteEntity(ctxWithToken, hostClusterID, "NODE_GROUP", nodepoolName)
	case "gcp":
		req = apiClient.HostclusterApiAPI.HostclusterApiDeleteEntity(ctxWithToken, hostClusterID, "NODEPOOL", nodepoolName)
	case "azure":
		req = apiClient.HostclusterApiAPI.HostclusterApiDeleteEntity(ctxWithToken, hostClusterID, "AZURE_NODEPOOL", nodepoolName)
	default:
		return fmt.Errorf("nodepools are not supported for cloud provider: %s", hostCluster.GetCloudProvider())
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		fleetErr := handleFleetError(err)
		// Simplify not found errors
		errMsg := fleetErr.Error()
		if strings.Contains(errMsg, "not_found") || strings.Contains(errMsg, "Not found") || strings.Contains(errMsg, "notFound") || strings.Contains(errMsg, "404") {
			return fmt.Errorf("nodepool '%s' not found in deployment cell '%s'", nodepoolName, hostClusterID)
		}
		return fleetErr
	}

	return nil
}
