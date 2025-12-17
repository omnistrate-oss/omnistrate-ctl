package helm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	saveExample = `# Install the Redis Operator Helm Chart
omnistrate-ctl helm save redis --repo-url=https://charts.bitnami.com/bitnami --version=20.0.1 --namespace=redis-operator`
)

var saveCmd = &cobra.Command{
	Use:          "save chart --repo-name=[repo-name] --repo-url=[repo-url] --version=[version] --namespace=[namespace] --values-file=[values-file]",
	Short:        "Save a Helm Chart for your service",
	Long:         `This command helps you save the templates for your helm charts.`,
	Example:      saveExample,
	RunE:         runSave,
	SilenceUsage: true,
}

func init() {
	saveCmd.Args = cobra.ExactArgs(1) // Require exactly one argument

	saveCmd.Flags().String("repo-name", "", "Helm Chart repository name")
	saveCmd.Flags().String("repo-url", "", "Helm Chart repository URL")
	saveCmd.Flags().String("version", "", "Helm Chart version")
	saveCmd.Flags().String("namespace", "", "Helm Chart namespace")
	saveCmd.Flags().String("values-file", "", "Helm Chart values file containing custom values defined as a JSON")

	err := saveCmd.MarkFlagRequired("repo-url")
	if err != nil {
		return
	}

	err = saveCmd.MarkFlagRequired("version")
	if err != nil {
		return
	}

	err = saveCmd.MarkFlagRequired("namespace")
	if err != nil {
		return
	}
}

func runSave(cmd *cobra.Command, args []string) error {
	// Get flags
	chart := args[0]
	repoName, _ := cmd.Flags().GetString("repo-name")
	repoURL, _ := cmd.Flags().GetString("repo-url")
	version, _ := cmd.Flags().GetString("version")
	namespace, _ := cmd.Flags().GetString("namespace")
	valuesFile, _ := cmd.Flags().GetString("values-file")
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
		msg := "Saving Helm Chart..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	var values map[string]any
	if len(valuesFile) > 0 {
		// Read Values file as a JSON
		if _, err = os.Stat(valuesFile); os.IsNotExist(err) {
			err = fmt.Errorf("can't load values file from non existent path: %s", valuesFile)
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		var data []byte
		if data, err = os.ReadFile(valuesFile); err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if err = json.Unmarshal(data, &values); err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Save Helm Chart
	helmPackage, err := dataaccess.SaveHelmChart(cmd.Context(), token, chart, version, namespace, repoName, repoURL, values)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully saved Helm Chart")

	// Print output
	err = utils.PrintTextTableJsonOutput(output, helmPackage)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
