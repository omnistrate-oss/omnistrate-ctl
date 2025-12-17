package detail

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
	detailExample = `# Get upgrade status detail
omnistrate-ctl upgrade status detail [upgrade-id]`
)

var Cmd = &cobra.Command{
	Use:          "detail [upgrade-id] [flags]",
	Short:        "Get Upgrade status detail",
	Example:      detailExample,
	RunE:         run,
	SilenceUsage: true,
}

func init() {
	Cmd.Args = cobra.ExactArgs(1)
}

func run(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	upgradePathID := args[0]

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
		msg := "Retrieving upgrade status detail..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	formattedUpgradeStatusDetails := make([]*model.UpgradeStatusDetail, 0)

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

	instanceUpgrades, err := dataaccess.ListEligibleInstancesPerUpgrade(cmd.Context(), token, serviceID, productTierID, upgradePathID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	for _, instanceUpgrade := range instanceUpgrades {
		startTime := ""
		if instanceUpgrade.UpgradeStartTime != nil {
			startTime = *instanceUpgrade.UpgradeStartTime
		}

		endTime := ""
		if instanceUpgrade.UpgradeEndTime != nil {
			endTime = *instanceUpgrade.UpgradeEndTime
		}
		formattedUpgradeStatusDetails = append(formattedUpgradeStatusDetails, &model.UpgradeStatusDetail{
			UpgradeID:        upgradePathID,
			InstanceID:       instanceUpgrade.InstanceId,
			UpgradeStatus:    instanceUpgrade.Status,
			UpgradeStartTime: startTime,
			UpgradeEndTime:   endTime,
		})
	}

	if len(formattedUpgradeStatusDetails) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No upgrade found")
		return nil
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Upgrade status detail retrieved")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, formattedUpgradeStatusDetails)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
