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
	triggerBackupExample = `# Trigger an automatic backup for an instance
omnistrate-ctl snapshot trigger-backup instance-abcd1234`
)

var triggerBackupCmd = &cobra.Command{
	Use:          "trigger-backup [instance-id]",
	Short:        "Trigger an automatic backup for your instance",
	Long:         `This command helps you trigger an automatic backup for your instance.`,
	Example:      triggerBackupExample,
	RunE:         runTriggerBackup,
	SilenceUsage: true,
}

func init() {
	triggerBackupCmd.Args = cobra.ExactArgs(1)
}

func runTriggerBackup(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]

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
		spinner = sm.AddSpinner("Triggering backup...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := common.GetInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	result, err := dataaccess.TriggerResourceInstanceAutoBackup(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully triggered backup")

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
