package subscription

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listForServiceCmd = &cobra.Command{
	Use:   "list-for-service",
	Short: "List subscriptions for a specific service environment",
	Long:  "List all subscriptions for a specific service and environment using fleet API.",
	RunE:  runListForService,
}

func init() {
	listForServiceCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	listForServiceCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = listForServiceCmd.MarkFlagRequired("service-id")
	_ = listForServiceCmd.MarkFlagRequired("environment-id")
}

func runListForService(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.ListSubscriptions(ctx, token, serviceID, environmentID)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}