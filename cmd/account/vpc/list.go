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
	listExample = `# List all VPCs for an account
omnistrate-ctl account vpc list [account-id]`
)

var listCmd = &cobra.Command{
	Use:          "list [account-id]",
	Short:        "List VPCs for a Cloud Provider Account",
	Long:         `Lists all VPCs registered with a BYOA cloud provider account, including their status (AVAILABLE, READY, SYNCING, etc).`,
	Example:      listExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runList,
	SilenceUsage: true,
}

func runList(cmd *cobra.Command, args []string) error {
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
		spinner = sm.AddSpinner("Listing VPCs...")
		sm.Start()
	}

	result, err := dataaccess.ListAccountConfigVPCs(cmd.Context(), token, accountID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Found %d VPC(s)", len(result.CloudNativeNetworks)))

	return printVPCOutput(output, result)
}
