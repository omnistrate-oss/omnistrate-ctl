package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const verifyExample = `# Send a verification email to a customer portal user
omnistrate-ctl customer verify --user-id user-123`

var verifyCmd = &cobra.Command{
	Use:          "verify [flags]",
	Short:        "Send a customer portal user verification email",
	Long:         "This command sends a verification email to a customer portal user.",
	Example:      verifyExample,
	RunE:         runVerify,
	SilenceUsage: true,
}

func init() {
	verifyCmd.Flags().String("user-id", "", "Customer user ID")
	_ = verifyCmd.MarkFlagRequired("user-id")
}

func runVerify(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	userID, _ := cmd.Flags().GetString("user-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = dataaccess.SendCustomerUserVerificationEmail(cmd.Context(), token, userID); err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully sent verification email to customer user %s\n", userID)
	return nil
}
