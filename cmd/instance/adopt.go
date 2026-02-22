package instance

import (
	"context"
	"fmt"
	"os"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

// AdoptionConfig represents the YAML configuration structure for resource adoption
type AdoptionConfig struct {
	ResourceAdoptionConfiguration map[string]ResourceAdoptionConfig `yaml:"resourceAdoptionConfiguration"`
}

// ResourceAdoptionConfig represents the configuration for adopting a single resource
type ResourceAdoptionConfig struct {
	HelmAdoptionConfiguration *HelmAdoptionConfig `yaml:"helmAdoptionConfiguration,omitempty"`
}

// HelmAdoptionConfig represents the Helm adoption configuration
type HelmAdoptionConfig struct {
	ChartRepoURL         string             `yaml:"chartRepoURL"`
	Password             *string            `yaml:"password,omitempty"`
	ReleaseName          string             `yaml:"releaseName"`
	ReleaseNamespace     string             `yaml:"releaseNamespace"`
	RuntimeConfiguration *HelmRuntimeConfig `yaml:"runtimeConfiguration,omitempty"`
	Username             *string            `yaml:"username,omitempty"`
}

// HelmRuntimeConfig represents the Helm runtime configuration
type HelmRuntimeConfig struct {
	DisableHooks         bool  `yaml:"disableHooks"`
	Recreate             bool  `yaml:"recreate"`
	ResetThenReuseValues bool  `yaml:"resetThenReuseValues"`
	ResetValues          bool  `yaml:"resetValues"`
	ReuseValues          bool  `yaml:"reuseValues"`
	SkipCRDs             bool  `yaml:"skipCRDs"`
	TimeoutNanos         int64 `yaml:"timeoutNanos"`
	UpgradeCRDs          bool  `yaml:"upgradeCRDs"`
	Wait                 bool  `yaml:"wait"`
	WaitForJobs          bool  `yaml:"waitForJobs"`
}

const adoptExample = `# Adopt a resource instance with basic parameters
omnistrate-ctl instance adopt --service-id my-service --service-plan-id my-plan --host-cluster-id my-cluster --primary-resource-key my-resource

# Adopt a resource instance with YAML configuration file
omnistrate-ctl instance adopt --service-id my-service --service-plan-id my-plan --host-cluster-id my-cluster --primary-resource-key my-resource --config-file adoption-config.yaml

# Example adoption-config.yaml format:
resourceAdoptionConfiguration:
  myRedis:
    helmAdoptionConfiguration:
      chartRepoURL: "https://charts.bitnami.com/bitnami"
      releaseName: "my-redis-instance"
      releaseNamespace: "default"
      username: "admin"
      password: "secretpassword"
      runtimeConfiguration:
        disableHooks: false
        recreate: false
        resetThenReuseValues: false
        resetValues: false
        reuseValues: true
        skipCRDs: false
        timeoutNanos: 300000000000  # 5 minutes in nanoseconds
        upgradeCRDs: true
        wait: true
        waitForJobs: true
  myDatabase:
    helmAdoptionConfiguration:
      chartRepoURL: "https://charts.example.com/postgres"
      releaseName: "my-postgres-instance"
      releaseNamespace: "production"
      runtimeConfiguration:
        disableHooks: false
        recreate: true
        resetThenReuseValues: false
        resetValues: false
        reuseValues: false
        skipCRDs: true
        timeoutNanos: 600000000000  # 10 minutes in nanoseconds
        upgradeCRDs: false
        wait: true
        waitForJobs: false`

var adoptCmd = &cobra.Command{
	Use:          "adopt",
	Short:        "Adopt a resource instance",
	Long:         `Adopt a resource instance with the specified parameters and optional resource adoption configuration.`,
	Example:      adoptExample,
	RunE:         runAdopt,
	SilenceUsage: true,
}

func init() {
	adoptCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	adoptCmd.Flags().StringP("service-plan-id", "p", "", "Service plan ID (required)")
	adoptCmd.Flags().StringP("host-cluster-id", "c", "", "Host cluster ID (required)")
	adoptCmd.Flags().StringP("primary-resource-key", "k", "", "Primary resource key to adopt (required)")
	adoptCmd.Flags().StringP("service-plan-version", "g", "", "Service plan version (optional)")
	adoptCmd.Flags().StringP("subscription-id", "u", "", "Subscription ID (optional)")
	adoptCmd.Flags().StringP("customer-email", "e", "", "Customer email for notifications (optional)")
	adoptCmd.Flags().StringP("config-file", "f", "", "YAML file containing resource adoption configuration (optional)")

	_ = adoptCmd.MarkFlagRequired("service-id")
	_ = adoptCmd.MarkFlagRequired("service-plan-id")
	_ = adoptCmd.MarkFlagRequired("host-cluster-id")
	_ = adoptCmd.MarkFlagRequired("primary-resource-key")
}

func runAdopt(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	servicePlanID, err := cmd.Flags().GetString("service-plan-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	hostClusterID, err := cmd.Flags().GetString("host-cluster-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	primaryResourceKey, err := cmd.Flags().GetString("primary-resource-key")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	servicePlanVersion, err := cmd.Flags().GetString("service-plan-version")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	subscriptionID, err := cmd.Flags().GetString("subscription-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	customerEmail, err := cmd.Flags().GetString("customer-email")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if customerEmail != "" && subscriptionID != "" {
		utils.PrintError(fmt.Errorf("cannot specify both customer email and subscription ID"))
		return fmt.Errorf("cannot specify both customer email and subscription ID")
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

	// Retrieve output format flag
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// If the customer email is provided, lookup subscription for this customer for the given service and plan
	if customerEmail != "" {
		subscription, err := dataaccess.GetSubscriptionByCustomerEmail(context.Background(), token, serviceID, servicePlanID, customerEmail)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to retrieve subscription ID for customer email %s: %w", customerEmail, err))
			return err
		}
		subscriptionID = subscription.Id
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Adopting resource instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()

		utils.PrintInfo("Adopting resource instance:")
		utils.PrintInfo(fmt.Sprintf("  Service ID: %s", serviceID))
		utils.PrintInfo(fmt.Sprintf("  Service Plan ID: %s", servicePlanID))
		utils.PrintInfo(fmt.Sprintf("  Host Cluster ID: %s", hostClusterID))
		utils.PrintInfo(fmt.Sprintf("  Primary Resource Key: %s", primaryResourceKey))
		if servicePlanVersion != "" {
			utils.PrintInfo(fmt.Sprintf("  Service Plan Version: %s", servicePlanVersion))
		}
		if subscriptionID != "" {
			utils.PrintInfo(fmt.Sprintf("  Subscription ID: %s", subscriptionID))
		}
		if configFile != "" {
			utils.PrintInfo(fmt.Sprintf("  Config File: %s", configFile))
		}
	}

	// Parse the config file if provided
	var adoptionConfig *AdoptionConfig
	if configFile != "" {
		adoptionConfig, err = parseConfigFile(configFile)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to parse config file: %w", err))
			return err
		}
	}

	// Build the adoption request
	request := openapiclientfleet.AdoptResourceInstanceRequest2{}

	// Convert YAML config to SDK format if provided
	if adoptionConfig != nil && len(adoptionConfig.ResourceAdoptionConfiguration) > 0 {
		sdkConfig := convertToSDKResourceAdoptionConfiguration(adoptionConfig.ResourceAdoptionConfiguration)
		request.ResourceAdoptionConfiguration = &sdkConfig
	}

	// Prepare optional parameters
	var servicePlanVersionPtr *string
	if servicePlanVersion != "" {
		servicePlanVersionPtr = &servicePlanVersion
	}
	var subscriptionIDPtr *string
	if subscriptionID != "" {
		subscriptionIDPtr = &subscriptionID
	}

	// Perform the actual adoption using the SDK
	result, err := dataaccess.AdoptResourceInstance(ctx, token, serviceID, servicePlanID, hostClusterID, primaryResourceKey, request, servicePlanVersionPtr, subscriptionIDPtr)
	if err != nil {
		if spinner != nil {
			utils.HandleSpinnerError(spinner, sm, err)
		} else {
			utils.PrintError(fmt.Errorf("failed to adopt resource instance: %w", err))
		}
		return err
	}

	if spinner != nil {
		utils.HandleSpinnerSuccess(spinner, sm, "Resource instance adoption initiated successfully!")
	}

	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if result.GetId() != "" {
		utils.PrintSuccess(fmt.Sprintf("Adoption initiated successfully! Instance ID: %s", result.GetId()))
	} else {
		utils.PrintSuccess("Adoption initiated successfully!")
	}

	return nil
}

// parseConfigFile reads and parses the YAML configuration file
func parseConfigFile(configFile string) (*AdoptionConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var config AdoptionConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

// convertToSDKResourceAdoptionConfiguration converts YAML config to SDK format
func convertToSDKResourceAdoptionConfiguration(yamlConfig map[string]ResourceAdoptionConfig) map[string]openapiclientfleet.ResourceAdoptionConfiguration {
	sdkConfig := make(map[string]openapiclientfleet.ResourceAdoptionConfiguration)

	for resourceKey, resourceConfig := range yamlConfig {
		sdkResourceConfig := openapiclientfleet.ResourceAdoptionConfiguration{}

		if resourceConfig.HelmAdoptionConfiguration != nil {
			helmConfig := convertToSDKHelmAdoptionConfiguration(resourceConfig.HelmAdoptionConfiguration)
			sdkResourceConfig.HelmAdoptionConfiguration = &helmConfig
		}

		sdkConfig[resourceKey] = sdkResourceConfig
	}

	return sdkConfig
}

// convertToSDKHelmAdoptionConfiguration converts YAML Helm config to SDK format
func convertToSDKHelmAdoptionConfiguration(yamlConfig *HelmAdoptionConfig) openapiclientfleet.HelmAdoptionConfiguration {
	sdkConfig := openapiclientfleet.HelmAdoptionConfiguration{
		ChartRepoURL:     yamlConfig.ChartRepoURL,
		ReleaseName:      yamlConfig.ReleaseName,
		ReleaseNamespace: yamlConfig.ReleaseNamespace,
	}

	if yamlConfig.Username != nil {
		sdkConfig.Username = yamlConfig.Username
	}

	if yamlConfig.Password != nil {
		sdkConfig.Password = yamlConfig.Password
	}

	if yamlConfig.RuntimeConfiguration != nil {
		runtimeConfig := convertToSDKHelmRuntimeConfiguration(yamlConfig.RuntimeConfiguration)
		sdkConfig.RuntimeConfiguration = &runtimeConfig
	}

	return sdkConfig
}

// convertToSDKHelmRuntimeConfiguration converts YAML runtime config to SDK format
func convertToSDKHelmRuntimeConfiguration(yamlConfig *HelmRuntimeConfig) openapiclientfleet.HelmRuntimeConfiguration {
	return openapiclientfleet.HelmRuntimeConfiguration{
		DisableHooks:         utils.ToPtr(yamlConfig.DisableHooks),
		Recreate:             utils.ToPtr(yamlConfig.Recreate),
		ResetThenReuseValues: utils.ToPtr(yamlConfig.ResetThenReuseValues),
		ResetValues:          utils.ToPtr(yamlConfig.ResetValues),
		ReuseValues:          utils.ToPtr(yamlConfig.ReuseValues),
		SkipCRDs:             utils.ToPtr(yamlConfig.SkipCRDs),
		TimeoutNanos:         utils.ToPtr(yamlConfig.TimeoutNanos),
		UpgradeCRDs:          utils.ToPtr(yamlConfig.UpgradeCRDs),
		Wait:                 utils.ToPtr(yamlConfig.Wait),
		WaitForJobs:          utils.ToPtr(yamlConfig.WaitForJobs),
	}
}
