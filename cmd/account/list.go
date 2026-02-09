package account

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List accounts
omnistrate-ctl account list`
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List Cloud Provider Accounts",
	Long: `This command helps you list Cloud Provider Accounts.
You can filter for specific accounts by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of accounts. E.g.: key1:value1,key2:value2, which filters accounts where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Account{}), ",")+". Check the examples for more details.")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")

	// Parse and validate filters
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Account{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Retrieving accounts..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Retrieve accounts and accounts
	listRes, err := dataaccess.ListAccounts(cmd.Context(), token, "all")
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	var formattedAccounts []model.Account

	// Process and filter accounts
	for _, account := range listRes.AccountConfigs {
		formattedAccount, err := formatAccount(&account)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		match, err := utils.MatchesFilters(formattedAccount, filterMaps)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if match {
			formattedAccounts = append(formattedAccounts, formattedAccount)
		}
	}

	// Handle case when no accounts match
	if len(formattedAccounts) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No accounts found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved accounts")
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedAccounts)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Ask user to verify account if output is not JSON
	if output != "json" {
		dataaccess.AskVerifyAccountIfAny(cmd.Context())
	}

	return nil
}

// Helper functions

func formatAccount(account *openapiclient.DescribeAccountConfigResult) (model.Account, error) {
	if account == nil {
		return model.Account{}, fmt.Errorf("account is nil")
	}

	var targetAccountID, cloudProvider string

	// Handle AWS account
	if account.AwsAccountID != nil {
		targetAccountID = *account.AwsAccountID
		cloudProvider = "AWS"
	} else if account.GcpProjectID != nil && account.GcpProjectNumber != nil {
		// Handle GCP account
		targetAccountID = fmt.Sprintf("%s(ProjectID: %s)", *account.GcpProjectID, *account.GcpProjectNumber)
		cloudProvider = "GCP"
	} else if account.AzureSubscriptionID != nil {
		// Handle Azure account
		targetAccountID = *account.AzureSubscriptionID
		cloudProvider = "Azure"
	} else if account.OciTenancyID != nil {
		// Handle OCI account
		targetAccountID = *account.OciTenancyID
		cloudProvider = "OCI"
	} else {
		// Handle unknown account type
		targetAccountID = "unknown"
		cloudProvider = "unknown"
	}

	return model.Account{
		ID:              account.Id,
		Name:            account.Name,
		Status:          account.Status,
		CloudProvider:   cloudProvider,
		TargetAccountID: targetAccountID,
	}, nil
}
