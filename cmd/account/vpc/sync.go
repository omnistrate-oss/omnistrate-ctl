package vpc

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	syncExample = `# Sync VPCs for an account
omnistrate-ctl account vpc sync [account-id]`
)

var syncCmd = &cobra.Command{
	Use:          "sync [account-id]",
	Short:        "Discover and sync VPCs from the cloud provider",
	Long:         `Triggers VPC discovery for a BYOA cloud provider account. Discovered VPCs will appear with AVAILABLE status and can then be imported.`,
	Example:      syncExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runSync,
	SilenceUsage: true,
}

func runSync(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
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
		spinner = sm.AddSpinner("Syncing VPCs...")
		sm.Start()
	}

	result, err := dataaccess.SyncAccountConfigVPCs(cmd.Context(), token, accountID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Discovered %d VPC(s)", len(result.VPCs)))

	return printVPCOutput(output, result)
}
