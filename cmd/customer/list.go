package customer

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const listExample = `# List customer portal users
omnistrate-ctl customer list

# List customer portal users as JSON
omnistrate-ctl customer list --output json`

var listCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List customer portal users",
	Long:         "This command lists customer portal users.",
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().String("next-page-token", "", "Token for the next page of results")
	listCmd.Flags().Int64("page-size", 0, "Number of users to return per page")
	listCmd.Flags().Bool("exclude-stats", false, "Exclude user statistics from the response")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, _ := cmd.Flags().GetString("output")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	excludeStats, _ := cmd.Flags().GetBool("exclude-stats")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing customer users...")
		sm.Start()
	}

	result, err := dataaccess.ListCustomerUsers(cmd.Context(), token, dataaccess.CustomerUserListOptions{
		NextPageToken: nextPageToken,
		PageSize:      pageSize,
		ExcludeStats:  excludeStats,
	})
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	users := make([]model.CustomerUser, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, formatUser(user))
	}

	if len(users) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No customer users found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Found %d customer user(s)", len(users)))
	}

	if err = utils.PrintTextTableJsonArrayOutput(output, users); err != nil {
		utils.PrintError(err)
		return err
	}

	if result.NextPageToken != nil && *result.NextPageToken != "" && output != "json" {
		fmt.Printf("Next page token: %s\n", *result.NextPageToken)
	}

	return nil
}
