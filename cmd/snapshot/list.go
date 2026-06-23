package snapshot

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/formatter"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List all snapshots for a service environment
omnistrate-ctl snapshot list --service-id service-abcd --environment-id env-1234`

	snapshotTypeManual    = "ManualSnapshot"
	snapshotTypeAutomated = "AutomatedSnapshot"
	snapshotTypeAll       = "all"
)

var listCmd = &cobra.Command{
	Use:          "list --service-id <service-id> --environment-id <environment-id>",
	Short:        "List all snapshots for a service environment",
	Long:         `This command helps you list all snapshots available across all instances in a service environment.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Args = cobra.NoArgs
	listCmd.Flags().String("service-id", "", "The ID of the service (required)")
	listCmd.Flags().String("environment-id", "", "The ID of the environment (required)")
	listCmd.Flags().String("snapshot-type", snapshotTypeManual, "Filter by snapshot type: ManualSnapshot, AutomatedSnapshot, or all")
	listCmd.Flags().String("product-tier-id", "", "Filter snapshots by product tier ID")

	if err := listCmd.MarkFlagRequired("service-id"); err != nil {
		return
	}
	if err := listCmd.MarkFlagRequired("environment-id"); err != nil {
		return
	}
}

func runList(cmd *cobra.Command, args []string) error {
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

	snapshotType, err := cmd.Flags().GetString("snapshot-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	snapshotType, err = normalizeSnapshotType(snapshotType)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	productTierID, err := cmd.Flags().GetString("product-tier-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
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
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing snapshots...")
		sm.Start()
	}

	result, err := dataaccess.ListAllSnapshots(cmd.Context(), token, serviceID, environmentID, dataaccess.ListAllSnapshotsOptions{
		ProductTierID: productTierID,
		SnapshotType:  snapshotType,
	})
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully listed snapshots")

	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if result == nil || len(result.Snapshots) == 0 {
		utils.PrintInfo("No snapshots found.")
		return nil
	}

	return utils.PrintTextTableJsonArrayOutput(output, formatter.FormatSnapshotSummaries(result.Snapshots))
}

func normalizeSnapshotType(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", strings.ToLower(snapshotTypeManual), "manual":
		return snapshotTypeManual, nil
	case strings.ToLower(snapshotTypeAutomated), "automated":
		return snapshotTypeAutomated, nil
	case snapshotTypeAll:
		return "", nil
	default:
		return "", fmt.Errorf("invalid snapshot type %q (supported: ManualSnapshot, AutomatedSnapshot, all)", raw)
	}
}
