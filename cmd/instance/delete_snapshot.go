package instance

import (
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	deleteSnapshotExample = `# Delete a specific snapshot
omnistrate-ctl instance delete-snapshot instance-abcd1234 --snapshot-id snapshot-xyz789`
)

var deleteSnapshotCmd = &cobra.Command{
	Use:          "delete-snapshot [instance-id] --snapshot-id <snapshot-id>",
	Short:        "Delete an instance snapshot",
	Long:         `This command helps you delete a specific snapshot for your instance.`,
	Example:      deleteSnapshotExample,
	RunE:         runDeleteSnapshot,
	SilenceUsage: true,
}

func init() {
	deleteSnapshotCmd.Args = cobra.ExactArgs(1)
	deleteSnapshotCmd.Flags().String("snapshot-id", "", "The ID of the snapshot to delete (required)")

	if err := deleteSnapshotCmd.MarkFlagRequired("snapshot-id"); err != nil {
		return
	}
}

func runDeleteSnapshot(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]

	snapshotID, err := cmd.Flags().GetString("snapshot-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, _ := cmd.Flags().GetString("output")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Deleting snapshot...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	err = dataaccess.DeleteResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, snapshotID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted snapshot")

	fmt.Printf("Successfully deleted snapshot '%s' from instance '%s'\n", snapshotID, instanceID)
	return nil
}
