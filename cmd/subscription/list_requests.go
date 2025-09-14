package subscription

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listRequestsCmd = &cobra.Command{
	Use:   "list-requests",
	Short: "List subscription requests",
	Long:  "List all pending and processed subscription requests for a service environment.",
	RunE:  runListRequests,
}

func init() {
	listRequestsCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	listRequestsCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = listRequestsCmd.MarkFlagRequired("service-id")
	_ = listRequestsCmd.MarkFlagRequired("environment-id")
}

func runListRequests(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.ListSubscriptionRequests(ctx, token, serviceID, environmentID)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}
