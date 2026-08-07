package service

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const resourceDeleteExample = `# Delete a resource with service and resource IDs
omnistrate-ctl service resource delete --service-id=[service-ID] --id=[resource-ID]

# Delete a resource using the service ID as a path segment
omnistrate-ctl service [service-ID] resource delete --id=[resource-ID]

# Dry run resource deletion
omnistrate-ctl service [service-ID] resource delete --id=[resource-ID] --dry-run`

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
		Use:          "delete --service-id [service-ID] --id [resource-ID] [flags]",
		Short:        "Delete a service resource",
		Long:         `This command helps you delete a resource from a service.`,
		Example:      resourceDeleteExample,
		Args:         cobra.NoArgs,
		RunE:         runResourceDelete,
		SilenceUsage: true,
	}

	resourceDeleteCmd.Flags().String("service-id", "", "Service ID")
	resourceDeleteCmd.Flags().String("id", "", "Resource ID")
	resourceDeleteCmd.Flags().Bool("dry-run", false, "Validate resource deletion without deleting it")

	return resourceDeleteCmd
}

func runResourceDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	resourceID, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return deleteResource(cmd.Context(), serviceID, resourceID, dryRun, output)
}

func runServiceResourceDeletePath(cmd *cobra.Command, args []string) (bool, error) {
	normalizedArgs, rewrote := normalizeServiceResourceDeleteArgs(args)
	if !rewrote {
		return false, nil
	}

	if hasHelpFlag(args) {
		return true, runResourceDeleteHelp(cmd)
	}

	serviceID, resourceID, dryRun, output, err := parseServiceResourceDeletePath(normalizedArgs)
	if err != nil {
		utils.PrintError(err)
		return true, err
	}

	if output == "" {
		output, _ = cmd.Flags().GetString("output")
	}

	return true, deleteResource(cmd.Context(), serviceID, resourceID, dryRun, output)
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func runResourceDeleteHelp(cmd *cobra.Command) error {
	resourceDeleteCmd, _, err := cmd.Find([]string{"resource", "delete"})
	if err != nil {
		return err
	}
	return resourceDeleteCmd.Help()
}

func normalizeServiceResourceDeleteArgs(args []string) ([]string, bool) {
	if len(args) < 3 || args[1] != "resource" || args[2] != "delete" {
		return args, false
	}

	normalized := make([]string, 0, len(args)+2)
	normalized = append(normalized, "resource", "delete", "--service-id", args[0])
	normalized = append(normalized, args[3:]...)

	return normalized, true
}

func parseServiceResourceDeletePath(args []string) (serviceID string, resourceID string, dryRun bool, output string, err error) {
	if len(args) < 4 || args[0] != "resource" || args[1] != "delete" || args[2] != "--service-id" {
		err = errors.New("invalid service resource delete arguments")
		return
	}

	serviceID = args[3]
	flags := pflag.NewFlagSet("service resource delete", pflag.ContinueOnError)
	flags.String("service-id", serviceID, "Service ID")
	flags.String("id", "", "Resource ID")
	flags.Bool("dry-run", false, "Validate resource deletion without deleting it")
	flags.StringP("output", "o", "", "Output format (text|table|json)")

	err = flags.Parse(args[4:])
	if err != nil {
		return
	}

	serviceID, err = flags.GetString("service-id")
	if err != nil {
		return
	}
	resourceID, err = flags.GetString("id")
	if err != nil {
		return
	}
	dryRun, err = flags.GetBool("dry-run")
	if err != nil {
		return
	}
	output, err = flags.GetString("output")
	if err != nil {
		return
	}

	err = validateResourceDeleteArguments(serviceID, resourceID)
	return
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

func deleteResource(ctx context.Context, serviceID, resourceID string, dryRun bool, output string) error {
	err := validateResourceDeleteArguments(serviceID, resourceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Deleting resource...")
		sm.Start()
	}

	err = dataaccess.DeleteResource(ctx, token, serviceID, resourceID, dryRun)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted resource")
	return nil
}
