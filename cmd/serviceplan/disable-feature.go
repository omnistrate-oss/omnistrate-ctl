package serviceplan

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	disableFeatureExample = `# Disable service plan feature 
omnistrate-ctl service-plan disable-feature [service-name] [plan-name] --feature [feature-name]

#  Disable service plan feature by ID instead of name
omnistrate-ctl service-plan enable-feature --service-id [service-id] --plan-id [plan-id] --feature [feature-name]`
)

var disableCmd = &cobra.Command{
	Use:          "disable-feature [service-name] [plan-name] [flags]",
	Short:        "Disable feature for a service plan",
	Long:         `This command helps you disable active service plan feature.`,
	Example:      disableFeatureExample,
	RunE:         runDisableFeature,
	SilenceUsage: true,
}

func init() {
	disableCmd.Flags().StringP(EnvironmentFlag, "", "", "Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment")
	disableCmd.Flags().StringP(ServiceIDFlag, "", "", "Service ID. Required if service name is not provided")
	disableCmd.Flags().StringP(PlanIDFlag, "", "", "Environment ID. Required if plan name is not provided")

	disableCmd.Flags().String(FeatureNameFlag, "", "Name / identifier of the feature to disable")
}

func runDisableFeature(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	serviceID, _ := cmd.Flags().GetString(ServiceIDFlag)
	planID, _ := cmd.Flags().GetString(PlanIDFlag)
	environment, _ := cmd.Flags().GetString(EnvironmentFlag)
	featureName, _ := cmd.Flags().GetString(FeatureNameFlag)

	// Validate input arguments
	if err := validateDisableArguments(args, serviceID, planID, featureName); err != nil {
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
	msg := "Disabling service plan feature..."
	spinner = sm.AddSpinner(msg)
	sm.Start()

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
	err = dataaccess.DisableServiceModelFeature(cmd.Context(), token, serviceID, modelId, featureName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle output based on format
	utils.HandleSpinnerSuccess(spinner, sm, "Feature disabled successfully")
	return nil
}

func validateDisableArguments(args []string, serviceID, planID, featureName string) (
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
	return
}
