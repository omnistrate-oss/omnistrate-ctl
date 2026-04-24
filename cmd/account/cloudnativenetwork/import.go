package cloudnativenetwork

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	importExample = `# Import a cloud-native network to make it available for deployments
omnistrate-ctl account cloud-native-network import [account-id] --network-id=[network-id]`
)

var importCmd = &cobra.Command{
	Use:          "import [account-id] --network-id=[network-id]",
	Short:        "Import an AVAILABLE cloud-native network for deployments",
	Long:         `Imports a discovered cloud-native network, changing its status from AVAILABLE to READY so it can be used for service deployments.`,
	Example:      importExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runImport,
	SilenceUsage: true,
}

func init() {
	importCmd.Flags().String("network-id", "", "The cloud-native network ID to import (required)")
	_ = importCmd.MarkFlagRequired("network-id")
}

func runImport(cmd *cobra.Command, args []string) error {
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
		spinner = sm.AddSpinner("Importing cloud-native network...")
		sm.Start()
	}

	result, err := dataaccess.ImportAccountConfigCloudNativeNetwork(cmd.Context(), token, accountID, networkID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native network imported successfully")

	return printCloudNativeNetworkOutput(output, result)
}
