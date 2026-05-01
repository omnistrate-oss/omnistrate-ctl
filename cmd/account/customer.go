package account

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/account/cloudnativenetwork"
	"github.com/spf13/cobra"
)

var customerCmd = &cobra.Command{
	Use:          "customer [operation] [flags]",
	Short:        "Manage customer BYOA account onboarding",
	Long:         "This command helps you onboard customer cloud accounts into BYOA service plans.",
	Run:          runCustomer,
	SilenceUsage: true,
}

func init() {
	customerCmd.AddCommand(customerCreateCmd)
	customerCmd.AddCommand(customerUpdateCmd)
	customerCmd.AddCommand(customerDeleteCmd)
	customerCmd.AddCommand(customerListCmd)
	customerCmd.AddCommand(customerDescribeCmd)
	customerCmd.AddCommand(cloudnativenetwork.Cmd)
}

func runCustomer(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
