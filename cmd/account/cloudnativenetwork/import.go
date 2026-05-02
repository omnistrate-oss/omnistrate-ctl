package cloudnativenetwork

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	importExample = `# Import a single cloud-native network
omnistrate-ctl account cloud-native-network import [account-id] --network-id=[network-id]

# Import multiple cloud-native networks at once
omnistrate-ctl account cloud-native-network import [account-id] --network-ids=vpc-abc123,vpc-def456`
)

var importCmd = &cobra.Command{
	Use:          "import [account-id]",
	Short:        "Import one or more AVAILABLE cloud-native networks for deployments",
	Long:         `Imports discovered cloud-native networks, changing their status from AVAILABLE to READY so they can be used for service deployments. Use --network-id for a single network or --network-ids for bulk import.`,
	Example:      importExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runImport,
	SilenceUsage: true,
}

func init() {
	importCmd.Flags().String("network-id", "", "The cloud-native network ID to import")
	importCmd.Flags().String("network-ids", "", "Comma-separated list of cloud-native network IDs to import")
	importCmd.MarkFlagsMutuallyExclusive("network-id", "network-ids")
}

func runImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	networkID, _ := cmd.Flags().GetString("network-id")
	networkIDsRaw, _ := cmd.Flags().GetString("network-ids")
	output, _ := cmd.Flags().GetString("output")

	// Determine network IDs from flags.
	var networkIDs []string
	if networkID != "" {
		networkIDs = []string{networkID}
	} else if networkIDsRaw != "" {
		networkIDs = parseNetworkIDs(networkIDsRaw)
	}
	if len(networkIDs) == 0 {
		return errors.New("either --network-id or --network-ids must be provided")
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
		spinner = sm.AddSpinner(fmt.Sprintf("Importing %d cloud-native network(s)...", len(networkIDs)))
		sm.Start()
	}

	var result *dataaccess.CloudNativeNetworkResult
	if len(networkIDs) == 1 {
		result, err = dataaccess.ImportAccountConfigCloudNativeNetwork(cmd.Context(), token, accountID, networkIDs[0])
	} else {
		result, err = dataaccess.BulkImportAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID, networkIDs)
	}
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Imported %d cloud-native network(s)", len(networkIDs)))

	return printCloudNativeNetworkOutput(output, result)
}

func parseNetworkIDs(raw string) []string {
	var ids []string
	for _, id := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}
