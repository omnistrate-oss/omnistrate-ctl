package customer

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const describeExample = `# Describe a customer portal user
omnistrate-ctl customer describe --user-id user-123`

var describeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a customer portal user",
	Long:         "This command describes a customer portal user.",
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Flags().String("user-id", "", "Customer user ID")
	_ = describeCmd.MarkFlagRequired("user-id")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, _ := cmd.Flags().GetString("output")
	userID, _ := cmd.Flags().GetString("user-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	result, err := dataaccess.DescribeCustomerUser(cmd.Context(), token, userID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = utils.PrintTextTableJsonOutput(output, formatDescribeUser(result)); err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
