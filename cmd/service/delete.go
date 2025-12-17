package service

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	deleteExample = `# Delete service with name
omnistrate-ctl service delete [service-name]

# Delete service with ID
omnistrate-ctl service delete --id=[service-ID]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [service-name] [flags]",
	Short:        "Delete a service",
	Long:         `This command helps you delete a service using its name or ID.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Args = cobra.MaximumNArgs(1) // Require at most one argument

	deleteCmd.Flags().String("id", "", "Service ID")
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	// Retrieve flags
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate input args
	err = validateDeleteArguments(args, id)
	if err != nil {
		utils.PrintError(err)
		return err
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
		msg := "Deleting service..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if service exists
	id, err = getService(cmd.Context(), token, name, id)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete service
	err = dataaccess.DeleteService(cmd.Context(), token, id)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted service")

	return nil
}

func validateDeleteArguments(args []string, serviceIDArg string) error {
	if len(args) == 0 && serviceIDArg == "" {
		return errors.New("service name or ID must be provided")
	}

	if len(args) != 0 && serviceIDArg != "" {
		return errors.New("only one of service name or ID can be provided")
	}

	return nil
}
