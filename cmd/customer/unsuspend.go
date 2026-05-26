package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const unsuspendExample = `# Unsuspend a customer portal user
omnistrate-ctl customer unsuspend --user-id user-123`

var unsuspendCmd = &cobra.Command{
	Use:          "unsuspend [flags]",
	Short:        "Unsuspend a customer portal user",
	Long:         "This command unsuspends a customer portal user.",
	Example:      unsuspendExample,
	RunE:         runUnsuspend,
	SilenceUsage: true,
}

func init() {
	unsuspendCmd.Flags().String("user-id", "", "Customer user ID")
	_ = unsuspendCmd.MarkFlagRequired("user-id")
}

func runUnsuspend(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	userID, _ := cmd.Flags().GetString("user-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = dataaccess.UnsuspendCustomerUser(cmd.Context(), token, userID); err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully unsuspended customer user %s\n", userID)
	return nil
}
