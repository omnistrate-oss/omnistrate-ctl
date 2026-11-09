package deploymentcell

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [deployment-cell-id-or-key]",
	Short: "Describe a deployment cell",
	Long:  "Describe a deployment cell by ID or key, including its current and effective managed artifact versions.",
	Example: `# Describe a deployment cell
omnistrate-ctl deployment-cell describe hc-12345678

# Describe an adopted deployment cell by key
omnistrate-ctl deployment-cell describe my-adopted-cell

# Describe a deployment cell with JSON output
omnistrate-ctl deployment-cell describe hc-12345678 --output json`,
	Args:         cobra.ExactArgs(1),
	RunE:         runDescribe,
	SilenceUsage: true,
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellIdentifier := strings.TrimSpace(args[0])
	if deploymentCellIdentifier == "" {
		return fmt.Errorf("deployment cell ID or key is required")
	}

	output, err := cmd.Flags().GetString(common.OutputFlag)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	hostClusters, err := dataaccess.ListHostClusters(cmd.Context(), token, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve deployment cell %s: %w", deploymentCellIdentifier, err)
	}
	deploymentCellID := resolveDeploymentCellID(deploymentCellIdentifier, hostClusters.GetHostClusters())

	hostCluster, err := dataaccess.DescribeHostCluster(cmd.Context(), token, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to describe deployment cell %s: %w", deploymentCellIdentifier, err)
	}

	deploymentCell := formatDeploymentCell(hostCluster)
	if output == "table" {
		return utils.PrintTextTableJsonOutput(output, deploymentCell.ToDescribeTableView())
	}

	return utils.PrintTextTableJsonOutput(output, deploymentCell)
}

func resolveDeploymentCellID(identifier string, hostClusters []openapiclientfleet.HostCluster) string {
	for _, hostCluster := range hostClusters {
		if hostCluster.GetId() == identifier || hostCluster.GetKey() == identifier {
			return hostCluster.GetId()
		}
	}

	return identifier
}
