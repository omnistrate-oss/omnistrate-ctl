package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const suspendExample = `# Suspend a customer portal user
omnistrate-ctl customer suspend --user-id user-123`

var suspendCmd = &cobra.Command{
	Use:          "suspend [flags]",
	Short:        "Suspend a customer portal user",
	Long:         "This command suspends a customer portal user.",
	Example:      suspendExample,
	RunE:         runSuspend,
	SilenceUsage: true,
}

func init() {
	suspendCmd.Flags().String("user-id", "", "Customer user ID")
	_ = suspendCmd.MarkFlagRequired("user-id")
}

func runSuspend(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	userID, _ := cmd.Flags().GetString("user-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = dataaccess.SuspendCustomerUser(cmd.Context(), token, userID); err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully suspended customer user %s\n", userID)
	return nil
}
