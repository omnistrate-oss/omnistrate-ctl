package serviceplan

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	updateExample = `# Update service plan version name
omnistrate-ctl service-plan update [service-name] [plan-name] --version=[version] --name=[new-name]

# Update service plan version name by ID instead of name
omnistrate-ctl service-plan update --service-id=[service-id] --plan-id=[plan-id] --version=[version] --name=[new-name]`
)

func newUpdateCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [service-name] [plan-name] --version=[version] --name=[new-name] [flags]",
		Short: "Update Service Plan properties",
		Long: `This command helps you update various properties of a Service Plan.
Currently supports updating the name of a specific version of a Service Plan.
The version name is used as the release description for the version.`,
		Example:      servicePlanExample(commandPath, updateExample),
		RunE:         runUpdate,
		SilenceUsage: true,
	}

	cmd.Flags().String("version", "", "Specify the version number to update the name for.")
	cmd.Flags().String("name", "", "Specify the new name for the version.")
	cmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to update the version name in a specific environment")
	cmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	cmd.Flags().StringP("plan-id", "", "", "Plan ID. Required if plan name is not provided")

	err := cmd.MarkFlagRequired("version")
	if err != nil {
		return cmd
	}
	err = cmd.MarkFlagRequired("name")
	if err != nil {
		return cmd
	}
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")
	version, _ := cmd.Flags().GetString("version")
	newName, _ := cmd.Flags().GetString("name")
	environment, _ := cmd.Flags().GetString("environment")

	// Validate input arguments
	if err := validateUpdateArguments(args, serviceID, planID); err != nil {
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

	// Initialize spinner
	sm := utils.NewSpinnerManager()
	spinner := sm.AddSpinner("Updating service plan version name...")
	sm.Start()

	// Check if the service plan exists
	serviceID, _, planID, _, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get the target version
	version, err = getTargetVersion(cmd.Context(), token, serviceID, planID, version)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Update the version set name
	_, err = dataaccess.UpdateVersionSetName(cmd.Context(), token, serviceID, planID, version, newName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle success
	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Service plan version '%s' name updated successfully to '%s'", version, newName))

	return nil
}

// Helper functions

func validateUpdateArguments(args []string, serviceID, planID string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and service plan name or the service ID and service plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	return nil
}
