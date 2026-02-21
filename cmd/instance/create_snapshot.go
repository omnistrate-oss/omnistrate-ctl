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
	createSnapshotExample = `# Create a snapshot from the current instance state
omnistrate-ctl instance create-snapshot instance-abcd1234

# Create a snapshot in a specific region
omnistrate-ctl instance create-snapshot instance-abcd1234 --target-region us-east1

# Create a snapshot from another snapshot (copies to a target region)
omnistrate-ctl instance create-snapshot instance-abcd1234 --source-snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Create a snapshot with JSON output
omnistrate-ctl instance create-snapshot instance-abcd1234 --output json`
)

var createSnapshotCmd = &cobra.Command{
	Use:   "create-snapshot [instance-id]",
	Short: "Create a snapshot for an instance",
	Long: `This command helps you create an on-demand snapshot of your instance.

By default it creates a snapshot from the current instance state. Optionally specify
a target region for the snapshot.

When --source-snapshot-id is provided, the snapshot is created by copying the specified
source snapshot to the target region. In this mode --target-region is required.`,
	Example:      createSnapshotExample,
	RunE:         runCreateSnapshot,
	SilenceUsage: true,
}

func init() {
	createSnapshotCmd.Args = cobra.ExactArgs(1)
	createSnapshotCmd.Flags().String("target-region", "", "The target region to create the snapshot in (defaults to the instance region)")
	createSnapshotCmd.Flags().String("source-snapshot-id", "", "Source snapshot ID to create the new snapshot from (uses the copy API; requires --target-region)")
}

func runCreateSnapshot(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	targetRegion, err := cmd.Flags().GetString("target-region")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	sourceSnapshotID, err := cmd.Flags().GetString("source-snapshot-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if sourceSnapshotID != "" && targetRegion == "" {
		err := errors.New("--target-region is required when --source-snapshot-id is specified")
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
		spinner = sm.AddSpinner("Creating snapshot...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if sourceSnapshotID != "" {
		// Create from an existing snapshot using the copy API
		result, err := dataaccess.CopyResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, sourceSnapshotID, targetRegion)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated snapshot creation from source snapshot")

		if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
			return err
		}
	} else {
		// Create from the current instance state
		result, err := dataaccess.CreateInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, targetRegion)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated snapshot creation")

		if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
			return err
		}
	}

	return nil
}
