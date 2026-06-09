package cloudnativenetwork

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

func newRemoveCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove [account-id] --region=[region] --network-id=[network-id]",
		Short:        "Remove an imported cloud-native network (revert to AVAILABLE)",
		Long:         `Reverts a previously imported cloud-native network from READY back to AVAILABLE status, removing it from the deployment target pool.`,
		Example:      fmt.Sprintf("# Remove a cloud-native network (revert to AVAILABLE)\nomnistrate-ctl %s remove [account-id] --region=[region] --network-id=[network-id]", commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runRemove,
		SilenceUsage: true,
	}

	cmd.Flags().String("region", "", "The cloud provider region of the cloud-native network to remove (required)")
	cmd.Flags().String("network-id", "", "The cloud-native network ID to remove (required)")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("network-id")
	return cmd
}

func runRemove(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	region, _ := cmd.Flags().GetString("region")
	networkID, _ := cmd.Flags().GetString("network-id")
	output, _ := cmd.Flags().GetString("output")

	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if networkID == "" {
		return fmt.Errorf("network-id cannot be empty")
	}

	targets := []dataaccess.CloudNativeNetworkTarget{{
		Region:    region,
		NetworkID: networkID,
	}}

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

	result, err := dataaccess.BulkUnimportAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID, targets)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native network removed successfully")

	return printCloudNativeNetworkOutput(output, result)
}
