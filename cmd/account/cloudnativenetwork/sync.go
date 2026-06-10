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

func newSyncCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [account-id]",
		Short: "Sync cloud-native networks from the cloud provider",
		Long:  `Discovers or re-validates cloud-native networks for a BYOA account configuration.`,
		Example: fmt.Sprintf(`# Sync all cloud-native networks for an account
omnistrate-ctl %s sync [account-id]

# Sync all networks in specific regions
omnistrate-ctl %s sync [account-id] --region=us-east-1 --region=us-west-2

# Sync specific networks
omnistrate-ctl %s sync [account-id] --region=us-east-1 --network-id=vpc-abc123

# Sync networks and include host clusters in discovery
omnistrate-ctl %s sync [account-id] --region=us-east-1 --include-host-clusters`, commandPath, commandPath, commandPath, commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runSync,
		SilenceUsage: true,
	}
	cmd.Flags().StringSlice("region", nil, "Cloud region to discover (repeatable)")
	cmd.Flags().StringSlice("network-id", nil, "Cloud-native network ID to sync in the specified region (repeatable)")
	cmd.Flags().StringSlice("network", nil, "Specific network to sync in region:network-id format (repeatable)")
	cmd.Flags().Bool("include-host-clusters", false, "Include host clusters when refreshing targeted cloud-native networks")
	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	output, _ := cmd.Flags().GetString("output")
	regions, _ := cmd.Flags().GetStringSlice("region")
	networkIDs, _ := cmd.Flags().GetStringSlice("network-id")
	networks, _ := cmd.Flags().GetStringSlice("network")
	includeHostClusters, _ := cmd.Flags().GetBool("include-host-clusters")

	targets, err := syncTargetsFromFlags(regions, networkIDs, networks)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if includeHostClusters && len(targets) == 0 {
		err := fmt.Errorf("include-host-clusters requires at least one sync target (--region, --network-id, or --network)")
		utils.PrintError(err)
		return err
	}
	if includeHostClusters {
		for i := range targets {
			targets[i].IncludeHostClusters = &includeHostClusters
		}
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

func syncTargetsFromFlags(regions, networkIDs, networks []string) ([]dataaccess.CloudNativeNetworkTarget, error) {
	targets := make([]dataaccess.CloudNativeNetworkTarget, 0, len(regions)+len(networkIDs)+len(networks))

	cleanRegions := make([]string, 0, len(regions))
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			return nil, fmt.Errorf("region cannot be empty")
		}
		cleanRegions = append(cleanRegions, region)
	}

	if len(networkIDs) > 0 {
		if len(cleanRegions) != 1 {
			return nil, fmt.Errorf("network-id requires exactly one region")
		}
		for _, networkID := range networkIDs {
			networkID = strings.TrimSpace(networkID)
			if networkID == "" {
				return nil, fmt.Errorf("network-id cannot be empty")
			}
			targets = append(targets, dataaccess.CloudNativeNetworkTarget{
				Region:    cleanRegions[0],
				NetworkID: networkID,
			})
		}
	} else {
		for _, region := range cleanRegions {
			targets = append(targets, dataaccess.CloudNativeNetworkTarget{Region: region})
		}
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
