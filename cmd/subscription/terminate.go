package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var terminateCmd = &cobra.Command{
	Use:   "terminate <subscription-id>",
	Short: "Terminate a subscription",
	Long:  "Permanently terminate a subscription for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTerminate,
}

func init() {
	terminateCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	terminateCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = terminateCmd.MarkFlagRequired("service-id")
	_ = terminateCmd.MarkFlagRequired("environment-id")
}

func runTerminate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	subscriptionID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.TerminateSubscription(ctx, token, serviceID, environmentID, subscriptionID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully terminated subscription %s\n", subscriptionID)
	return nil
}