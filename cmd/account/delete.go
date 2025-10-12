package account

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deleteExample = `# Delete account with name or id
omctl account delete [account-name or account-id]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [account-name or account-id] [flags]",
	Short:        "Delete a Cloud Provider Account",
	Long:         "This command helps you delete a cloud provider account.",
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Args = cobra.MaximumNArgs(1) // Require at most 1 argument
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	var nameOrID string
	if len(args) > 0 {
		nameOrID = args[0]
	}

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate input args
	err = validateDeleteArguments(args)
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
		msg := "Deleting account..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if account exists
	var id string
	id, err = getAccountID(cmd.Context(), token, nameOrID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete account
	err = dataaccess.DeleteAccount(cmd.Context(), token, id)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted account")

	return nil
}

// Helper functions

func validateDeleteArguments(args []string) error {
	if len(args) == 0 {
		return errors.New("account name or ID must be provided")
	}

	return nil
}
