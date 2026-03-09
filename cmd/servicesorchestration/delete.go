package servicesorchestration

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deleteExample = `# Delete an services orchestration deployment
omnistrate-ctl services-orchestration delete so-abcd1234`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [services-orchestration-id] [flags]",
	Short:        "Delete a services orchestration deployment",
	Long:         `This command helps you delete a services orchestration deployment from your account.`,
	Deprecated:   "This command is deprecated and will be removed in a future release.",
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().BoolP("yes", "y", false, "Pre-approve the deletion of the services orchestration deployment without prompting for confirmation")
	deleteCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("services orchestration id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	soID := args[0]

	// Retrieve flags
	output, _ := cmd.Flags().GetString("output")
	yes, _ := cmd.Flags().GetBool("yes")

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Confirm deletion
	if !yes {
		confirmed, err := utils.ConfirmAction("Are you sure you want to delete this services orchestration deployment?")
		if err != nil {
			utils.PrintError(err)
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Initialize spinner if output is not JSON
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Deleting instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if services orchestration exists
	_, err = dataaccess.DescribeServicesOrchestration(
		cmd.Context(),
		token,
		soID,
	)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Delete the services orchestration
	err = dataaccess.DeleteServicesOrchestration(cmd.Context(), token, soID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted services orchestration deployment")

	return nil
}
