package instance

import (
	"errors"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/formatter"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listSnapshotsExample = `# List snapshots for an instance
omnistrate-ctl instance list-snapshots instance-abcd1234"`
)

var listSnapshotsCmd = &cobra.Command{
	Use:          "list-snapshots [instance-id]",
	Short:        "List all snapshots for an instance",
	Long:         `This command helps you list all snapshots available for your instance.`,
	Example:      listSnapshotsExample,
	RunE:         runListSnapshots,
	SilenceUsage: true,
}

func init() {
	listSnapshotsCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runListSnapshots(cmd *cobra.Command, args []string) error {
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
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Listing snapshots..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists and get its details
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// List snapshots
	result, err := dataaccess.ListResourceInstanceSnapshots(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully listed snapshots")

	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if result == nil || len(result.Snapshots) == 0 {
		utils.PrintInfo(fmt.Sprintf("No snapshots found for instance %s.", instanceID))
		return nil
	}

	return utils.PrintTextTableJsonArrayOutput(output, formatter.FormatSnapshotSummaries(result.Snapshots))
}
