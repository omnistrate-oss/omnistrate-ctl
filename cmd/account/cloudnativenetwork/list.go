package cloudnativenetwork

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

func newListCmd(commandPath string) *cobra.Command {
	return &cobra.Command{
		Use:          "list [account-id]",
		Short:        "List cloud-native networks for a BYOA account",
		Long:         `Lists the cloud-native networks registered under a BYOA account configuration, including import and in-use status.`,
		Example:      fmt.Sprintf("# List cloud-native networks registered under an account\nomnistrate-ctl %s list [account-id]", commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runList,
		SilenceUsage: true,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	output, _ := cmd.Flags().GetString("output")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing cloud-native networks...")
		sm.Start()
	}

	result, err := dataaccess.ListAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native networks listed successfully")
	return printCloudNativeNetworkOutput(output, result)
}
