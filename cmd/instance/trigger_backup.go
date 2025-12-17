package instance

import (
	"errors"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	triggerBackupExample = `# Trigger an automatic backup for an instance
omnistrate-ctl instance trigger-backup instance-abcd1234`
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
	triggerBackupCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runTriggerBackup(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]

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

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Triggering backup..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists and get its details
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Trigger backup
	result, err := dataaccess.TriggerResourceInstanceAutoBackup(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully triggered backup")

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
