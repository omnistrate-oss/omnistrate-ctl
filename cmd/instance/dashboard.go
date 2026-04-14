package instance

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const dashboardExample = `# Open the interactive dashboard TUI for an instance
omnistrate-ctl instance dashboard instance-abcd1234

# Get raw dashboard metadata as JSON
omnistrate-ctl instance dashboard instance-abcd1234 -o json`

var dashboardCmd = &cobra.Command{
	Use:          "dashboard [instance-id]",
	Short:        "Get Grafana dashboard access details for an instance",
	Long:         `This command opens an interactive dashboard TUI with customer and internal metrics views when metrics are enabled. Use -o json for raw metadata.`,
	Example:      dashboardExample,
	RunE:         runDashboard,
	SilenceUsage: true,
}

func init() {
	dashboardCmd.Args = cobra.ExactArgs(1)
	dashboardCmd.Flags().StringP("output", "o", "text", "Output format (text|table|json).")
}

func runDashboard(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" && output != "text" && output != "table" {
		err = errors.New("unsupported output format")
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
		spinner = sm.AddSpinner("Fetching instance dashboard details...")
		sm.Start()
	}

	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	instance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	dashboardCatalog, err := dataaccess.NewDashboardService().GetDashboardCatalog(instance)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if dashboardCatalog.InstanceID == "" {
		dashboardCatalog.InstanceID = instanceID
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved instance dashboard details")

	if output == "json" {
		if err = utils.PrintTextTableJsonOutput(output, dashboardCatalog); err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	if err = printDashboardTUI(dashboardCatalog); err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
