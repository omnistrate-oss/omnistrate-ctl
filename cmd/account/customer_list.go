package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const customerListExample = `# List all customer BYOA account onboarding instances
omnistrate-ctl account customer list

# Filter by service and plan
omnistrate-ctl account customer list --service Nebius --plan "Nebius BYOA Compute Variants"

# Filter by cloud provider
omnistrate-ctl account customer list --cloud-provider nebius

# Filter by subscription or customer email
omnistrate-ctl account customer list --subscription-id sub-123456
omnistrate-ctl account customer list --service Nebius --plan "Nebius BYOA Compute Variants" --customer-email customer@example.com`

type customerAccountListFilters struct {
	CloudProvider  string
	Service        string
	Plan           string
	SubscriptionID string
	CustomerEmail  string
}

var (
	resolveCustomerListServiceFn      = resolveServiceByNameOrID
	resolveCustomerListPlanIDFn       = resolveServicePlanIDAcrossEnvironments
	describeServiceOfferingForListFn  = dataaccess.DescribeServiceOffering
	listSubscriptionsForEnvironmentFn = dataaccess.ListSubscriptions
)

var customerListCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List customer BYOA account onboarding instances",
	Long:         "This command lists customer BYOA account onboarding instances created through account customer create.",
	Example:      customerListExample,
	RunE:         runCustomerList,
	SilenceUsage: true,
}

func init() {
	customerListCmd.Flags().String("cloud-provider", "", "Filter by cloud provider")
	customerListCmd.Flags().String("service", "", "Filter by service name or ID")
	customerListCmd.Flags().String("plan", "", "Filter by service plan name or ID")
	customerListCmd.Flags().String("subscription-id", "", "Filter by subscription ID")
	customerListCmd.Flags().String(customerEmailFlag, "", "Filter by customer email; requires both --service and --plan")
}

func runCustomerList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, _ := cmd.Flags().GetString("output")
	filters, err := customerAccountListFiltersFromFlags(cmd)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	subscriptionIDs, err := resolveCustomerAccountListSubscriptionIDs(cmd, token, filters)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing customer BYOA account onboarding instances...")
		sm.Start()
	}

	searchResult, err := searchInventoryFn(cmd.Context(), token, customerAccountSearchQuery)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	items := make([]customerAccountListItem, 0)
	for _, record := range searchResult.ResourceInstanceResults {
		item, ok := customerAccountListItemFromSearchRecord(record)
		if !ok || !matchesCustomerAccountListFilters(record, filters, subscriptionIDs) {
			continue
		}
		items = append(items, *item)
	}

	if len(items) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No customer BYOA account onboarding instances found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Found %d customer BYOA account onboarding instance(s)", len(items)))
	}

	if err = utils.PrintTextTableJsonArrayOutput(output, items); err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func customerAccountListFiltersFromFlags(cmd *cobra.Command) (customerAccountListFilters, error) {
	cloudProvider, _ := cmd.Flags().GetString("cloud-provider")
	service, _ := cmd.Flags().GetString("service")
	plan, _ := cmd.Flags().GetString("plan")
	subscriptionID, _ := cmd.Flags().GetString("subscription-id")
	customerEmail, _ := cmd.Flags().GetString(customerEmailFlag)

	filters := customerAccountListFilters{
		CloudProvider:  strings.TrimSpace(cloudProvider),
		Service:        strings.TrimSpace(service),
		Plan:           strings.TrimSpace(plan),
		SubscriptionID: strings.TrimSpace(subscriptionID),
		CustomerEmail:  strings.TrimSpace(customerEmail),
	}

	return filters, validateCustomerAccountListFilters(filters)
}

func validateCustomerAccountListFilters(filters customerAccountListFilters) error {
	if filters.CustomerEmail != "" && filters.SubscriptionID != "" {
		return fmt.Errorf("cannot specify both --customer-email and --subscription-id")
	}

	if filters.CustomerEmail != "" {
		if err := utils.ValidateEmail(filters.CustomerEmail); err != nil {
			return fmt.Errorf("invalid --customer-email value: %w", err)
		}
		if filters.Service == "" || filters.Plan == "" {
			return fmt.Errorf("--customer-email requires both --service and --plan")
		}
	}

	return nil
}

func resolveCustomerAccountListSubscriptionIDs(
	cmd *cobra.Command,
	token string,
	filters customerAccountListFilters,
) (map[string]struct{}, error) {
	if filters.SubscriptionID != "" {
		return map[string]struct{}{filters.SubscriptionID: {}}, nil
	}

	if filters.CustomerEmail == "" {
		return nil, nil
	}

	serviceID, _, err := resolveCustomerListServiceFn(cmd.Context(), token, filters.Service)
	if err != nil {
		return nil, err
	}

	planID, err := resolveCustomerListPlanIDFn(cmd.Context(), token, serviceID, filters.Plan)
	if err != nil {
		return nil, err
	}

	offeringResult, err := describeServiceOfferingForListFn(cmd.Context(), token, serviceID, planID, "")
	if err != nil {
		return nil, err
	}
	if offeringResult == nil || offeringResult.ConsumptionDescribeServiceOfferingResult == nil {
		return nil, fmt.Errorf("service offering response is empty for %s / %s", filters.Service, filters.Plan)
	}

	subscriptionIDs := make(map[string]struct{})
	seenEnvironments := make(map[string]struct{})

	for _, offering := range offeringResult.ConsumptionDescribeServiceOfferingResult.Offerings {
		if !strings.EqualFold(offering.ProductTierID, planID) {
			continue
		}
		if _, exists := seenEnvironments[offering.ServiceEnvironmentID]; exists {
			continue
		}
		seenEnvironments[offering.ServiceEnvironmentID] = struct{}{}

		subscriptions, err := listSubscriptionsForEnvironmentFn(cmd.Context(), token, serviceID, offering.ServiceEnvironmentID)
		if err != nil {
			return nil, err
		}

		for _, subscription := range subscriptions.Subscriptions {
			if !strings.EqualFold(subscription.ProductTierId, planID) {
				continue
			}
			if !strings.EqualFold(subscription.RootUserEmail, filters.CustomerEmail) {
				continue
			}
			subscriptionIDs[strings.TrimSpace(subscription.Id)] = struct{}{}
		}
	}

	if len(subscriptionIDs) == 0 {
		return nil, fmt.Errorf(
			"no subscription found for customer %s in service %s plan %s",
			filters.CustomerEmail,
			filters.Service,
			filters.Plan,
		)
	}

	return subscriptionIDs, nil
}

func resolveServicePlanIDAcrossEnvironments(ctx context.Context, token string, serviceID string, planArg string) (string, error) {
	offeringResult, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, "", "")
	if err != nil {
		return "", err
	}
	if offeringResult == nil || offeringResult.ConsumptionDescribeServiceOfferingResult == nil {
		return "", fmt.Errorf("service offering response is empty for service %s", serviceID)
	}

	matches := make(map[string]string)
	for _, offering := range offeringResult.ConsumptionDescribeServiceOfferingResult.Offerings {
		if strings.EqualFold(offering.ProductTierID, planArg) || strings.EqualFold(offering.ProductTierName, planArg) {
			matches[offering.ProductTierID] = offering.ProductTierName
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("service plan %q not found for service %s", planArg, serviceID)
	case 1:
		for productTierID := range matches {
			return productTierID, nil
		}
	default:
		return "", fmt.Errorf("multiple service plans matched %q; use a product tier ID", planArg)
	}

	return "", fmt.Errorf("failed to resolve service plan %q for service %s", planArg, serviceID)
}

func matchesCustomerAccountListFilters(
	record openapiclientfleet.ResourceInstanceSearchRecord,
	filters customerAccountListFilters,
	subscriptionIDs map[string]struct{},
) bool {
	if filters.CloudProvider != "" && !matchesSelector(filters.CloudProvider, record.CloudProvider) {
		return false
	}

	if filters.Service != "" && !matchesSelector(filters.Service, record.ServiceId, record.ServiceName) {
		return false
	}

	planName := ""
	if record.ProductTierName != nil {
		planName = *record.ProductTierName
	}
	if filters.Plan != "" && !matchesSelector(filters.Plan, record.ProductTierId, planName) {
		return false
	}

	if subscriptionIDs == nil {
		return true
	}

	subscriptionID := strings.TrimSpace(utils.FromPtr(record.SubscriptionId))
	if subscriptionID == "" {
		return false
	}

	_, exists := subscriptionIDs[subscriptionID]
	return exists
}

func matchesSelector(selector string, values ...string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return true
	}

	for _, value := range values {
		if strings.EqualFold(selector, strings.TrimSpace(value)) {
			return true
		}
	}

	return false
}
