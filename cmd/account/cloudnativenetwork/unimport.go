package cloudnativenetwork

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	removeExample = `# Remove a cloud-native network (revert to AVAILABLE)
omnistrate-ctl account cloud-native-network remove [account-id] --network-id=[network-id]`
)

var removeCmd = &cobra.Command{
	Use:          "remove [account-id] --network-id=[network-id]",
	Short:        "Remove an imported cloud-native network (revert to AVAILABLE)",
	Long:         `Reverts a previously imported cloud-native network from READY back to AVAILABLE status, removing it from the deployment target pool.`,
	Example:      removeExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runRemove,
	SilenceUsage: true,
}

func init() {
	removeCmd.Flags().String("network-id", "", "The cloud-native network ID to remove (required)")
	_ = removeCmd.MarkFlagRequired("network-id")
}

func runRemove(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	networkID, _ := cmd.Flags().GetString("network-id")
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
		spinner = sm.AddSpinner("Removing cloud-native network...")
		sm.Start()
	}

	result, err := dataaccess.UnimportAccountConfigCloudNativeNetwork(cmd.Context(), token, accountID, networkID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native network removed successfully")

	return printCloudNativeNetworkOutput(output, result)
}
