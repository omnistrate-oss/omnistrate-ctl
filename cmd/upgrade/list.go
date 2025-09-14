package upgrade

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List upgrade paths",
	Long:  "List upgrade paths for a service and product tier with filtering options.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	listCmd.Flags().StringP("product-tier-id", "p", "", "Product tier ID (required)")
	listCmd.Flags().String("source-version", "", "Source product tier version to filter by")
	listCmd.Flags().String("target-version", "", "Target product tier version to filter by")
	listCmd.Flags().String("status", "", "Status of upgrade path to filter by")
	listCmd.Flags().String("type", "", "Type of upgrade path to filter by")
	listCmd.Flags().String("next-page-token", "", "Token for next page of results")
	listCmd.Flags().Int64("page-size", 0, "Number of results per page")

	_ = listCmd.MarkFlagRequired("service-id")
	_ = listCmd.MarkFlagRequired("product-tier-id")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	serviceID, _ := cmd.Flags().GetString("service-id")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")
	sourceVersion, _ := cmd.Flags().GetString("source-version")
	targetVersion, _ := cmd.Flags().GetString("target-version")
	status, _ := cmd.Flags().GetString("status")
	upgradeType, _ := cmd.Flags().GetString("type")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	pageSize, _ := cmd.Flags().GetInt64("page-size")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	opts := &dataaccess.ListUpgradePathsOptions{
		SourceProductTierVersion: sourceVersion,
		TargetProductTierVersion: targetVersion,
		Status:                   status,
		Type:                     upgradeType,
		NextPageToken:            nextPageToken,
	}

	if pageSize > 0 {
		opts.PageSize = &pageSize
	}

	result, err := dataaccess.ListUpgradePaths(ctx, token, serviceID, productTierID, opts)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}
