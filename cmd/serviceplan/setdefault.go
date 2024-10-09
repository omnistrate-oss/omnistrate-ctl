package serviceplan

import (
	"fmt"
	"strings"

	"github.com/chelnak/ysmrr"
	inventoryapi "github.com/omnistrate/api-design/v1/pkg/fleet/gen/inventory_api"
	"github.com/omnistrate/ctl/internal/config"
	"github.com/omnistrate/ctl/internal/dataaccess"
	"github.com/omnistrate/ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	setDefaultExample = `# Set service plan as default
omctl service-plan set-default [service-name] [plan-name] --version=[version]

# Set  service plan as default by ID instead of name
omctl service-plan set-default --service-id=[service-id] --plan-id=[plan-id] --version=[version]`
)

var setDefaultCmd = &cobra.Command{
	Use:   "set-default [service-name] [plan-name] --version=[version] [flags]",
	Short: "Set a Version of a Service Plan as Default(Preferred)",
	Long: `This command helps you set a Version of a Service Plan as the default (preferred) version for your service.
By setting it as default, new instance deployments from your customers will be created with this version by default.`,
	Example:      setDefaultExample,
	RunE:         runSetDefault,
	SilenceUsage: true,
}

func init() {
	setDefaultCmd.Flags().String("version", "", "Specify the version number to set the default to. Use 'latest' to set the latest version as default.")
	setDefaultCmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to set the default version in a specific environment")
	setDefaultCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	setDefaultCmd.Flags().StringP("plan-id", "", "", "Plan ID. Required if plan name is not provided")

	err := setDefaultCmd.MarkFlagRequired("version")
	if err != nil {
		return
	}
}

func runSetDefault(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	version, _ := cmd.Flags().GetString("version")
	environment, _ := cmd.Flags().GetString("environment")
	output, _ := cmd.Flags().GetString("output")
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")

	// Validate input arguments
	if err := validateSetDefaultArguments(args, serviceID, planID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and plan names if provided in args
	var serviceName, planName string
	if len(args) == 2 {
		serviceName, planName = args[0], args[1]
	}

	// Validate user login
	token, err := config.GetToken()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Setting default service plan..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if the service plan exists
	serviceID, _, planID, _, _, err = getServicePlan(token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get the target version
	targetVersion, err := getTargetVersion(token, serviceID, planID, version)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Set the default service plan
	_, err = dataaccess.SetDefaultServicePlan(token, serviceID, planID, targetVersion)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully set default service plan")

	// Get the service plan details
	searchRes, err := dataaccess.SearchInventory(token, fmt.Sprintf("serviceplan:%s", planID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var targetServicePlan *inventoryapi.ServicePlanSearchRecord
	for _, servicePlan := range searchRes.ServicePlanResults {
		if string(servicePlan.ServiceID) != serviceID || servicePlan.ID != planID || servicePlan.Version != targetVersion {
			continue
		}
		targetServicePlan = servicePlan
	}

	// Format output
	formattedServicePlanVersion, err := formatServicePlanVersion(targetServicePlan, false)
	if err != nil {
		return err
	}

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedServicePlanVersion); err != nil {
		return err
	}

	return nil
}

func getServicePlan(token, serviceIDArg, serviceNameArg, planIDArg, planNameArg, envNameArg string) (serviceID, serviceName, planID, planName, environment string, err error) {
	searchRes, err := dataaccess.SearchInventory(token, "service:s")
	if err != nil {
		return
	}

	serviceFound := 0
	for _, service := range searchRes.ServiceResults {
		if !strings.EqualFold(service.Name, serviceNameArg) && service.ID != serviceIDArg {
			continue
		}
		serviceID = service.ID
		serviceFound += 1
	}

	if serviceFound == 0 {
		err = fmt.Errorf("service not found. Please check input values and try again")
		return
	}

	if serviceFound > 1 {
		err = fmt.Errorf("multiple services with the same name found. Please provide the service ID instead of the name")
		return
	}

	envFound := 0
	servicePlanFound := 0
	describeServiceRes, err := dataaccess.DescribeService(token, serviceID)
	if err != nil {
		return
	}
	for _, env := range describeServiceRes.ServiceEnvironments {
		if envNameArg != "" && !strings.EqualFold(envNameArg, env.Name) {
			continue
		}
		envFound += 1
		for _, servicePlan := range env.ServicePlans {
			if !strings.EqualFold(servicePlan.Name, planNameArg) && string(servicePlan.ProductTierID) != planIDArg {
				continue
			}
			environment = env.Name
			planID = string(servicePlan.ProductTierID)
			servicePlanFound += 1
		}
	}

	if envNameArg != "" && envFound == 0 {
		err = fmt.Errorf("environment not found. Please check input values and try again")
		return
	}

	if envNameArg != "" && envFound > 1 {
		err = fmt.Errorf("multiple environments with the same name found. Please provide the environment name instead of the name")
		return
	}

	if servicePlanFound == 0 {
		err = fmt.Errorf("service plan not found. Please check input values and try again")
		return
	}

	if servicePlanFound > 1 {
		err = fmt.Errorf("multiple service plans with the same name found. Please specify the environment or provide the plan ID instead of the name")
		return
	}

	return
}

func getTargetVersion(token, serviceID, productTierID, version string) (targetVersion string, err error) {
	switch version {
	case "latest":
		targetVersion, err = dataaccess.FindLatestVersion(token, serviceID, productTierID)
		if err != nil {
			return
		}
	case "preferred":
		targetVersion, err = dataaccess.FindPreferredVersion(token, serviceID, productTierID)
		if err != nil {
			return
		}
	default:
		targetVersion = version
	}

	return
}

func validateSetDefaultArguments(args []string, serviceID, planID string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and plan name or the service ID and plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	return nil
}
