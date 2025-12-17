package serviceplan

import (
	"fmt"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	enableFeatureExample = `# Enable service plan feature 
omnistrate-ctl service-plan enable-feature [service-name] [plan-name] --feature [feature-name] --feature-configuration [feature-configuration]

# Enable service plan feature by ID instead of name and configure using file
omnistrate-ctl service-plan enable-feature --service-id [service-id] --plan-id [plan-id] --feature [feature-name] --feature-configuration-file /path/to/feature-config-file.json`
)

var enableCmd = &cobra.Command{
	Use:          "enable-feature [service-name] [plan-name] [flags]",
	Short:        "Enable feature for a service plan",
	Long:         `This command helps you enable & configure service plan features such as CUSTOM_TERRAFORM_POLICY.`,
	Example:      enableFeatureExample,
	RunE:         runEnableFeature,
	SilenceUsage: true,
}

func init() {
	enableCmd.Flags().StringP(EnvironmentFlag, "", "", "Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment")
	enableCmd.Flags().StringP(ServiceIDFlag, "", "", "Service ID. Required if service name is not provided")
	enableCmd.Flags().StringP(PlanIDFlag, "", "", "Environment ID. Required if plan name is not provided")

	enableCmd.Flags().String(FeatureNameFlag, "", "Name / identifier of the feature to enable")
	enableCmd.Flags().String(FeatureConfigurationFlag, "", "Configuration of the feature")
	enableCmd.Flags().String(FeatureConfigurationFileFlag, "", "Json file containing feature configuration")
	if err := enableCmd.MarkFlagFilename(FeatureConfigurationFileFlag); err != nil {
		return
	}
}

func runEnableFeature(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	serviceID, _ := cmd.Flags().GetString(ServiceIDFlag)
	planID, _ := cmd.Flags().GetString(PlanIDFlag)
	environment, _ := cmd.Flags().GetString(EnvironmentFlag)
	featureName, _ := cmd.Flags().GetString(FeatureNameFlag)
	featureConfiguration, _ := cmd.Flags().GetString(FeatureConfigurationFlag)
	featureConfigurationFile, _ := cmd.Flags().GetString(FeatureConfigurationFileFlag)

	// Validate input arguments
	if err := validateEnableArguments(args, serviceID, planID, featureName, featureConfiguration, featureConfigurationFile); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and service plan names if provided in args
	var serviceName, planName string
	if len(args) == 2 {
		serviceName, planName = args[0], args[1]
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
	sm = ysmrr.NewSpinnerManager()
	msg := "Enabling service plan feature..."
	spinner = sm.AddSpinner(msg)
	sm.Start()

	// Format parameters
	var featureConfigMap map[string]any
	if len(featureConfigurationFile) > 0 {
		// Format parameters
		featureConfigMap, err = common.FormatParams(featureConfiguration, featureConfigurationFile)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Check if the service plan exists
	serviceID, _, planID, _, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe the service plan
	servicePlan, err := dataaccess.DescribeProductTier(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	modelId := servicePlan.ServiceModelId

	// Update service model
	err = dataaccess.EnableServiceModelFeature(cmd.Context(), token, serviceID, modelId, featureName, featureConfigMap)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle output based on format
	utils.HandleSpinnerSuccess(spinner, sm, "Feature enabled successfully")
	return nil
}

func validateEnableArguments(args []string, serviceID, planID, featureName, featureConfiguration, featureConfigurationFile string) (
	err error) {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and service plan name or the service ID and service plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	if len(featureName) == 0 {
		return fmt.Errorf("missing parameter '%s'", FeatureNameFlag)
	}
	if len(featureConfiguration) > 0 && len(featureConfigurationFile) > 0 {
		return fmt.Errorf("cannot provide value for both '%s' and '%s'", FeatureConfigurationFlag, FeatureConfigurationFileFlag)
	}
	return
}
