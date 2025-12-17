package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe subscription
omnistrate-ctl subscription describe [subscription-id]`
)

var describeCmd = &cobra.Command{
	Use:          "describe [subscription-id]",
	Short:        "Describe a Customer Subscription to your service",
	Long:         `This command helps you get detailed information about a Customer Subscription.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	subscriptionID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate output flag
	if output != "json" {
		err = errors.New("only json output is supported")
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
		msg := "Describing subscription..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if the subscription exists
	subscription, err := getSubscription(cmd.Context(), token, subscriptionID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved subscription details")

	// Format subscription
	formattedSubscription := formatSubscription(subscription, false)

	// Print output
	err = utils.PrintTextTableJsonOutput(output, formattedSubscription)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// Helper functions

func getSubscription(ctx context.Context, token, subscriptionID string) (*openapiclientfleet.SubscriptionSearchRecord, error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("subscription:%s", subscriptionID))
	if err != nil {
		return nil, err
	}

	for _, subscription := range searchRes.SubscriptionResults {
		if subscription.Id == subscriptionID {
			return &subscription, nil
		}
	}

	err = fmt.Errorf("%s not found. Please check the subscription ID and try again", subscriptionID)
	return nil, err
}

func formatSubscription(subscription *openapiclientfleet.SubscriptionSearchRecord, truncateNames bool) model.Subscription {
	serviceName := subscription.ServiceName
	planName := subscription.ServicePlanName
	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		planName = utils.TruncateString(planName, defaultMaxNameLength)
	}

	formattedSubscription := model.Subscription{
		SubscriptionID:         subscription.Id,
		ServiceID:              subscription.ServiceID,
		ServiceName:            serviceName,
		PlanID:                 subscription.ProductTierID,
		PlanName:               planName,
		Environment:            subscription.ServiceEnvironmentName,
		SubscriptionOwnerName:  subscription.RootUserName,
		SubscriptionOwnerEmail: subscription.RootUserEmail,
		Status:                 subscription.Status,
	}

	return formattedSubscription
}
