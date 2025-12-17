package helm

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listInstallationsExample = `# List all Helm Packages and the Kubernetes clusters that they are installed on
omnistrate-ctl helm list-installations --host-cluster-id=[host-cluster-id]`
)

type helmPackageInstallationIntermediate struct {
	ChartName     string
	ChartVersion  string
	ChartRepoName string
	ChartRepoURL  string
	Namespace     string
	HostClusterID string
	Status        string
}

var listInstallationsCmd = &cobra.Command{
	Use:          "list-installations --host-cluster-id=[host-cluster-id]",
	Short:        "List all Helm Packages and the Kubernetes clusters that they are installed on",
	Long:         `This command helps you list all the Helm Packages and the Kubernetes clusters that they are installed on.`,
	Example:      listInstallationsExample,
	RunE:         runListInstallations,
	SilenceUsage: true,
}

func init() {
	saveCmd.Args = cobra.NoArgs // Require no arguments

	listInstallationsCmd.Flags().String("host-cluster-id", "", "Host cluster ID")
}

func runListInstallations(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	hostClusterID, _ := cmd.Flags().GetString("host-cluster-id")
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
		msg := "Listing Helm package installations..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	var helmPackageResult *openapiclientfleet.ListHelmPackageInstallationsResult
	helmPackageResult, err = dataaccess.ListHelmChartInstallations(cmd.Context(), token, hostClusterID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var intermediates []helmPackageInstallationIntermediate
	for _, helmPackageInstallation := range helmPackageResult.HelmPackageInstallations {
		// Convert HelmPackageInstallation to intermediate struct
		intermediate := helmPackageInstallationIntermediate{
			ChartName:     helmPackageInstallation.HelmPackage.ChartName,
			ChartVersion:  helmPackageInstallation.HelmPackage.ChartVersion,
			ChartRepoName: helmPackageInstallation.HelmPackage.ChartRepoName,
			ChartRepoURL:  helmPackageInstallation.HelmPackage.ChartRepoUrl,
			Namespace:     helmPackageInstallation.HelmPackage.Namespace,
			HostClusterID: helmPackageInstallation.HostClusterID,
			Status:        helmPackageInstallation.Status,
		}
		intermediates = append(intermediates, intermediate)
	}

	if len(intermediates) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No Helm package installations found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved Helm package installations")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, intermediates)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
