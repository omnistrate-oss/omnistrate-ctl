package environment

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
	deleteExample = `# Delete environment
omnistrate-ctl environment delete [service-name] [environment-name]

# Delete environment by ID instead of name
omnistrate-ctl environment delete --service-id=[service-id] --environment-id=[environment-id]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [service-name] [environment-name] [flags]",
	Short:        "Delete a Service Environment",
	Long:         `This command helps you delete an environment from your service.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	deleteCmd.Flags().StringP("environment-id", "", "", "Environment ID. Required if environment name is not provided")
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, _ := cmd.Flags().GetString("output")
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	// Validate input arguments
	if err := validateDeleteArguments(args, serviceID, environmentID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and environment names if provided in args
	var serviceName, environmentName string
	if len(args) == 2 {
		serviceName, environmentName = args[0], args[1]
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
		spinner = sm.AddSpinner("Deleting environment...")
		sm.Start()
	}

	// Check if the environment exists
	serviceID, _, environmentID, err = getServiceEnvironment(cmd.Context(), token, serviceID, serviceName, environmentID, environmentName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete the environment
	if err = dataaccess.DeleteServiceEnvironment(cmd.Context(), token, serviceID, environmentID); err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle success message
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted environment")

	return nil
}

// Helper functions

func validateDeleteArguments(args []string, serviceID, environmentID string) error {
	if len(args) == 0 && (serviceID == "" || environmentID == "") {
		return fmt.Errorf("please provide the service name and environment name or the service ID and environment ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [environment-name]", strings.Join(args, " "))
	}
	return nil
}
