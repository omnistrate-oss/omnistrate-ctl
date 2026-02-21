package snapshot

import (
	"fmt"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"

	"github.com/chelnak/ysmrr"
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
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json`
)

var restoreCmd = &cobra.Command{
	Use:          "restore --service-id <service-id> --environment-id <environment-id> --snapshot-id <snapshot-id>",
	Short:        "Create a new instance by restoring from a snapshot",
	Long:         `This command helps you create a new instance by restoring from a snapshot.`,
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

	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner

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
		var choice string
		choice, err = prompt.New().Ask(fmt.Sprintf("NOTICE: System is initiating restoration with service ID %s, using service plan version %s override. Please verify plan compatibility with the target snapshot before proceeding. Continue to proceed?", serviceID, tierVersionOverride)).
			Choose([]string{
				"Yes",
				"No",
			}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			utils.PrintError(err)
			return err
		}

		switch choice {
		case "Yes":
			break
		case "No":
			utils.PrintInfo("Operation cancelled")
			return nil
		}
	}

	networkType, err := cmd.Flags().GetString("network-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Creating new instance from snapshot...")
		sm.Start()
	}

	result, err := dataaccess.RestoreSnapshot(cmd.Context(), token, serviceID, environmentID, snapshotID, formattedParams, tierVersionOverride, networkType)
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
