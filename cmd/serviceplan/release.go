package serviceplan

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	releaseExample = `# Release service plan by name
omnistrate-ctl service-plan release [service-name] [plan-name]

# Release service plan by ID
omnistrate-ctl service-plan release --service-id=[service-id] --plan-id=[plan-id]`
)

var releaseCmd = &cobra.Command{
	Use:          "release [service-name] [plan-name] [flags]",
	Short:        "Release a Service Plan",
	Long:         `This command helps you release a Service Plan for your service. You can specify a custom release description and set the service plan as preferred if needed.`,
	Example:      releaseExample,
	RunE:         runRelease,
	SilenceUsage: true,
}

func init() {
	releaseCmd.Flags().String("release-description", "", "Set custom release description for this release version")
	releaseCmd.Flags().Bool("release-as-preferred", false, "Release the service plan as preferred")
	releaseCmd.Flags().Bool("dryrun", false, "Perform a dry run without making any changes")
	releaseCmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to release the service plan in a specific environment")

	releaseCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	releaseCmd.Flags().StringP("plan-id", "", "", "Plan ID. Required if plan name is not provided")
}

func runRelease(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	releaseDescription, _ := cmd.Flags().GetString("release-description")
	releaseAsPreferred, _ := cmd.Flags().GetBool("release-as-preferred")
	dryRun, _ := cmd.Flags().GetBool("dryrun")
	output, _ := cmd.Flags().GetString("output")
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")
	environment, _ := cmd.Flags().GetString("environment")

	// Validate input arguments
	if err := validateReleaseArguments(args, serviceID, planID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and plan names if provided in args
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
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Releasing service plan..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if service plan exists
	serviceID, _, planID, _, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get service api id
	productTier, err := dataaccess.DescribeProductTier(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	serviceModel, err := dataaccess.DescribeServiceModel(cmd.Context(), token, serviceID, productTier.ServiceModelId)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	serviceAPIID := serviceModel.ServiceApiId

	// Release service plan
	err = dataaccess.ReleaseServicePlan(cmd.Context(), token, serviceID, serviceAPIID, planID, getReleaseDescription(releaseDescription), releaseAsPreferred, dryRun)
	if err != nil {
		spinner.Error()
		sm.Stop()
		utils.PrintError(err)
		return err
	}

	if !dryRun {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully released service plan")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully performed dry run for service plan release")
	}
	// Get the service plan details
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("serviceplan:%s", planID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	targetVersion, err := dataaccess.FindLatestVersion(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var targetServicePlan openapiclientfleet.ServicePlanSearchRecord
	for _, servicePlan := range searchRes.ServicePlanResults {
		if servicePlan.ServiceId != serviceID || servicePlan.Id != planID || servicePlan.Version != targetVersion {
			continue
		}
		targetServicePlan = servicePlan
	}

	// Format output
	formattedServicePlan := formatServicePlanVersion(targetServicePlan, false)

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedServicePlan); err != nil {
		return err
	}

	return nil
}

func validateReleaseArguments(args []string, serviceID, planID string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and service plan name or the service ID and service plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	return nil
}

func getReleaseDescription(releaseDescription string) *string {
	if releaseDescription == "" {
		return nil
	}
	return &releaseDescription
}
