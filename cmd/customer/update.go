package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const updateExample = `# Update attributes on a customer portal user
omnistrate-ctl customer update --user-id user-123 --attribute plan=enterprise --attribute region=us-west-2`

var updateCmd = &cobra.Command{
	Use:          "update [flags]",
	Short:        "Update a customer portal user",
	Long:         "This command updates customer portal user attributes.",
	Example:      updateExample,
	RunE:         runUpdate,
	SilenceUsage: true,
}

func init() {
	updateCmd.Flags().String("user-id", "", "Customer user ID")
	updateCmd.Flags().StringArray("attribute", []string{}, "Customer user attribute in key=value format. Can be repeated or comma-separated")
	_ = updateCmd.MarkFlagRequired("user-id")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	userID, _ := cmd.Flags().GetString("user-id")
	attributeValues, _ := cmd.Flags().GetStringArray("attribute")

	attributes, err := parseAttributes(attributeValues)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if len(attributes) == 0 {
		err = fmt.Errorf("at least one --attribute is required")
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = dataaccess.UpdateCustomerUser(cmd.Context(), token, userID, dataaccess.CustomerUserUpdateRequest{Attributes: attributes}); err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully updated customer user %s\n", userID)
	return nil
}
