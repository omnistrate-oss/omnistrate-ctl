package service

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const resourceDeleteExample = `# Delete a resource with service and resource IDs
omnistrate-ctl service resource delete --service-id=[service-ID] --resource-id=[resource-ID]`

func newResourceCmd() *cobra.Command {
	resourceCmd := &cobra.Command{
		Use:          "resource [operation] [flags]",
		Short:        "Manage service resources",
		Long:         `This command helps you manage resources for a service.`,
		RunE:         runResource,
		SilenceUsage: true,
	}

	resourceCmd.AddCommand(newResourceDeleteCmd())

	return resourceCmd
}

func runResource(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func newResourceDeleteCmd() *cobra.Command {
	resourceDeleteCmd := &cobra.Command{
		Use:          "delete --service-id [service-ID] --resource-id [resource-ID] [flags]",
		Short:        "Delete a service resource",
		Long:         `This command helps you delete a resource from a service.`,
		Example:      resourceDeleteExample,
		Args:         cobra.NoArgs,
		RunE:         runResourceDelete,
		SilenceUsage: true,
	}

	resourceDeleteCmd.Flags().String("service-id", "", "Service ID")
	resourceDeleteCmd.Flags().String("resource-id", "", "Resource ID")

	return resourceDeleteCmd
}

func runResourceDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	resourceID, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return runResourceDeleteWithOptions(cmd, serviceID, resourceID, output)
}

func validateResourceDeleteArguments(serviceID, resourceID string) error {
	if serviceID == "" {
		return errors.New("service ID must be provided")
	}
	if resourceID == "" {
		return errors.New("resource ID must be provided")
	}
	return nil
}

func runResourceDeleteWithOptions(cmd *cobra.Command, serviceID, resourceID, output string) error {
	err := validateResourceDeleteArguments(serviceID, resourceID)
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
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Deleting resource...")
		sm.Start()
	}

	// Check if service exists
	serviceID, err = getService(cmd.Context(), token, "", serviceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete resource
	err = dataaccess.DeleteResource(cmd.Context(), token, serviceID, resourceID, false)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted resource")
	return nil
}
