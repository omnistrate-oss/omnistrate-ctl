package instance

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

func ensureInstanceDebugAccess(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return nil
	}
	reason, err := cmd.Flags().GetString("reason")
	if err != nil {
		return fmt.Errorf("failed to get access reason: %w", err)
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := cmd.Context()
	instanceID := args[0]
	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}
	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to describe resource instance: %w", err)
	}

	deploymentCellIDs := deploymentCellIDsForDebug(instanceData)
	if len(deploymentCellIDs) == 0 {
		return fmt.Errorf("instance %s has no deployment cell", instanceID)
	}
	for _, deploymentCellID := range deploymentCellIDs {
		if _, err = common.EnsureDeploymentCellAccess(
			ctx, token, deploymentCellID, "cluster-admin", reason); err != nil {
			return fmt.Errorf("failed to obtain debug access to deployment cell %s: %w", deploymentCellID, err)
		}
	}
	return nil
}

func deploymentCellIDsForDebug(instance *openapiclientfleet.ResourceInstance) []string {
	if instance == nil {
		return nil
	}
	deploymentCellIDs := make([]string, 0, 2)
	seen := make(map[string]bool, 2)
	for _, deploymentCellID := range []string{
		instance.GetDeploymentCellID(),
		instance.GetControlPlaneDeploymentCellID(),
	} {
		deploymentCellID = strings.TrimSpace(deploymentCellID)
		if deploymentCellID != "" && !seen[deploymentCellID] {
			seen[deploymentCellID] = true
			deploymentCellIDs = append(deploymentCellIDs, deploymentCellID)
		}
	}
	return deploymentCellIDs
}
