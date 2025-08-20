package dataaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"net/http"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func DebugHostCluster(ctx context.Context, token string, hostClusterID string) (*model.DebugHostClusterResult, error) {
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

	// Try to execute the request - this will fail with the JSON unmarshaling error
	_, r, err := req.Execute()

	// Super hacky solution to handle os.File serialization error https://github.com/omnistrate-oss/omnistrate-sdk-go/blob/6f12a05781fb478479c0cf774edad39bd2a1a9ac/fleet/docs/DebugHostClusterResult.md?plain=1#L7
	// Check if this is the specific JSON unmarshaling error we expect
	if err != nil && strings.Contains(err.Error(), "cannot unmarshal string into Go struct field") {
		// This is the expected error - the actual JSON data might be in the error details
		// Let's check if this is a GenericOpenAPIError that contains the response body
		var serviceErr *openapiclientfleet.GenericOpenAPIError
		if errors.As(err, &serviceErr) {
			// Try to get the response body from the error
			responseBody := serviceErr.Body()
			if len(responseBody) > 0 {
				// Parse the JSON to extract customHelmExecutionLogs
				var rawResponse map[string]interface{}
				if parseErr := json.Unmarshal(responseBody, &rawResponse); parseErr != nil {
					return nil, fmt.Errorf("failed to parse response JSON: %w", parseErr)
				}

				// Create our custom result struct
				debugRes := &model.DebugHostClusterResult{}

				// Handle customHelmExecutionLogs specially
				if customLogs, exists := rawResponse["customHelmExecutionLogs"]; exists && customLogs != nil {
					if logsMap, ok := customLogs.(map[string]interface{}); ok {
						// Convert to map[string]string for easier handling
						stringLogsMap := make(map[string]string)
						for serviceName, logContent := range logsMap {
							if logStr, ok := logContent.(string); ok {
								stringLogsMap[serviceName] = logStr
							}
						}
						debugRes.CustomHelmExecutionLogs = stringLogsMap
					}
				}

				return debugRes, nil
			}
		}

		// If we can't extract from GenericOpenAPIError, try parsing the error message directly
		errorStr := err.Error()

		// Look for JSON pattern in the error message - it might start with {
		jsonStartIndex := strings.Index(errorStr, `{"customHelmExecutionLogs":`)
		if jsonStartIndex != -1 {
			// Extract the JSON part from the error message
			jsonStr := errorStr[jsonStartIndex:]

			// Parse the JSON to extract customHelmExecutionLogs
			var rawResponse map[string]interface{}
			if parseErr := json.Unmarshal([]byte(jsonStr), &rawResponse); parseErr != nil {
				return nil, fmt.Errorf("failed to parse error message JSON: %w", parseErr)
			}

			// Create our custom result struct
			debugRes := &model.DebugHostClusterResult{}

			// Handle customHelmExecutionLogs specially
			if customLogs, exists := rawResponse["customHelmExecutionLogs"]; exists && customLogs != nil {
				if logsMap, ok := customLogs.(map[string]interface{}); ok {
					// Convert to map[string]string for easier handling
					stringLogsMap := make(map[string]string)
					for serviceName, logContent := range logsMap {
						if logStr, ok := logContent.(string); ok {
							stringLogsMap[serviceName] = logStr
						}
					}
					debugRes.CustomHelmExecutionLogs = stringLogsMap
				}
			}

			return debugRes, nil
		}

		return nil, fmt.Errorf("unexpected error format: %w", err)
	}

	// If it's a different error, return it
	if err != nil {
		return nil, handleFleetError(err)
	}

	// If somehow the request succeeded (shouldn't happen with current API), return empty result
	return &model.DebugHostClusterResult{}, nil
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
