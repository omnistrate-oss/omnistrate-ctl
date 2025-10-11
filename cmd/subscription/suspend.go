package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var suspendCmd = &cobra.Command{
	Use:   "suspend <subscription-id>",
	Short: "Suspend a subscription",
	Long:  "Temporarily suspend a subscription for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSuspend,
}

func init() {
	suspendCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	suspendCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = suspendCmd.MarkFlagRequired("service-id")
	_ = suspendCmd.MarkFlagRequired("environment-id")
}

func runSuspend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	subscriptionID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.SuspendSubscription(ctx, token, serviceID, environmentID, subscriptionID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully suspended subscription %s\n", subscriptionID)
	return nil
}
