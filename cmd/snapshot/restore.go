package snapshot

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	restoreExample = `# Restore to a new instance from a snapshot
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json

# Restore to the original source instance (preserving its ID and endpoint)
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --restore-to-source`
)

var restoreCmd = &cobra.Command{
	Use:          "restore --service-id <service-id> --environment-id <environment-id> --snapshot-id <snapshot-id> [--restore-to-source]",
	Short:        "Create an instance by restoring from a snapshot",
	Long:         `This command helps you create an instance by restoring from a snapshot.`,
	Example:      restoreExample,
	RunE:         runRestore,
	SilenceUsage: true,
}

func init() {
	restoreCmd.Args = cobra.NoArgs
	restoreCmd.Flags().String("service-id", "", "The ID of the service (required)")
	restoreCmd.Flags().String("environment-id", "", "The ID of the environment (required)")
	restoreCmd.Flags().String("snapshot-id", "", "The ID of the snapshot to restore from (required)")
	restoreCmd.Flags().String("param", "", "Parameters override for the instance deployment")
	restoreCmd.Flags().String("param-file", "", "Json file containing parameters override for the instance deployment")
	restoreCmd.Flags().String("tierversion-override", "", "Override the tier version for the restored instance")
	restoreCmd.Flags().String("network-type", "", "Optional network type change for the instance deployment (PUBLIC / INTERNAL)")
	restoreCmd.Flags().Bool("restore-to-source", false, "Restore to the original source instance, preserving its ID and endpoint")

	if err := restoreCmd.MarkFlagRequired("service-id"); err != nil {
		return
	}
	if err := restoreCmd.MarkFlagRequired("environment-id"); err != nil {
		return
	}
	if err := restoreCmd.MarkFlagRequired("snapshot-id"); err != nil {
		return
	}
	if err := restoreCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}
}

func runRestore(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

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

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner

	formattedParams, err := common.FormatParams(param, paramFile)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	tierVersionOverride, err := cmd.Flags().GetString("tierversion-override")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if tierVersionOverride != "" {
		confirmed, err := utils.ConfirmAction(fmt.Sprintf("NOTICE: System is initiating restoration with service ID %s, using service plan version %s override. Please verify plan compatibility with the target snapshot before proceeding. Continue to proceed?", serviceID, tierVersionOverride))
		if err != nil {
			utils.PrintError(err)
			return err
		}
		if !confirmed {
			utils.PrintInfo("Operation cancelled")
			return nil
		}
	}

	networkType, err := cmd.Flags().GetString("network-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	restoreToSource, err := cmd.Flags().GetBool("restore-to-source")
	if err != nil {
		utils.PrintError(err)
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

	result, err := dataaccess.RestoreSnapshot(cmd.Context(), token, serviceID, environmentID, snapshotID, formattedParams, tierVersionOverride, networkType, restoreToSource)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated restore operation from snapshot")

	if err = utils.PrintTextTableJsonOutput(output, result); err != nil {
		return err
	}

	return nil
}
