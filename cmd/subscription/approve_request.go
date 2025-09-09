package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var approveRequestCmd = &cobra.Command{
	Use:   "approve-request <request-id>",
	Short: "Approve a subscription request",
	Long:  "Approve a pending subscription request for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runApproveRequest,
}

func init() {
	approveRequestCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	approveRequestCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = approveRequestCmd.MarkFlagRequired("service-id")
	_ = approveRequestCmd.MarkFlagRequired("environment-id")
}

func runApproveRequest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	requestID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.ApproveSubscriptionRequest(ctx, token, serviceID, environmentID, requestID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully approved subscription request %s\n", requestID)
	return nil
}