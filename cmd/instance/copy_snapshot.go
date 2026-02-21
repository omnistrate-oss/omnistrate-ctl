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
	copySnapshotExample = `# Copy a snapshot to another region
omnistrate-ctl instance copy-snapshot instance-abcd1234 --snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Copy the latest snapshot to another region
omnistrate-ctl instance copy-snapshot instance-abcd1234 --target-region us-east1
`
)

var copySnapshotCmd = &cobra.Command{
	Use:          "copy-snapshot [instance-id] --snapshot-id <snapshot-id> --target-region <region>",
	Short:        "Copy an instance snapshot to another region",
	Long:         `This command helps you copy an instance snapshot to a different region in the same cloud account for redundancy or disaster recovery. Do not support cross-cloud / cross-account snapshot copies at this time.`,
	Example:      copySnapshotExample,
	RunE:         runCopySnapshot,
	SilenceUsage: true,
}

func init() {
	copySnapshotCmd.Args = cobra.ExactArgs(1)
	copySnapshotCmd.Flags().String("snapshot-id", "", "The ID of the snapshot or backup to copy. If not provided, the latest snapshot will be used.")
	copySnapshotCmd.Flags().String("target-region", "", "The region to copy the snapshot into")

	if err := copySnapshotCmd.MarkFlagRequired("target-region"); err != nil {
		return
	}
}

func runCopySnapshot(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	instanceID := args[0]

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	snapshotID, err := cmd.Flags().GetString("snapshot-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	targetRegion, err := cmd.Flags().GetString("target-region")
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
		spinner = sm.AddSpinner("Copying snapshot...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	result, err := dataaccess.CopyResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, snapshotID, targetRegion)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated snapshot copy")

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
