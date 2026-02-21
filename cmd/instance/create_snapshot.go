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
	createSnapshotExample = `# Create a snapshot for an instance
omnistrate-ctl instance create-snapshot instance-abcd1234

# Create a snapshot in a specific region
omnistrate-ctl instance create-snapshot instance-abcd1234 --target-region us-east1

# Create a snapshot with JSON output
omnistrate-ctl instance create-snapshot instance-abcd1234 --output json`
)

var createSnapshotCmd = &cobra.Command{
	Use:          "create-snapshot [instance-id]",
	Short:        "Create a snapshot for an instance",
	Long:         `This command helps you create an on-demand snapshot of your instance. Optionally specify a target region for the snapshot.`,
	Example:      createSnapshotExample,
	RunE:         runCreateSnapshot,
	SilenceUsage: true,
}

func init() {
	createSnapshotCmd.Args = cobra.ExactArgs(1)
	createSnapshotCmd.Flags().String("target-region", "", "The target region to create the snapshot in (defaults to the instance region)")
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

	result, err := dataaccess.CreateInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, targetRegion)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated snapshot creation")

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
