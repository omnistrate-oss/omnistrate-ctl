package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const deleteExample = `# Delete a customer portal user
omnistrate-ctl customer delete --user-id user-123`

var deleteCmd = &cobra.Command{
	Use:          "delete [flags]",
	Short:        "Delete a customer portal user",
	Long:         "This command deletes a customer portal user.",
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().String("user-id", "", "Customer user ID")
	_ = deleteCmd.MarkFlagRequired("user-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	userID, _ := cmd.Flags().GetString("user-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = dataaccess.DeleteCustomerUser(cmd.Context(), token, userID); err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully deleted customer user %s\n", userID)
	return nil
}
