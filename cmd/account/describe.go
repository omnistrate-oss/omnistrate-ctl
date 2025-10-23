package account

import (
	"context"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe account with name or id
omctl account describe [account-name or account-id]`
)

var describeCmd = &cobra.Command{
	Use:          "describe [account-name or account-id] [flags]",
	Short:        "Describe a Cloud Provider Account",
	Long:         "This command helps you get details of a cloud provider account.",
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.MaximumNArgs(1) // Require at most 1 argument

	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported.") // Override inherited flag
}

func runDescribe(cmd *cobra.Command, args []string) error {
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
	err = validateDescribeArguments(args, output)
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

	// Describe account
	account, err := dataaccess.DescribeAccount(cmd.Context(), token, id)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved account details")

	// Print output
	err = utils.PrintTextTableJsonOutput(output, account)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Ask user to verify account if output is not JSON
	if output != "json" {
		dataaccess.AskVerifyAccountIfAny(cmd.Context())
	}

	return nil
}

// Helper functions

func validateDescribeArguments(args []string, output string) error {
	if len(args) == 0 {
		return errors.New("account name or ID must be provided")
	}

	if output != "json" {
		return errors.New("only json output is supported")
	}

	return nil
}

func getAccountID(ctx context.Context, token, accountNameOrIDArg string) (accountID string, err error) {
	// List accounts
	listRes, err := dataaccess.ListAccounts(ctx, token, "all")
	if err != nil {
		return
	}

	count := 0
	for _, account := range listRes.AccountConfigs {
		// Check for exact match (case-insensitive) with name or ID
		if strings.EqualFold(account.Name, accountNameOrIDArg) {
			accountID = account.Id
			count++
		}
		if strings.EqualFold(account.Id, accountNameOrIDArg) {
			accountID = account.Id
			count++
		}
	}

	if count == 0 {
		err = errors.New("account not found")
		return
	}

	if count > 1 {
		err = errors.New("multiple accounts found with the same name. Please specify account ID")
		return
	}

	return
}
