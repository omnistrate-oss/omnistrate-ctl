package account

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const customerUpdateExample = `# Update the backing account name and description for a customer BYOA onboarding instance
omnistrate-ctl account customer update instance-abcd1234 --name=my-customer-account --description="customer hosted account"

# Replace Nebius bindings on the backing account
omnistrate-ctl account customer update instance-abcd1234 --nebius-bindings-file=./nebius-bindings.yaml`

var customerUpdateCmd = &cobra.Command{
	Use:          "update [customer-account-instance-id] [flags]",
	Short:        "Update a customer BYOA account onboarding instance",
	Long:         "This command updates mutable fields on the backing account config associated with a customer BYOA onboarding instance.",
	Example:      customerUpdateExample,
	RunE:         runCustomerUpdate,
	SilenceUsage: true,
}

func init() {
	customerUpdateCmd.Args = cobra.ExactArgs(1)

	customerUpdateCmd.Flags().String("name", "", "Updated backing account name")
	customerUpdateCmd.Flags().String("description", "", "Updated backing account description")
	customerUpdateCmd.Flags().String("nebius-bindings-file", "", "Path to a YAML file describing the full replacement Nebius bindings")
	customerUpdateCmd.Flags().Bool("skip-wait", false, "Skip waiting for the backing account to become READY after replacing Nebius bindings")
	_ = customerUpdateCmd.MarkFlagFilename("nebius-bindings-file")
}

func runCustomerUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, _ := cmd.Flags().GetString("output")
	if output != "json" && output != "table" && output != "text" {
		err := errors.New("unsupported output format")
		utils.PrintError(err)
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	nebiusBindingsFile, _ := cmd.Flags().GetString("nebius-bindings-file")
	skipWait, _ := cmd.Flags().GetBool("skip-wait")

	params, err := buildUpdateAccountParams(cmd, args[0], name, description, nebiusBindingsFile)
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
		spinner = sm.AddSpinner("Resolving customer BYOA account onboarding instance...")
		sm.Start()
	}

	ref, err := resolveCustomerAccountInstanceByID(cmd.Context(), token, args[0])
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	describeResult, err := describeCustomerAccountByInstanceRef(cmd.Context(), token, ref)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	if describeResult.Summary.AccountConfigID == "" {
		err = errors.New("backing account config is not available for this customer onboarding instance yet")
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if len(params.NebiusBindings) > 0 {
		if describeResult.Account == nil || describeResult.Account.NebiusTenantID == nil {
			err = errors.New("Nebius bindings can only be updated for Nebius-backed customer accounts")
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	if output != "json" {
		utils.HandleSpinnerSuccess(spinner, sm, "Resolved customer BYOA account onboarding instance")
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Updating backing account config...")
		sm.Start()
	}

	accountConfigID, err := updateAccountFn(cmd.Context(), token, dataaccess.UpdateAccountParams{
		AccountConfigID: describeResult.Summary.AccountConfigID,
		Name:            params.Name,
		Description:     params.Description,
		NebiusBindings:  params.NebiusBindings,
	})
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully updated backing account config")

	if len(params.NebiusBindings) > 0 && !skipWait {
		var waitSpinner *utils.Spinner
		if output != "json" {
			fmt.Printf("\n")
			sm = utils.NewSpinnerManager()
			waitSpinner = sm.AddSpinner("Waiting for the backing account to become READY...")
			sm.Start()
		}

		if err = WaitForAccountReady(cmd.Context(), token, accountConfigID); err != nil {
			utils.HandleSpinnerError(waitSpinner, sm, err)
			utils.PrintError(fmt.Errorf("backing account did not become READY: %v", err))
			return err
		}

		utils.HandleSpinnerSuccess(waitSpinner, sm, "Backing account is now READY")
	}

	updatedAccount, err := describeAccountFn(cmd.Context(), token, accountConfigID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	updatedSummary := buildCustomerAccountSummary(ref, accountConfigID, updatedAccount)

	if output == "json" {
		result := customerAccountDescribeResult{
			Summary:  updatedSummary,
			Instance: describeResult.Instance,
			Account:  updatedAccount,
		}
		if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	if err = utils.PrintTextTableJsonOutput(output, updatedSummary); err != nil {
		utils.PrintError(err)
		return err
	}

	if updatedAccount.Status != "READY" {
		dataaccess.PrintNextStepVerifyAccountMsg(updatedAccount)
	}

	return nil
}
