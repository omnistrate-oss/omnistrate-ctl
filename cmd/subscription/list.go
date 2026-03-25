package subscription

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List subscriptions of the service postgres and mysql in the prod environment
omnistrate-ctl subscription list -f="service_name:postgres,environment:prod" -f="service:mysql,environment:prod"`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List Customer Subscriptions to your services",
	Long: `This command helps you list Customer Subscriptions to your services.
You can filter for specific subscriptions by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of subscriptions. E.g.: key1:value1,key2:value2, which filters subscriptions where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Subscription{}), ",")+". Check the examples for more details.")
	listCmd.Flags().Bool("truncate", false, "Truncate long names in the output")
	listCmd.Flags().BoolP("interactive", "i", false, "Launch interactive list with fuzzy search and selection")
}

func runList(cmd *cobra.Command, args []string) error {
	// Get flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	filters, err := cmd.Flags().GetStringArray("filter")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	truncateNames, err := cmd.Flags().GetBool("truncate")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Parse filters into a map
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Subscription{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON and not interactive
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" && !interactive {
		sm = utils.NewSpinnerManager()
		msg := "Retrieving subscriptions..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Get all subscriptions
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, "subscription:s")
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	formattedSubscriptions := make([]model.Subscription, 0)
	for i := range searchRes.SubscriptionResults {
		subscription := searchRes.SubscriptionResults[i]
		if subscription.Id == "" {
			continue
		}

		formattedSubscription := formatSubscription(&subscription, truncateNames)

		// Check if the subscription matches the filters
		ok, err := utils.MatchesFilters(formattedSubscription, filterMaps)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if ok {
			formattedSubscriptions = append(formattedSubscriptions, formattedSubscription)
		}
	}

	if len(formattedSubscriptions) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No subscriptions found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved subscriptions")
	}

	// Interactive mode
	if interactive {
		return runInteractiveSubscriptionList(formattedSubscriptions)
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, formattedSubscriptions)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func runInteractiveSubscriptionList(subscriptions []model.Subscription) error {
	items := make([]utils.InteractiveListItem, len(subscriptions))
	for i, sub := range subscriptions {
		rawJSON, _ := json.Marshal(sub)
		items[i] = utils.NewInteractiveListItem(
			sub.SubscriptionOwnerName,
			fmt.Sprintf("%s · %s · %s · %s", sub.SubscriptionID, sub.ServiceName, sub.PlanName, sub.Environment),
			sub.SubscriptionID,
			sub.Status,
			rawJSON,
		)
	}

	selected, err := utils.RunInteractiveList(utils.InteractiveListConfig{
		Title:    "Subscriptions",
		Items:    items,
		ShowJSON: true,
	})
	if err != nil {
		return err
	}

	if selected != nil {
		var prettyJSON json.RawMessage
		if err := json.Unmarshal([]byte(selected.JSONData()), &prettyJSON); err == nil {
			data, _ := json.MarshalIndent(prettyJSON, "", "    ")
			fmt.Println(string(data))
		}
	}

	return nil
}
