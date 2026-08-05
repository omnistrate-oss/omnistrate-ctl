package build

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const deleteDeprecatedResourceExample = `# Validate that a deprecated resource can be deleted
omnistrate-ctl build delete-deprecated-resource --service-id s-123 --resource-id r-123 --dry-run

# Delete a deprecated resource
omnistrate-ctl build delete-deprecated-resource --service-id s-123 --resource-id r-123 --yes`

var deleteDeprecatedResourceCmd = &cobra.Command{
	Use:          "delete-deprecated-resource --service-id [service-id] --resource-id [resource-id] [flags]",
	Short:        "Delete a deprecated resource",
	Long:         `Delete an already deprecated resource from a service. This command does not deprecate resources; it only deletes resources that are ready for deletion.`,
	Example:      deleteDeprecatedResourceExample,
	RunE:         runDeleteDeprecatedResource,
	SilenceUsage: true,
}

func init() {
	deleteDeprecatedResourceCmd.Flags().String("service-id", "", "Service ID that owns the deprecated resource")
	deleteDeprecatedResourceCmd.Flags().String("resource-id", "", "Deprecated resource ID to delete")
	deleteDeprecatedResourceCmd.Flags().Bool("dry-run", false, "Validate the deprecated resource deletion without deleting it")
	deleteDeprecatedResourceCmd.Flags().BoolP("yes", "y", false, "Pre-approve deleting the deprecated resource without prompting for confirmation")

	_ = deleteDeprecatedResourceCmd.MarkFlagRequired("service-id")
	_ = deleteDeprecatedResourceCmd.MarkFlagRequired("resource-id")

	BuildCmd.AddCommand(deleteDeprecatedResourceCmd)
}

func runDeleteDeprecatedResource(cmd *cobra.Command, args []string) error {
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
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if !dryRun && !yes {
		confirmed, err := confirmDeleteDeprecatedResource(os.Stdin, os.Stdout, resourceID)
		if err != nil {
			utils.PrintError(err)
			return err
		}
		if !confirmed {
			return nil
		}
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Deleting deprecated resource..."
		if dryRun {
			msg = "Validating deprecated resource deletion..."
		}
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	if err := dataaccess.DeleteResource(cmd.Context(), token, serviceID, resourceID, dryRun); err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if dryRun {
		utils.HandleSpinnerSuccess(spinner, sm, "Deprecated resource deletion dry run succeeded")
		return nil
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted deprecated resource")
	return nil
}

func confirmDeleteDeprecatedResource(reader io.Reader, writer io.Writer, resourceID string) (bool, error) {
	if _, err := fmt.Fprintf(writer, "Are you sure you want to delete deprecated resource %s? (yes/no): ", resourceID); err != nil {
		return false, err
	}

	response, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(response)) {
	case "yes", "y":
		return true, nil
	default:
		return false, nil
	}
}
