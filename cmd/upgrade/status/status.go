package status

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/upgrade/status/detail"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	statusExample = `# Get upgrade status
omnistrate-ctl upgrade status [upgrade-id]`
)

var LastUpgradeStatus model.UpgradeStatus
var Cmd = &cobra.Command{
	Use:          "status [upgrade-id] [flags]",
	Short:        "Get Upgrade status",
	Example:      statusExample,
	RunE:         run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(detail.Cmd)

	Cmd.Args = cobra.MinimumNArgs(1)

}

func run(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not json
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Retrieving upgrade status..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	formattedUpgradeStatuses := make([]*model.UpgradeStatus, 0)

	for _, upgradePathID := range args {
		searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("upgradepath:%s", upgradePathID))
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if len(searchRes.UpgradePathResults) == 0 {
			err = fmt.Errorf("%s not found", upgradePathID)
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		found := false
		var serviceID, productTierID string
		for _, upgradePath := range searchRes.UpgradePathResults {
			if upgradePath.Id == upgradePathID {
				found = true
				serviceID = upgradePath.ServiceId
				productTierID = upgradePath.ProductTierID
				break
			}
		}

		if !found {
			err = fmt.Errorf("%s not found", upgradePathID)
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		upgrade, err := dataaccess.DescribeUpgradePath(cmd.Context(), token, serviceID, productTierID, upgradePathID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		LastUpgradeStatus = model.UpgradeStatus{
			UpgradeID:      upgradePathID,
			Total:          upgrade.TotalCount,
			Pending:        upgrade.PendingCount,
			InProgress:     upgrade.InProgressCount,
			Completed:      upgrade.CompletedCount,
			Failed:         upgrade.FailedCount,
			Scheduled:      utils.FromInt64Ptr(upgrade.ScheduledCount),
			Skipped:        upgrade.SkippedCount,
			Status:         upgrade.Status,
			NotifyCustomer: utils.FromPtrOrDefault(upgrade.NotifyCustomer, false),
		}
		formattedUpgradeStatuses = append(formattedUpgradeStatuses, &LastUpgradeStatus)
	}

	if len(formattedUpgradeStatuses) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No upgrades found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Upgrade status retrieved")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, formattedUpgradeStatuses)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" {
		println("\nTo get more details, run the following command(s):")
		for _, s := range formattedUpgradeStatuses {
			println(fmt.Sprintf("  omnistrate-ctl upgrade status detail %s", s.UpgradeID))
		}
	}

	return nil
}
