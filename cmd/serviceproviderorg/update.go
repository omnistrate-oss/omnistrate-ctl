package serviceproviderorg

import (
	"context"
	"fmt"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var updateServiceProviderOrgCmd = &cobra.Command{
	Use:   "update",
	Short: "Update service provider organization configuration template for target environment",
	Long: `Update the service provider organization configuration template for a selected target environment.

You specify which environment the update applies to and provide a configuration file.
Updating the environment overrides the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the service provider org at any time.

Organization ID is automatically determined from your credentials.

Examples:
  # Update configuration for production environment
  omnistrate-ctl serviceproviderorg update -e PROD -f config-template.yaml

  # Update staging environment with configuration file
  omnistrate-ctl serviceproviderorg update -e STAGING -f config-template.yaml`,
	RunE:         runUpdateServiceProviderOrg,
	SilenceUsage: true,
}

func init() {
	updateServiceProviderOrgCmd.Flags().StringP("environment", "e", "", "Target environment (PROD, PRIVATE, CANARY, STAGING, QA, DEV)")
	updateServiceProviderOrgCmd.Flags().StringP("config-file", "f", "", "Path to configuration YAML file")
	updateServiceProviderOrgCmd.MarkFlagRequired("config-file")
}

// Valid environment types based on the EnvironmentType enum
var validEnvironments = []string{"PROD", "PRIVATE", "CANARY", "STAGING", "QA", "DEV"}

func runUpdateServiceProviderOrg(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate environment if provided
	if environment != "" {
		if !isValidEnvironment(environment) {
			utils.PrintError(fmt.Errorf("invalid environment '%s'. Valid values are: %v", environment, validEnvironments))
			return fmt.Errorf("invalid environment type")
		}
	}

	configFile, err := cmd.Flags().GetString("config-file")
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

	// Load configuration from file
	configTemplate, err := loadConfigurationFromYAMLFile(configFile)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to load configuration from file: %w", err))
		return err
	}

	// Confirm update if not from file
	if configFile == "" {
		fmt.Printf("\nConfiguration to be applied to %s environment:\n", environment)
		configYAML, _ := yaml.Marshal(configTemplate)
		fmt.Println(string(configYAML))

		confirmChoice, err := prompt.New().Ask("Do you want to apply this configuration?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			utils.PrintError(err)
			return err
		}

		confirm := confirmChoice == "Yes"

		if !confirm {
			utils.PrintInfo("Configuration update cancelled")
			return nil
		}
	}

	// Update the configuration (organization ID comes from token/credentials)
	err = dataaccess.UpdateServiceProviderOrganization(ctx, token, configTemplate, environment)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Describe the updated configuration
	updatedConfig, err := dataaccess.GetServiceProviderOrganization(ctx, token)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to retrieve updated service provider org configuration: %w", err))
		return err
	}

	if updatedConfig == nil || updatedConfig.DeploymentCellConfigurationsPerEnv == nil {
		utils.PrintError(fmt.Errorf("no service provider organization configuration found for environment %s", environment))
		return fmt.Errorf("no service provider organization configuration found")
	}

	// transform to deploymentCellConfigurations
	if updatedConfig.DeploymentCellConfigurationsPerEnv[environment] == nil {
		utils.PrintError(fmt.Errorf("no service provider organization configuration found for environment %s", environment))
		return fmt.Errorf("no service provider organization configuration found for environment %s", environment)
	}

	outputModel, err := convertToDeploymentCellConfigurations(updatedConfig.DeploymentCellConfigurationsPerEnv[environment])
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to convert deployment cell configurations: %w", err))
		return err
	}

	// Print the updated configuration
	fmt.Printf("\nUpdated service provider organization configuration for %s environment:\n", environment)
	updatedConfigYAML, err := yaml.Marshal(outputModel)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to marshal updated configuration: %w", err))
		return err
	}

	fmt.Println(string(updatedConfigYAML))
	utils.PrintSuccess(fmt.Sprintf("Successfully updated service provider org configuration template for %s environment", environment))

	return nil
}

func loadConfigurationFromYAMLFile(filePath string) (model.DeploymentCellConfigurations, error) {
	data, err := utils.ReadFile(filePath)
	if err != nil {
		return model.DeploymentCellConfigurations{}, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var templateConfig model.DeploymentCellConfigurations
	err = yaml.Unmarshal(data, &templateConfig)
	if err != nil {
		return model.DeploymentCellConfigurations{}, fmt.Errorf("failed to parse configuration YAML: %w", err)
	}

	return templateConfig, nil
}

func isValidEnvironment(env string) bool {
	for _, valid := range validEnvironments {
		if env == valid {
			return true
		}
	}
	return false
}

// convertToDeploymentCellConfigurations converts the OpenAPI response to model.DeploymentCellConfigurationTemplate
func convertToDeploymentCellConfigurations(apiResponse interface{}) (model.DeploymentCellConfigurationTemplate, error) {
	var result model.DeploymentCellConfigurationTemplate

	// Initialize the DeploymentCellConfigurations map
	result.DeploymentCellConfigurations = make(map[string]model.DeploymentCellConfiguration)

	// Handle the case where apiResponse is nil
	if apiResponse == nil {
		return result, nil
	}

	// Try to convert interface{} to map[string]interface{} first (raw API response)
	if configMap, ok := apiResponse.(map[string]interface{}); ok {
		// Check if DeploymentCellConfigurationPerCloudProvider exists
		if deploymentCellConfigInterface, exists := configMap["DeploymentCellConfigurationPerCloudProvider"]; exists {
			if cloudProviderConfigs, ok := deploymentCellConfigInterface.(map[string]interface{}); ok {
				// Convert each cloud provider configuration
				for cloudProvider, cellConfigInterface := range cloudProviderConfigs {
					cellConfig := convertToDeploymentCellConfiguration(cellConfigInterface)
					result.DeploymentCellConfigurations[cloudProvider] = cellConfig
				}
			}
		}
		return result, nil
	}

	// Fallback: Try to convert if it's already an OpenAPI client type
	// This handles cases where the API response is already structured
	return result, fmt.Errorf("unexpected data type for API response: %T", apiResponse)
}
