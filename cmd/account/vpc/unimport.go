package vpc

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	unimportExample = `# Unimport a VPC (revert to AVAILABLE)
omnistrate-ctl account vpc unimport [account-id] --vpc-id=[vpc-id]`
)

var unimportCmd = &cobra.Command{
	Use:          "unimport [account-id] --vpc-id=[vpc-id]",
	Short:        "Unimport a VPC (revert to AVAILABLE)",
	Long:         `Reverts a previously imported VPC from READY back to AVAILABLE status, removing it from the deployment target pool.`,
	Example:      unimportExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runUnimport,
	SilenceUsage: true,
}

func init() {
	unimportCmd.Flags().String("vpc-id", "", "The VPC ID to unimport (required)")
	_ = unimportCmd.MarkFlagRequired("vpc-id")
}

func runUnimport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	vpcID, _ := cmd.Flags().GetString("vpc-id")
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
		spinner = sm.AddSpinner("Unimporting VPC...")
		sm.Start()
	}

	result, err := dataaccess.UnimportAccountConfigVPC(cmd.Context(), token, accountID, vpcID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "VPC unimported successfully")

	return printVPCOutput(output, result)
}
