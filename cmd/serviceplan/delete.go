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
	deleteExample = `# Delete service plan
omnistrate-ctl service-plan delete [service-name] [plan-name]

# Delete service plan by ID instead of name
omnistrate-ctl service-plan delete --service-id=[service-id] --plan-id=[plan-id]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [service-name] [plan-name] [flags]",
	Short:        "Delete a Service Plan",
	Long:         `This command helps you delete a Service Plan from your service.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to delete the service plan in a specific environment")
	deleteCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	deleteCmd.Flags().StringP("plan-id", "", "", "Plan ID. Required if plan name is not provided")
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, _ := cmd.Flags().GetString("output")
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")
	environment, _ := cmd.Flags().GetString("environment")

	// Validate input arguments
	if err := validateDeleteArguments(args, serviceID, planID); err != nil {
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
		msg := "Deleting service plan..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if the service plan exists
	serviceID, _, planID, _, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete service plan
	err = dataaccess.DeleteProductTier(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Service plan deleted successfully")

	return nil
}

func validateDeleteArguments(args []string, serviceID, planID string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and plan name or the service ID and plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	return nil
}
