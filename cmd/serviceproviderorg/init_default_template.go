package serviceproviderorg

import (
	"context"
	"fmt"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initDefaultTemplateCmd = &cobra.Command{
	Use:   "init-deployment-cell-config-template",
	Short: "Initialize deployment cell configuration template for service provider organization",
	Long: `Initialize service provider organization-level deployment cell configuration template.

This command initializes the default organization-level configuration template for deployment cells. 
This step is purely at the service provider org level; no reference to any specific service is needed.

The configuration will be stored as a template that can be applied to different 
environments (production, staging, development) and used to synchronize deployment cells.

Organization ID is automatically determined from your credentials.

Examples:
  # Initialize deployment cell configuration template with default settings
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template

  # Save template configuration to a local file
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template --output-file template.yaml`,
	RunE:         runInitDefaultTemplate,
	SilenceUsage: true,
}

func init() {
	initDefaultTemplateCmd.Flags().StringP("output-file", "", "", "Path to output the template configuration to a local YAML file (optional)")
}

func runInitDefaultTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	outputFile, err := cmd.Flags().GetString("output-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get default configuration from service provider organization using proper Go models
	templateConfig, err := getDefaultTemplateConfiguration(ctx, token)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get default template configuration: %w", err))
		return err
	}

	// Save template to file before sending to server if requested
	if outputFile != "" {
		err = writeTemplateToYAMLFile(outputFile, templateConfig)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to write template to file: %w", err))
			return err
		}
		utils.PrintSuccess(fmt.Sprintf("Template configuration saved to %s", outputFile))
	}

	return nil
}

func getDefaultTemplateConfiguration(ctx context.Context, token string) (model.DeploymentCellConfigurationTemplate, error) {
	// Retrieve the default template from service provider organization's DefaultDeploymentCellConfigurations
	res, err := dataaccess.GetServiceProviderOrganization(ctx, token)
	println(fmt.Sprintf("Retrieved service provider organization: %v", res))
	if err != nil {
		return model.DeploymentCellConfigurationTemplate{}, fmt.Errorf("failed to get service provider organization: %w", err)
	}

	// Convert the API response to model.DeploymentCellConfigurationTemplate
	template := model.DeploymentCellConfigurationTemplate{
		DeploymentCellConfigurations: make(map[string]model.DeploymentCellConfiguration),
	}

	if res.DefaultDeploymentCellConfigurations == nil {
		return template, fmt.Errorf("no default deployment cell configurations found in service provider organization")
	}

	for cloudProvider, cellConfig := range res.DefaultDeploymentCellConfigurations.DeploymentCellConfigurationPerCloudProvider {
		convertedConfig := convertToDeploymentCellConfiguration(cellConfig)
		template.DeploymentCellConfigurations[cloudProvider] = convertedConfig
	}
	return template, nil
}

func writeTemplateToYAMLFile(filePath string, config model.DeploymentCellConfigurationTemplate) error {
	// Marshal the model to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal template to YAML: %w", err)
	}

	// Write the YAML data to file using Go's standard library
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}

func convertTemplateToMap(templateConfig model.DeploymentCellConfigurationTemplate) (map[string]interface{}, error) {
	// Convert the template configuration (of type model.DeploymentCellConfigurationTemplate) to map[string]interface{}
	templateMap := make(map[string]interface{})

	// Convert DeploymentCellConfigurations
	if templateConfig.DeploymentCellConfigurations != nil {
		cloudProviderConfigs := make(map[string]interface{})

		for cloudProvider, config := range templateConfig.DeploymentCellConfigurations {
			if config.Amenities != nil {
				amenities := make(map[string]interface{})

				// Convert each amenity
				for _, amenity := range config.Amenities {
					amenityConfig := map[string]interface{}{
						"modifiable":  amenity.Modifiable,
						"description": amenity.Description,
						"isManaged":   amenity.IsManaged,
						"type":        amenity.Type,
					}

					// Add properties if they exist
					if amenity.Properties != nil {
						amenityConfig["properties"] = amenity.Properties
					}

					amenities[amenity.Name] = amenityConfig
				}

				cloudProviderConfigs[string(cloudProvider)] = map[string]interface{}{
					"amenities": amenities,
				}
			}
		}

		templateMap["deploymentCellConfigurations"] = cloudProviderConfigs
	}

	return templateMap, nil
}

func convertToDeploymentCellConfiguration(cellConfig interface{}) model.DeploymentCellConfiguration {
	// Initialize the result with empty amenities slice
	result := model.DeploymentCellConfiguration{
		Amenities: make([]model.Amenity, 0),
	}

	// Handle the case where cellConfig is a map[string]interface{} from the API response
	if cellConfig == nil {
		return result
	}

	// Try to convert interface{} to map[string]interface{} first (raw API response)
	if configMap, ok := cellConfig.(map[string]interface{}); ok {
		if amenitiesInterface, exists := configMap["Amenities"]; exists {
			if amenitiesList, ok := amenitiesInterface.([]interface{}); ok {
				amenities := make([]model.Amenity, 0, len(amenitiesList))

				for _, amenityInterface := range amenitiesList {
					if amenityMap, ok := amenityInterface.(map[string]interface{}); ok {
						amenity := model.Amenity{}

						// Extract name (required field)
						if name, ok := amenityMap["Name"].(string); ok {
							amenity.Name = name
						}

						// Extract optional fields
						if modifiable, ok := amenityMap["Modifiable"].(bool); ok {
							amenity.Modifiable = &modifiable
						}
						if description, ok := amenityMap["Description"].(string); ok {
							amenity.Description = &description
						}
						if isManaged, ok := amenityMap["IsManaged"].(bool); ok {
							amenity.IsManaged = &isManaged
						}
						if amenityType, ok := amenityMap["Type"].(string); ok {
							amenity.Type = &amenityType
						}
						if properties, ok := amenityMap["Properties"].(map[string]interface{}); ok {
							amenity.Properties = copyProperties(properties)
						}

						// Only add amenity if it has a name
						if amenity.Name != "" {
							amenities = append(amenities, amenity)
						}
					}
				}

				result.Amenities = amenities
			}
		}
		return result
	}

	// Fallback: Try to convert interface{} to OpenAPI client type (if it's already converted)
	var config *openapiclient.DeploymentCellConfiguration
	switch v := cellConfig.(type) {
	case *openapiclient.DeploymentCellConfiguration:
		config = v
	case openapiclient.DeploymentCellConfiguration:
		config = &v
	default:
		// If we can't convert, return empty result
		return result
	}

	// Check if the config and its amenities exist
	if config == nil || config.Amenities == nil {
		return result
	}

	// Convert each amenity from the API response to our model
	amenities := make([]model.Amenity, 0, len(config.Amenities))

	for _, apiAmenity := range config.Amenities {
		// Safely handle potential nil apiAmenity and required fields
		if apiAmenity.Name != nil {
			amenity := model.Amenity{
				Name:        *apiAmenity.Name,
				Modifiable:  apiAmenity.Modifiable,
				Description: apiAmenity.Description,
				IsManaged:   apiAmenity.IsManaged,
				Type:        apiAmenity.Type,
				Properties:  copyProperties(apiAmenity.Properties),
			}
			amenities = append(amenities, amenity)
		}
	}

	result.Amenities = amenities
	return result
}

func copyProperties(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}
