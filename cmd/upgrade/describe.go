package upgrade

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <upgrade-path-id>",
	Short: "Describe an upgrade path",
	Long:  "Get detailed information about a specific upgrade path.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribe,
}

func init() {
	describeCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	describeCmd.Flags().StringP("product-tier-id", "p", "", "Product tier ID (required)")

	_ = describeCmd.MarkFlagRequired("service-id")
	_ = describeCmd.MarkFlagRequired("product-tier-id")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	upgradePathID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.DescribeUpgradePath(ctx, token, serviceID, productTierID, upgradePathID)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}