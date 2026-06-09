package cloudnativenetwork

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

func newImportCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [account-id] --region=[region] --network-id=[network-id]",
		Short: "Import cloud-native networks for deployments",
		Long:  `Marks discovered cloud-native networks as READY so they can be used as deployment targets.`,
		Example: fmt.Sprintf(`# Import a cloud-native network for deployments
omnistrate-ctl %s import [account-id] --region=[region] --network-id=[network-id]

# Import multiple cloud-native networks in the same region
omnistrate-ctl %s import [account-id] --region=[region] --network-id=[network-id-1] --network-id=[network-id-2]`, commandPath, commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runImport,
		SilenceUsage: true,
	}
	cmd.Flags().String("region", "", "The cloud provider region of the cloud-native network to import (required)")
	cmd.Flags().StringSlice("network-id", nil, "Cloud-native network ID to import (repeatable, required)")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("network-id")
	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	region, _ := cmd.Flags().GetString("region")
	networkIDs, _ := cmd.Flags().GetStringSlice("network-id")
	output, _ := cmd.Flags().GetString("output")

	targets, err := importTargetsFromFlags(region, networkIDs)
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
		spinner = sm.AddSpinner("Importing cloud-native network...")
		sm.Start()
	}

	result, err := dataaccess.BulkImportAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID, targets)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native network imported successfully")
	return printCloudNativeNetworkOutput(output, result)
}

func importTargetsFromFlags(region string, networkIDs []string) ([]dataaccess.CloudNativeNetworkTarget, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	targets := make([]dataaccess.CloudNativeNetworkTarget, 0, len(networkIDs))
	for _, networkID := range networkIDs {
		networkID = strings.TrimSpace(networkID)
		if networkID == "" {
			return nil, fmt.Errorf("network-id cannot be empty")
		}
		targets = append(targets, dataaccess.CloudNativeNetworkTarget{
			Region:    region,
			NetworkID: networkID,
		})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one network-id is required")
	}

	return targets, nil
}
