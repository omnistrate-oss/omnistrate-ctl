package vpc

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
	bulkImportExample = `# Bulk import multiple VPCs
omnistrate-ctl account vpc bulk-import [account-id] --vpc-ids=vpc-abc123,vpc-def456`
)

var bulkImportCmd = &cobra.Command{
	Use:          "bulk-import [account-id] --vpc-ids=[vpc-id1,vpc-id2,...]",
	Short:        "Import multiple VPCs at once",
	Long:         `Imports multiple discovered VPCs in a single operation, changing their status from AVAILABLE to READY.`,
	Example:      bulkImportExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runBulkImport,
	SilenceUsage: true,
}

func init() {
	bulkImportCmd.Flags().String("vpc-ids", "", "Comma-separated list of VPC IDs to import (required)")
	_ = bulkImportCmd.MarkFlagRequired("vpc-ids")
}

func runBulkImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	vpcIDsRaw, _ := cmd.Flags().GetString("vpc-ids")
	output, _ := cmd.Flags().GetString("output")

	vpcIDs := parseVPCIDs(vpcIDsRaw)
	if len(vpcIDs) == 0 {
		return errors.New("at least one VPC ID must be provided")
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
		spinner = sm.AddSpinner(fmt.Sprintf("Importing %d VPC(s)...", len(vpcIDs)))
		sm.Start()
	}

	result, err := dataaccess.BulkImportAccountConfigVPCs(cmd.Context(), token, accountID, vpcIDs)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Imported %d VPC(s)", len(vpcIDs)))

	return printVPCOutput(output, result)
}

func parseVPCIDs(raw string) []string {
	var ids []string
	for _, id := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}
