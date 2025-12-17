package helm

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List all Helm packages that are saved
omnistrate-ctl helm list`
)

var listCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List all Helm packages that are saved",
	Long:         `This command helps you list all the Helm packages that are saved.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func runList(cmd *cobra.Command, args []string) error {
	// Get flags
	output, _ := cmd.Flags().GetString("output")

	// Validate user is currently logged in
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
		msg := "Listing Helm packages..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Retrieve Helm packages
	var helmPackageResult *openapiclient.ListHelmPackagesResult
	helmPackageResult, err = dataaccess.ListHelmCharts(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if len(helmPackageResult.HelmPackages) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No Helm packages found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved Helm packages")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, helmPackageResult.HelmPackages)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
