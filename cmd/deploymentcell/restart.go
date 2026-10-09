package deploymentcell

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [deployment-cell-id]",
	Short: "Restart a deployment cell",
	Long:  "Restart the deployment of a deployment cell by ID. This command returns once the restart has been requested.",
	Example: `# Restart a deployment cell
omnistrate-ctl deployment-cell restart hc-12345678

# Request a restart with JSON output
omnistrate-ctl deployment-cell restart hc-12345678 --output json`,
	Args:         cobra.ExactArgs(1),
	RunE:         runRestart,
	SilenceUsage: true,
}

func runRestart(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID := args[0]
	if strings.TrimSpace(deploymentCellID) == "" {
		return fmt.Errorf("deployment cell ID is required")
	}

	output, err := cmd.Flags().GetString(common.OutputFlag)
	if err != nil {
		return err
	}
	if output != "text" && output != "table" && output != "json" {
		return fmt.Errorf("unsupported output format: %s", output)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	if err := dataaccess.RestartHostClusterDeployment(cmd.Context(), token, deploymentCellID); err != nil {
		return fmt.Errorf("failed to restart deployment cell %s: %w", deploymentCellID, err)
	}

	return utils.PrintTextTableJsonOutput(output, model.DeploymentCellRestartResult{
		ID:      deploymentCellID,
		Message: "Restart requested",
	})
}
