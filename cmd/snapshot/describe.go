package snapshot

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe a specific snapshot
omnistrate-ctl snapshot describe instance-abcd1234 snapshot-xyz789`
)

var describeCmd = &cobra.Command{
	Use:          "describe [instance-id] [snapshot-id]",
	Short:        "Describe a specific instance snapshot",
	Long:         `This command helps you get detailed information about a specific snapshot for your instance.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(2)
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]
	snapshotID := args[1]

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Describing snapshot...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := common.GetInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	result, err := dataaccess.DescribeResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, snapshotID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully described snapshot")

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
