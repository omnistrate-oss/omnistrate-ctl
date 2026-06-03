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

const syncExample = `# Sync all cloud-native networks for an account
omnistrate-ctl account customer cloud-native-network sync [account-id]

# Sync all networks in specific regions
omnistrate-ctl account customer cloud-native-network sync [account-id] --region=us-east-1 --region=us-west-2

# Sync specific networks
omnistrate-ctl account customer cloud-native-network sync [account-id] --network=us-east-1:vpc-abc123`

var syncCmd = &cobra.Command{
	Use:          "sync [account-id]",
	Short:        "Sync cloud-native networks from the cloud provider",
	Long:         `Discovers or re-validates cloud-native networks for a BYOA account configuration.`,
	Example:      syncExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runSync,
	SilenceUsage: true,
}

func init() {
	syncCmd.Flags().StringSlice("region", nil, "Cloud region to discover (repeatable)")
	syncCmd.Flags().StringSlice("network", nil, "Specific network to sync in region:network-id format (repeatable)")
}

func runSync(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	output, _ := cmd.Flags().GetString("output")
	regions, _ := cmd.Flags().GetStringSlice("region")
	networks, _ := cmd.Flags().GetStringSlice("network")

	targets, err := syncTargetsFromFlags(regions, networks)
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
		spinner = sm.AddSpinner("Syncing cloud-native networks...")
		sm.Start()
	}

	result, err := dataaccess.SyncAccountConfigCloudNativeNetworksByTarget(cmd.Context(), token, accountID, targets)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Cloud-native networks synced successfully")
	return printCloudNativeNetworkOutput(output, result)
}

func syncTargetsFromFlags(regions, networks []string) ([]dataaccess.CloudNativeNetworkTarget, error) {
	targets := make([]dataaccess.CloudNativeNetworkTarget, 0, len(regions)+len(networks))
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			return nil, fmt.Errorf("region cannot be empty")
		}
		targets = append(targets, dataaccess.CloudNativeNetworkTarget{Region: region})
	}

	for _, network := range networks {
		parts := strings.SplitN(strings.TrimSpace(network), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid --network value %q: expected region:network-id", network)
		}
		targets = append(targets, dataaccess.CloudNativeNetworkTarget{
			Region:    strings.TrimSpace(parts[0]),
			NetworkID: strings.TrimSpace(parts[1]),
		})
	}

	return targets, nil
}
