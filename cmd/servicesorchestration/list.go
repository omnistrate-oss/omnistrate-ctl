package servicesorchestration

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List services orchestration deployments of the service postgres in the prod and dev environments
omctl services-orchestration list --environment-type=prod`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

var listCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List services orchestration deployments",
	Long:         `This command helps you list services orchestration deployments.`,
	Deprecated:   "This command is deprecated and will be removed in a future release.",
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringP("environment-type", "", "dev", "Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private'")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	environmentType, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != common.OutputTypeJson {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing services orchestration deployments...")
		sm.Start()
	}

	// Get all instances
	searchRes, err := dataaccess.ListServicesOrchestration(cmd.Context(), token, environmentType)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(searchRes) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No services orchestration deployments found.")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, searchRes)
	if err != nil {
		return err
	}

	return nil
}
