package organization

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var amenitiesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update organization amenities configuration template for target environment",
	Long: `Update the amenities configuration template for a selected target environment.

You specify which environment the update applies to. Updating the environment 
overrides the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the org at any time.

Organization ID is automatically determined from your credentials.

Examples:
  # Update configuration for production environment
  omnistrate-ctl organization amenities update -e production

  # Update with configuration from file
  omnistrate-ctl organization amenities update -e staging -f config.yaml

  # Interactive update
  omnistrate-ctl organization amenities update -e development --interactive`,
	RunE:         runAmenitiesUpdate,
	SilenceUsage: true,
}

func init() {
	amenitiesUpdateCmd.Flags().StringP("environment", "e", "", "Target environment (production, staging, development)")
	amenitiesUpdateCmd.Flags().StringP("config-file", "f", "", "Path to configuration YAML file (optional)")
	amenitiesUpdateCmd.Flags().Bool("interactive", false, "Use interactive mode to update amenities configuration")
	amenitiesUpdateCmd.Flags().Bool("merge", false, "Merge with existing configuration instead of replacing")
}

func runAmenitiesUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	interactive, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	merge, err := cmd.Flags().GetBool("merge")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
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

	// If environment not specified, prompt for it
	if environment == "" {
		environments, err := dataaccess.ListAvailableEnvironments(ctx, token)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		var envOptions []string
		for _, env := range environments {
			envOptions = append(envOptions, fmt.Sprintf("%s (%s)", env.DisplayName, env.Description))
		}

		result, err := prompt.New().Ask("Select target environment to update:").Choose(envOptions, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			utils.PrintError(err)
			return err
		}

		// Extract environment name from selection
		for _, env := range environments {
			optionText := fmt.Sprintf("%s (%s)", env.DisplayName, env.Description)
			if result == optionText {
				environment = env.Name
				break
			}
		}
	}

	var configTemplate map[string]interface{}

	// Get existing configuration if merging
	if merge {
		existingConfig, err := dataaccess.GetOrganizationAmenitiesConfiguration(ctx, token, "", environment)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to get existing configuration for merging: %w", err))
			return err
		}
		configTemplate = existingConfig.ConfigurationTemplate
	}

	if configFile != "" {
		// Load configuration from file
		newConfig, err := loadConfigurationFromYAMLFile(configFile)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to load configuration from file: %w", err))
			return err
		}

		if merge && configTemplate != nil {
			configTemplate = mergeConfigurations(configTemplate, newConfig)
		} else {
			configTemplate = newConfig
		}
	} else if interactive {
		// Interactive configuration update
		if merge && configTemplate != nil {
			configTemplate, err = interactiveConfigurationUpdate(configTemplate)
		} else {
			configTemplate, err = interactiveConfigurationSetup()
		}
		if err != nil {
			utils.PrintError(err)
			return err
		}
	} else {
		if !merge {
			utils.PrintError(fmt.Errorf("must specify either --config-file or --interactive for configuration update"))
			return fmt.Errorf("configuration source required")
		}
		// If merge flag is set but no new config provided, show current config
		fmt.Printf("Current configuration for %s environment:\n", environment)
		currentConfigYAML, _ := yaml.Marshal(configTemplate)
		fmt.Println(string(currentConfigYAML))
		return nil
	}

	// Validate configuration
	err = dataaccess.ValidateAmenitiesConfiguration(configTemplate)
	if err != nil {
		utils.PrintError(fmt.Errorf("configuration validation failed: %w", err))
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
	updatedConfig, err := dataaccess.UpdateOrganizationAmenitiesConfiguration(ctx, token, "", environment, configTemplate)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully updated amenities configuration template for %s environment", environment))

	// Print the updated configuration details
	if output == "table" {
		tableView := updatedConfig.ToTableView()
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
	} else {
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{updatedConfig})
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func interactiveConfigurationUpdate(existingConfig map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("\n🔄 Interactive Amenities Configuration Update")
	fmt.Println("Current configuration will be shown for each section. You can modify or keep existing values.")

	config := make(map[string]interface{})

	// Copy existing configuration
	for key, value := range existingConfig {
		config[key] = value
	}

	// Update logging configuration
	if existingLogging, ok := config["logging"].(map[string]interface{}); ok {
		updatedLogging, err := updateLoggingConfig(existingLogging)
		if err != nil {
			return nil, err
		}
		config["logging"] = updatedLogging
	} else {
		loggingConfig, err := configureLogging()
		if err != nil {
			return nil, err
		}
		config["logging"] = loggingConfig
	}

	// Update monitoring configuration
	if existingMonitoring, ok := config["monitoring"].(map[string]interface{}); ok {
		updatedMonitoring, err := updateMonitoringConfig(existingMonitoring)
		if err != nil {
			return nil, err
		}
		config["monitoring"] = updatedMonitoring
	} else {
		monitoringConfig, err := configureMonitoring()
		if err != nil {
			return nil, err
		}
		config["monitoring"] = monitoringConfig
	}

	// Update security configuration
	if existingSecurity, ok := config["security"].(map[string]interface{}); ok {
		updatedSecurity, err := updateSecurityConfig(existingSecurity)
		if err != nil {
			return nil, err
		}
		config["security"] = updatedSecurity
	} else {
		securityConfig, err := configureSecurity()
		if err != nil {
			return nil, err
		}
		config["security"] = securityConfig
	}

	return config, nil
}

func updateLoggingConfig(existing map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("📋 Updating Logging Settings")
	
	// Show current values and ask for updates
	currentLevel, _ := existing["level"].(string)
	fmt.Printf("Current logging level: %s\n", currentLevel)
	
	levelOptions := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	
	levelChoice, err := prompt.New().Ask("Select new logging level:").Choose(levelOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	existing["level"] = levelChoice
	
	return existing, nil
}

func updateMonitoringConfig(existing map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("\n📊 Updating Monitoring Settings")
	
	currentEnabled, _ := existing["enabled"].(bool)
	fmt.Printf("Current monitoring status: %t\n", currentEnabled)
	
	enableMonitoringChoice, err := prompt.New().Ask("Enable monitoring?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableMonitoring := enableMonitoringChoice == "Yes"
	existing["enabled"] = enableMonitoring
	
	if enableMonitoring {
		enablePrometheusChoice, err := prompt.New().Ask("Enable Prometheus metrics?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			return nil, err
		}
		existing["prometheus"] = enablePrometheusChoice == "Yes"

		enableGrafanaChoice, err := prompt.New().Ask("Enable Grafana dashboards?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			return nil, err
		}
		existing["grafana"] = enableGrafanaChoice == "Yes"
	}
	
	return existing, nil
}

func updateSecurityConfig(existing map[string]interface{}) (map[string]interface{}, error) {
	fmt.Println("\n🔒 Updating Security Settings")
	
	currentEncryption, _ := existing["encryption"].(string)
	fmt.Printf("Current encryption: %s\n", currentEncryption)
	
	encryptionOptions := []string{"AES128", "AES256", "ChaCha20-Poly1305"}
	
	encryptionChoice, err := prompt.New().Ask("Select encryption algorithm:").Choose(encryptionOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	existing["encryption"] = encryptionChoice
	
	return existing, nil
}

func mergeConfigurations(existing, new map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Start with existing configuration
	for key, value := range existing {
		result[key] = value
	}
	
	// Merge new configuration
	for key, value := range new {
		if existingValue, exists := result[key]; exists {
			// If both are maps, merge recursively
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if newMap, ok := value.(map[string]interface{}); ok {
					result[key] = mergeConfigurations(existingMap, newMap)
					continue
				}
			}
		}
		// Otherwise, replace with new value
		result[key] = value
	}
	
	return result
}