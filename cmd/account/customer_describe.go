package account

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const customerDescribeExample = `# Describe a customer BYOA account onboarding instance
omnistrate-ctl account customer describe instance-abcd1234

# Get full raw details including the backing account config
omnistrate-ctl account customer describe instance-abcd1234 -o json`

var customerDescribeCmd = &cobra.Command{
	Use:          "describe [customer-account-instance-id] [flags]",
	Short:        "Describe a customer BYOA account onboarding instance",
	Long:         "This command describes a customer BYOA account onboarding instance and its backing account config.",
	Example:      customerDescribeExample,
	RunE:         runCustomerDescribe,
	SilenceUsage: true,
}

func init() {
	customerDescribeCmd.Args = cobra.ExactArgs(1)
	customerDescribeCmd.Flags().StringP("output", "o", "table", "Output format (text|table|json).")
}

func runCustomerDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, _ := cmd.Flags().GetString("output")
	if output != "json" && output != "table" && output != "text" {
		err := errors.New("unsupported output format")
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
		spinner = sm.AddSpinner("Fetching customer BYOA account onboarding instance...")
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

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved customer BYOA account onboarding details")

	if output == "json" {
		if err = utils.PrintTextTableJsonOutput(output, describeResult); err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	if err = utils.PrintTextTableJsonOutput(output, describeResult.Summary); err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
