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
	bulkImportExample = `# Bulk import multiple cloud-native networks
omnistrate-ctl account cloud-native-network bulk-import [account-id] --network-ids=cnn-abc123,cnn-def456`
)

var bulkImportCmd = &cobra.Command{
	Use:          "bulk-import [account-id] --network-ids=[network-id1,network-id2,...]",
	Short:        "Import multiple cloud-native networks at once",
	Long:         `Imports multiple discovered cloud-native networks in a single operation, changing their status from AVAILABLE to READY.`,
	Example:      bulkImportExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runBulkImport,
	SilenceUsage: true,
}

func init() {
	bulkImportCmd.Flags().String("network-ids", "", "Comma-separated list of cloud-native network IDs to import (required)")
	_ = bulkImportCmd.MarkFlagRequired("network-ids")
}

func runBulkImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	networkIDsRaw, _ := cmd.Flags().GetString("network-ids")
	output, _ := cmd.Flags().GetString("output")

	networkIDs := parseNetworkIDs(networkIDsRaw)
	if len(networkIDs) == 0 {
		return errors.New("at least one cloud-native network ID must be provided")
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

	result, err := dataaccess.BulkImportAccountConfigCloudNativeNetworks(cmd.Context(), token, accountID, networkIDs)
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
