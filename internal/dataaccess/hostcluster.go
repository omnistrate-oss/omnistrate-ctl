package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

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
	var req openapiclientfleet.ApiHostclusterApiUpdateHostClusterRequest
	// Set sync with organization template flag if provided
	if syncWithOrgTemplate != nil {
		updateRequest.SyncWithOrgTemplate = syncWithOrgTemplate
		req = apiClient.HostclusterApiAPI.HostclusterApiUpdateHostCluster(ctxWithToken, hostClusterID).UpdateHostClusterRequest2(updateRequest)
	} else {
		// If sync with organization template is not provided, we do not set it
		updateRequest.PendingAmenities = pendingAmenities
		req = apiClient.HostclusterApiAPI.HostclusterApiUpdateHostCluster(ctxWithToken, hostClusterID).UpdateHostClusterRequest2(updateRequest)
	}

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
