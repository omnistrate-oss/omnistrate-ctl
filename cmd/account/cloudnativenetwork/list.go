package cloudnativenetwork

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List all cloud-native networks for an account
omnistrate-ctl account cloud-native-network list [account-id]`
)

var listCmd = &cobra.Command{
	Use:          "list [account-id]",
	Short:        "List cloud-native networks for a BYOA Cloud Provider Account",
	Long:         `Lists all cloud-native networks registered with a BYOA cloud provider account, including their status (AVAILABLE, READY, SYNCING, etc).`,
	Example:      listExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runList,
	SilenceUsage: true,
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

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Found %d cloud-native network(s)", len(result.CloudNativeNetworks)))

	return printCloudNativeNetworkOutput(output, result)
}
