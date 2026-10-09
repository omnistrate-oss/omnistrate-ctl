package deploymentcell

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const restartExample = `# Restart a deployment cell deployment
omnistrate-ctl deployment-cell restart hc-12345

# Request a restart with JSON output
omnistrate-ctl deployment-cell restart hc-12345 --output json`

var restartCmd = &cobra.Command{
	Use:          "restart [deployment-cell-id]",
	Short:        "Restart a deployment cell deployment",
	Long:         `Request a deployment restart for a deployment cell by its host cluster ID (hc-xxxxx). This command returns when the restart request is accepted; it does not wait for the deployment to become healthy.`,
	Example:      restartExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runRestart,
	SilenceUsage: true,
}

func runRestart(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID := strings.TrimSpace(args[0])
	if deploymentCellID == "" {
		err := fmt.Errorf("deployment cell ID cannot be empty")
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if output != "table" && output != "text" && output != "json" {
		err := fmt.Errorf("unsupported output format: %s", output)
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
		spinner = sm.AddSpinner("Requesting deployment cell restart...")
		sm.Start()
	}

	if err := dataaccess.RestartHostClusterDeployment(cmd.Context(), token, deploymentCellID); err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Deployment cell restart requested successfully")
	return utils.PrintTextTableJsonOutput(output, model.DeploymentCellRestartResult{
		ID:      deploymentCellID,
		Message: "Deployment restart requested successfully",
	})
}
