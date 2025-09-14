package operations

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Get service health summary",
	Long:  "Get the overall health status of a service and its environment.",
	RunE:  runHealth,
}

func init() {
	healthCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	healthCmd.Flags().StringP("environment-id", "e", "", "Service environment ID (required)")

	_ = healthCmd.MarkFlagRequired("service-id")
	_ = healthCmd.MarkFlagRequired("environment-id")
}

func runHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.GetServiceHealth(ctx, token, serviceID, environmentID)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}
