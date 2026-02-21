package instance

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	deleteSnapshotExample = `# Delete a specific snapshot
omnistrate-ctl instance delete-snapshot snapshot-xyz789 --service-id service-abcd --environment-id env-1234`
)

var deleteSnapshotCmd = &cobra.Command{
	Use:          "delete-snapshot [snapshot-id] --service-id <service-id> --environment-id <environment-id>",
	Short:        "Delete an instance snapshot",
	Long:         `This command helps you delete a specific snapshot.`,
	Example:      deleteSnapshotExample,
	RunE:         runDeleteSnapshot,
	SilenceUsage: true,
}

func init() {
	deleteSnapshotCmd.Args = cobra.ExactArgs(1)
	deleteSnapshotCmd.Flags().String("service-id", "", "The ID of the service (required)")
	deleteSnapshotCmd.Flags().String("environment-id", "", "The ID of the environment (required)")

	if err := deleteSnapshotCmd.MarkFlagRequired("service-id"); err != nil {
		return
	}
	if err := deleteSnapshotCmd.MarkFlagRequired("environment-id"); err != nil {
		return
	}
}

func runDeleteSnapshot(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	snapshotID := args[0]

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	environmentID, err := cmd.Flags().GetString("environment-id")
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

	err = dataaccess.DeleteResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, snapshotID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted snapshot")

	utils.PrintSuccess("Successfully deleted snapshot")
	return nil
}
