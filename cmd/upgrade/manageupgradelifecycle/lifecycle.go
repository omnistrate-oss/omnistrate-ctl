package manageupgradelifecycle

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	pauseExample = ` Pausing ongoing upgrade # 
omnistrate-ctl upgrade pause [upgrade-id] `
	resumeExample = ` Resuming paused upgrade # 
omnistrate-ctl upgrade resume [upgrade-id] `
	cancelExample = ` Cancelling uncompleted upgrade # 
omnistrate-ctl upgrade cancel [upgrade-id] `
	notifyCustomerExample = ` Enable customer notifications for a scheduled upgrade # 
omnistrate-ctl upgrade notify-customer [upgrade-id] `
	skipInstancesExample = ` Skip specific instances from an upgrade path #
omnistrate-ctl upgrade skip-instances [upgrade-id] --resource-ids instance-1,instance-2 `
)

var PauseCmd = &cobra.Command{
	Use:          "pause [upgrade-id] [flags]",
	Short:        "Pause an ongoing upgrade",
	Example:      pauseExample,
	RunE:         pause,
	SilenceUsage: true,
}

var ResumeCmd = &cobra.Command{
	Use:          "resume [upgrade-id] [flags]",
	Short:        "Resume a paused upgrade",
	Example:      resumeExample,
	RunE:         resume,
	SilenceUsage: true,
}

var CancelCmd = &cobra.Command{
	Use:          "cancel [upgrade-id] [flags]",
	Short:        "Cancel an uncompleted upgrade",
	Example:      cancelExample,
	RunE:         cancel,
	SilenceUsage: true,
}

var NotifyCustomerCmd = &cobra.Command{
	Use:          "notify-customer [upgrade-id] [flags]",
	Short:        "Enable customer notifications for a scheduled upgrade",
	Example:      notifyCustomerExample,
	RunE:         notifyCustomer,
	SilenceUsage: true,
}

var SkipInstancesCmd = &cobra.Command{
	Use:          "skip-instances [upgrade-id] [flags]",
	Short:        "Skip specific instances from an upgrade path",
	Example:      skipInstancesExample,
	RunE:         skipInstances,
	SilenceUsage: true,
}

func init() {
	PauseCmd.Args = cobra.MinimumNArgs(1)
	ResumeCmd.Args = cobra.MinimumNArgs(1)
	CancelCmd.Args = cobra.MinimumNArgs(1)
	NotifyCustomerCmd.Args = cobra.MinimumNArgs(1)
	SkipInstancesCmd.Args = cobra.MinimumNArgs(1)

	SkipInstancesCmd.Flags().String("resource-ids", "", "Comma-separated list of instance IDs to skip")
	_ = SkipInstancesCmd.MarkFlagRequired("resource-ids")
}

func cancel(cmd *cobra.Command, args []string) error {
	return manageLifecycle(cmd, args, model.CancelAction, nil)
}

func pause(cmd *cobra.Command, args []string) error {
	return manageLifecycle(cmd, args, model.PauseAction, nil)
}

func resume(cmd *cobra.Command, args []string) error {
	return manageLifecycle(cmd, args, model.ResumeAction, nil)
}

func notifyCustomer(cmd *cobra.Command, args []string) error {
	return manageLifecycle(cmd, args, model.NotifyCustomerAction, nil)
}

func skipInstances(cmd *cobra.Command, args []string) error {
	resourceIDs, err := cmd.Flags().GetString("resource-ids")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"resource-ids": resourceIDs,
	}
	return manageLifecycle(cmd, args, model.SkipInstancesAction, payload)
}

func manageLifecycle(cmd *cobra.Command, args []string, action model.UpgradeMaintenanceAction, actionPayload map[string]interface{}) error {
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
		msg := fmt.Sprintf("Managing lifecycle of upgrade %s", args[0])
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

		upgrade, err := dataaccess.ManageLifecycleWithPayload(cmd.Context(), token, serviceID, productTierID, upgradePathID, action, actionPayload)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		formattedUpgradeStatuses = append(formattedUpgradeStatuses, &model.UpgradeStatus{
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
		})
	}

	if len(formattedUpgradeStatuses) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No upgrades found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Upgrade pause request submitted")
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
			println(fmt.Sprintf("  omnistrate-ctl upgrade pause detail %s", s.UpgradeID))
		}
	}

	return nil
}
