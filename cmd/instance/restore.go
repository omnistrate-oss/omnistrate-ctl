package instance

import (
	"errors"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	restoreExample = `# Restore to a new instance from a snapshot
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json

# Restore to the original source instance (preserving its ID and endpoint)
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --restore-to-source --service-id service-xyz --environment-id env-abc`
)

var restoreCmd = &cobra.Command{
	Use:          "restore [instance-id] --snapshot-id <snapshot-id> [--service-id <service-id>] [--environment-id <environment-id>] [--param=param] [--param-file=file-path] [--restore-to-source] [--tierversion-override <tier-version>] [--network-type PUBLIC / INTERNAL]",
	Short:        "Restore an instance from a snapshot",
	Long:         `This command helps you restore an instance from a snapshot. By default, a new instance is created. When --restore-to-source is set, the snapshot is restored to the original source instance, preserving its ID and endpoint. Use --service-id and --environment-id when restoring a deleted instance that can no longer be looked up in inventory.`,
	Example:      restoreExample,
	RunE:         runRestore,
	SilenceUsage: true,
}

func init() {
	restoreCmd.Args = cobra.ExactArgs(1)
	restoreCmd.Flags().String("snapshot-id", "", "The ID of the snapshot to restore from")
	restoreCmd.Flags().String("service-id", "", "The ID of the service (required when the instance has been deleted)")
	restoreCmd.Flags().String("environment-id", "", "The ID of the environment (required when the instance has been deleted)")
	restoreCmd.Flags().String("param", "", "Parameters override for the instance deployment")
	restoreCmd.Flags().String("param-file", "", "Json file containing parameters override for the instance deployment")
	restoreCmd.Flags().String("tierversion-override", "", "Override the tier version for the restored instance")
	restoreCmd.Flags().String("network-type", "", "Optional network type change for the instance deployment (PUBLIC / INTERNAL)")
	restoreCmd.Flags().Bool("restore-to-source", false, "Restore to the original source instance, preserving its ID and endpoint")

	if err := restoreCmd.MarkFlagRequired("snapshot-id"); err != nil {
		return
	}
	if err := restoreCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}
}

func runRestore(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args and flags
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
	param, err := cmd.Flags().GetString("param")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	paramFile, err := cmd.Flags().GetString("param-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get optional service-id and environment-id flags
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

	// Get restore-to-source flag
	restoreToSource, err := cmd.Flags().GetBool("restore-to-source")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate service-id/environment-id flags
	if (serviceID == "") != (environmentID == "") {
		err = errors.New("--service-id and --environment-id must both be provided")
		utils.PrintError(err)
		return err
	}
	if restoreToSource && serviceID == "" {
		err = errors.New("--service-id and --environment-id are required when using --restore-to-source")
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

	// Get service and environment IDs from the instance if not provided via flags
	if serviceID == "" {
		serviceID, environmentID, _, _, err = getInstance(cmd.Context(), token, instanceID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Format parameters
	formattedParams, err := common.FormatParams(param, paramFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get tier version override
	tierVersionOverride, err := cmd.Flags().GetString("tierversion-override")
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// If tier version override is set, show warning and require confirmation
	if tierVersionOverride != "" {
		confirmed, err := utils.ConfirmAction(fmt.Sprintf("NOTICE: System is initiating restoration of instance with service ID %s, using service plan version %s override. Please verify plan compatibility with the target snapshot before proceeding. Continue to proceed?", serviceID, tierVersionOverride))
		if err != nil {
			utils.PrintError(err)
			return err
		}
		if !confirmed {
			utils.PrintInfo("Operation cancelled")
			return nil
		}
	}

	// Get network type override
	networkType, err := cmd.Flags().GetString("network-type")
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Creating new instance from snapshot..."
		if restoreToSource {
			msg = "Restoring snapshot to original source instance..."
		}
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Restore from snapshot
	result, err := dataaccess.RestoreResourceInstanceSnapshot(cmd.Context(), token, serviceID, environmentID, snapshotID, formattedParams, tierVersionOverride, networkType, restoreToSource)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated restore operation from snapshot")

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
