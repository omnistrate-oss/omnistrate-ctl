package helm

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	deleteExample = `# Delete a Helm package
omnistrate-ctl helm delete redis --version=20.0.1`
)

var deleteCmd = &cobra.Command{
	Use:          "delete chart --version=[version]",
	Short:        "Delete a Helm package for your service",
	Long:         `This command helps you delete the templates for your helm packages.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Args = cobra.ExactArgs(1) // Require exactly one argument

	deleteCmd.Flags().String("version", "", "Helm Chart version")

	err := deleteCmd.MarkFlagRequired("version")
	if err != nil {
		return
	}
}

func runDelete(cmd *cobra.Command, args []string) error {
	chart := args[0]
	version, _ := cmd.Flags().GetString("version")
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
		msg := "Deleting Helm package..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Delete Helm package
	err = dataaccess.DeleteHelmChart(cmd.Context(), token, chart, version)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted Helm package")

	return nil
}
