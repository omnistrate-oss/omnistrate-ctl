package vpc

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	importExample = `# Import a VPC to make it available for deployments
omnistrate-ctl account vpc import [account-id] --vpc-id=[vpc-id]`
)

var importCmd = &cobra.Command{
	Use:          "import [account-id] --vpc-id=[vpc-id]",
	Short:        "Import an AVAILABLE VPC for deployments",
	Long:         `Imports a discovered VPC, changing its status from AVAILABLE to READY so it can be used for service deployments.`,
	Example:      importExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runImport,
	SilenceUsage: true,
}

func init() {
	importCmd.Flags().String("vpc-id", "", "The VPC ID to import (required)")
	_ = importCmd.MarkFlagRequired("vpc-id")
}

func runImport(cmd *cobra.Command, args []string) error {
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
		spinner = sm.AddSpinner("Importing VPC...")
		sm.Start()
	}

	result, err := dataaccess.ImportAccountConfigVPC(cmd.Context(), token, accountID, vpcID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "VPC imported successfully")

	return printVPCOutput(output, result)
}
