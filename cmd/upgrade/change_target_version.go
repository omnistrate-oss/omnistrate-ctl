package upgrade

import (
	"context"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const changeTargetVersionExample = `# Change the target version for a scheduled upgrade path
omnistrate-ctl upgrade change-target-version upgrade-abcd1234 --service-id s-abcd1234 --product-tier-id pt-abcd1234 --target-version 90.0`

var changeTargetVersionCmd = &cobra.Command{
	Use:     "change-target-version <upgrade-path-id>",
	Short:   "Change the target version for a scheduled upgrade path",
	Long:    "Change the target product tier version for a scheduled upgrade path before it starts.",
	Example: changeTargetVersionExample,
	Args:    cobra.ExactArgs(1),
	RunE:    runChangeTargetVersion,
}

func init() {
	changeTargetVersionCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	changeTargetVersionCmd.Flags().StringP("product-tier-id", "p", "", "Product tier ID (required)")
	changeTargetVersionCmd.Flags().String("target-version", "", "New target product tier version (required)")

	_ = changeTargetVersionCmd.MarkFlagRequired("service-id")
	_ = changeTargetVersionCmd.MarkFlagRequired("product-tier-id")
	_ = changeTargetVersionCmd.MarkFlagRequired("target-version")
}

func runChangeTargetVersion(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	upgradePathID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")
	targetVersion, _ := cmd.Flags().GetString("target-version")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	result, err := dataaccess.ChangeUpgradePathTargetVersion(ctx, token, serviceID, productTierID, upgradePathID, targetVersion)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}
