package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var denyRequestCmd = &cobra.Command{
	Use:   "deny-request <request-id>",
	Short: "Deny a subscription request",
	Long:  "Deny a pending subscription request for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDenyRequest,
}

func init() {
	denyRequestCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	denyRequestCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = denyRequestCmd.MarkFlagRequired("service-id")
	_ = denyRequestCmd.MarkFlagRequired("environment-id")
}

func runDenyRequest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	requestID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.DenySubscriptionRequest(ctx, token, serviceID, environmentID, requestID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully denied subscription request %s\n", requestID)
	return nil
}
