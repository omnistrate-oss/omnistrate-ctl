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
	copyExample = `# Copy a snapshot to another region
omnistrate-ctl snapshot copy instance-abcd1234 --snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Copy the latest snapshot to another region
omnistrate-ctl snapshot copy instance-abcd1234 --target-region us-east1
`
)

var copyCmd = &cobra.Command{
	Use:          "copy [instance-id] --target-region <region>",
	Aliases:      []string{"create"},
	Short:        "Copy an instance snapshot to another region",
	Long:         `This command helps you copy an instance snapshot to a different region in the same cloud account for redundancy or disaster recovery. Does not support cross-cloud / cross-account snapshot copies at this time.`,
	Example:      copyExample,
	RunE:         runCopy,
	SilenceUsage: true,
}

func init() {
	copyCmd.Args = cobra.ExactArgs(1)
	copyCmd.Flags().String("snapshot-id", "", "The ID of the snapshot or backup to copy. If not provided, the latest snapshot will be used.")
	copyCmd.Flags().String("target-region", "", "The target region to copy the snapshot into (required)")

	if err := copyCmd.MarkFlagRequired("target-region"); err != nil {
		return
	}
}

func runCopy(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

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

	serviceID, environmentID, _, _, err := common.GetInstance(cmd.Context(), token, instanceID)
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
