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
	describeSnapshotExample = `# Describe a specific snapshot
omctl instance describe-snapshot instance-abcd1234 snapshot-xyz789`
)

var describeSnapshotCmd = &cobra.Command{
	Use:          "describe-snapshot [instance-id] [snapshot-id]",
	Short:        "Describe a specific instance snapshot",
	Long:         `This command helps you get detailed information about a specific snapshot for your instance.`,
	Example:      describeSnapshotExample,
	RunE:         runDescribeSnapshot,
	SilenceUsage: true,
}

func init() {
	describeSnapshotCmd.Args = cobra.ExactArgs(2) // Require exactly two arguments
}

func runDescribeSnapshot(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) < 2 {
		err := errors.New("both instance id and snapshot id are required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]
	snapshotID := args[1]

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
		msg := "Describing snapshot..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists and get its details
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe snapshot
	result, err := dataaccess.DescribeInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, instanceID, snapshotID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully described snapshot")

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}