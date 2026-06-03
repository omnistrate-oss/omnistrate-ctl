package cloudnativenetwork

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const importExample = `# Import cloud-native networks for deployments
omnistrate-ctl account customer cloud-native-network import [account-id] --network-id=vpc-abc123 --network-id=vpc-def456`

var importCmd = &cobra.Command{
	Use:          "import [account-id] --network-id=[network-id]",
	Short:        "Import cloud-native networks for deployments",
	Long:         `Marks discovered cloud-native networks as READY so they can be used as deployment targets.`,
	Example:      importExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runImport,
	SilenceUsage: true,
}

func init() {
	importCmd.Flags().StringSlice("network-id", nil, "Cloud-native network ID to import (repeatable)")
	_ = importCmd.MarkFlagRequired("network-id")
}

func runImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	networkIDs, _ := cmd.Flags().GetStringSlice("network-id")
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
		spinner = sm.AddSpinner("Importing cloud-native network...")
		sm.Start()
	}

	result, err := dataaccess.BulkImportAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID, networkIDs)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native network imported successfully")
	return printCloudNativeNetworkOutput(output, result)
}
