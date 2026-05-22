package account

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const customerDeleteExample = `# Delete a customer BYOA account onboarding instance
omnistrate-ctl account customer delete instance-abcd1234

# Delete and print the deleted instance summary
omnistrate-ctl account customer delete instance-abcd1234 -o json`

var customerDeleteCmd = &cobra.Command{
	Use:          "delete [customer-account-instance-id] [flags]",
	Short:        "Delete a customer BYOA account onboarding instance",
	Long:         "This command deletes the customer BYOA account onboarding instance for a service plan.",
	Example:      customerDeleteExample,
	RunE:         runCustomerDelete,
	SilenceUsage: true,
}

func init() {
	customerDeleteCmd.Args = cobra.ExactArgs(1)
}

func runCustomerDelete(cmd *cobra.Command, args []string) error {
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
		spinner = sm.AddSpinner("Resolving customer BYOA account onboarding instance...")
		sm.Start()
	}

	ref, err := resolveCustomerAccountInstanceByID(cmd.Context(), token, args[0])
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	instance, err := describeResourceInstanceFn(cmd.Context(), token, ref.ServiceID, ref.EnvironmentID, ref.InstanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	accountConfigID := extractCustomerAccountConfigID(instance)

	if output != "json" {
		utils.HandleSpinnerSuccess(spinner, sm, "Resolved customer BYOA account onboarding instance")
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Deleting customer BYOA account onboarding instance...")
		sm.Start()
	}

	if err = deleteResourceInstanceFn(cmd.Context(), token, ref.ServiceID, ref.EnvironmentID, ref.ResourceID, ref.InstanceID); err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted customer BYOA account onboarding instance")

	result := customerAccountDeleteResult{
		InstanceID:      ref.InstanceID,
		AccountConfigID: accountConfigID,
		Service:         ref.ServiceName,
		Environment:     ref.Environment,
		Plan:            ref.Plan,
		Resource:        ref.Resource,
		CloudProvider:   ref.CloudProvider,
		SubscriptionID:  ref.SubscriptionID,
		Deleted:         true,
	}

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
