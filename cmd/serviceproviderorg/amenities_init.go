package serviceproviderorg

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

var amenitiesInitCmd = &cobra.Command{
	Use:   "init-deployment-cell-config-template",
	Short: "Initialize deployment cell configuration template for service provider organization",
	Long: `Initialize service provider organization-level deployment cell configuration template through an interactive process.

This command starts an interactive process to define the default organization-level 
configuration template for deployment cells. This step is purely at the service provider org level; 
no reference to any specific service is needed.

The configuration will be stored as a template that can be applied to different 
environments (production, staging, development) and used to synchronize deployment cells.

Organization ID is automatically determined from your credentials.

Examples:
  # Initialize deployment cell configuration template interactively
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template

  # Initialize from YAML file
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template -f sample-amenities.yaml`,
	RunE:         runAmenitiesInit,
	SilenceUsage: true,
}

func init() {
	amenitiesInitCmd.Flags().StringP("config-file", "f", "", "Path to configuration YAML file (optional)")
	amenitiesInitCmd.Flags().Bool("interactive", true, "Use interactive mode to configure amenities")
}

func runAmenitiesInit(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

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

	var configTemplate map[string]interface{}

	if configFile != "" {
		// Load configuration from file
		configTemplate, err = loadConfigurationFromYAMLFile(configFile)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to load configuration from file: %w", err))
			return err
		}
	} else if interactive {
		// Interactive configuration setup
		configTemplate, err = interactiveConfigurationSetup()
		if err != nil {
			utils.PrintError(err)
			return err
		}
	} else {
		// Use default configuration
		configTemplate = getDefaultAmenitiesConfiguration()
	}

	// Validate configuration
	err = dataaccess.ValidateAmenitiesConfiguration(configTemplate)
	if err != nil {
		utils.PrintError(fmt.Errorf("configuration validation failed: %w", err))
		return err
	}

	// Initialize the configuration (organization ID comes from token/credentials)
	config, err := dataaccess.InitializeOrganizationAmenitiesConfiguration(ctx, token, configTemplate)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess("Successfully initialized deployment cell configuration template for service provider organization")

	// Print the configuration details
	if output == "table" {
		tableView := config.ToTableView()
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
	} else {
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{config})
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func interactiveConfigurationSetup() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	fmt.Println("\n🚀 Interactive Deployment Cell Configuration Template Setup")
	fmt.Println("Configure the default organization-level deployment cell configuration settings.")

	// Logging configuration
	loggingConfig, err := configureLogging()
	if err != nil {
		return nil, err
	}
	config["logging"] = loggingConfig

	// Monitoring configuration
	monitoringConfig, err := configureMonitoring()
	if err != nil {
		return nil, err
	}
	config["monitoring"] = monitoringConfig

	// Security configuration
	securityConfig, err := configureSecurity()
	if err != nil {
		return nil, err
	}
	config["security"] = securityConfig

	// Ask if user wants to add more sections
	addMoreChoice, err := prompt.New().Ask("Would you like to add additional configuration sections?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	addMore := addMoreChoice == "Yes"

	if addMore {
		customConfig, err := configureCustomSection()
		if err != nil {
			return nil, err
		}
		for key, value := range customConfig {
			config[key] = value
		}
	}

	return config, nil
}

func configureLogging() (map[string]interface{}, error) {
	fmt.Println("📋 Configuring Logging Settings")
	
	levelOptions := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	levelChoice, err := prompt.New().Ask("Select logging level:").Choose(levelOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	rotationOptions := []string{"daily", "weekly", "monthly", "size-based"}
	rotationChoice, err := prompt.New().Ask("Select log rotation policy:").Choose(rotationOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableStructuredChoice, err := prompt.New().Ask("Enable structured logging (JSON format)?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableStructured := enableStructuredChoice == "Yes"

	return map[string]interface{}{
		"level":            levelChoice,
		"rotation":         rotationChoice,
		"structured":       enableStructured,
		"retention_days":   30,
	}, nil
}

func configureMonitoring() (map[string]interface{}, error) {
	fmt.Println("\n📊 Configuring Monitoring Settings")
	
	enableMonitoringChoice, err := prompt.New().Ask("Enable monitoring?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableMonitoring := enableMonitoringChoice == "Yes"

	if !enableMonitoring {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	enablePrometheusChoice, err := prompt.New().Ask("Enable Prometheus metrics?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enablePrometheus := enablePrometheusChoice == "Yes"

	enableGrafanaChoice, err := prompt.New().Ask("Enable Grafana dashboards?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableGrafana := enableGrafanaChoice == "Yes"

	enableAlertingChoice, err := prompt.New().Ask("Enable alerting?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableAlerting := enableAlertingChoice == "Yes"

	return map[string]interface{}{
		"enabled":    true,
		"prometheus": enablePrometheus,
		"grafana":    enableGrafana,
		"alerting":   enableAlerting,
		"retention":  "30d",
	}, nil
}

func configureSecurity() (map[string]interface{}, error) {
	fmt.Println("\n🔒 Configuring Security Settings")
	
	encryptionOptions := []string{"AES128", "AES256", "ChaCha20-Poly1305"}
	encryptionChoice, err := prompt.New().Ask("Select encryption algorithm:").Choose(encryptionOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	tlsOptions := []string{"1.2", "1.3"}
	tlsChoice, err := prompt.New().Ask("Select minimum TLS version:").Choose(tlsOptions, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableHSTSChoice, err := prompt.New().Ask("Enable HTTP Strict Transport Security (HSTS)?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
	if err != nil {
		return nil, err
	}

	enableHSTS := enableHSTSChoice == "Yes"

	return map[string]interface{}{
		"encryption":  encryptionChoice,
		"tls_version": tlsChoice,
		"hsts":        enableHSTS,
		"csrf_protection": true,
	}, nil
}

func configureCustomSection() (map[string]interface{}, error) {
	fmt.Println("\n⚙️  Adding Custom Configuration Section")
	
	sectionName, err := prompt.New().Ask("Enter section name:").Input("")
	if err != nil {
		return nil, err
	}

	if sectionName == "" {
		return map[string]interface{}{}, nil
	}

	// For simplicity, we'll just ask for key-value pairs
	customSection := make(map[string]interface{})
	
	for {
		key, err := prompt.New().Ask("Enter configuration key (empty to finish):").Input("")
		if err != nil {
			return nil, err
		}
		
		if key == "" {
			break
		}

		value, err := prompt.New().Ask(fmt.Sprintf("Enter value for '%s':", key)).Input("")
		if err != nil {
			return nil, err
		}

		customSection[key] = value
	}

	return map[string]interface{}{
		sectionName: customSection,
	}, nil
}

func loadConfigurationFromYAMLFile(filePath string) (map[string]interface{}, error) {
	data, err := utils.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration YAML: %w", err)
	}

	return config, nil
}

func getDefaultAmenitiesConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"logging": map[string]interface{}{
			"level":            "INFO",
			"rotation":         "daily",
			"structured":       true,
			"retention_days":   30,
		},
		"monitoring": map[string]interface{}{
			"enabled":    true,
			"prometheus": true,
			"grafana":    true,
			"alerting":   false,
			"retention":  "30d",
		},
		"security": map[string]interface{}{
			"encryption":       "AES256",
			"tls_version":      "1.3",
			"hsts":            true,
			"csrf_protection": true,
		},
	}
}