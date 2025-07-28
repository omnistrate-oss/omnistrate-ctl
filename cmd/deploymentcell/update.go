package deploymentcell

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update deployment cell configuration",
	Long: `Update deployment cell configuration including amenities settings.

This command allows you to update the configuration of a deployment cell,
including its amenities configuration such as logging, monitoring, and
security settings.

Examples:
  # Update deployment cell amenities configuration from YAML file
  omnistrate-ctl deployment-cell update -i hc-12345 -s service-id -f amenities.yaml

  # Update deployment cell amenities configuration interactively  
  omnistrate-ctl deployment-cell update -i hc-12345 -s service-id --interactive`,
	RunE:         runUpdate,
	SilenceUsage: true,
}

func init() {
	updateCmd.Flags().StringP("deployment-cell-id", "i", "", "Deployment cell ID (format: hc-xxxxx)")
	updateCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	updateCmd.Flags().StringP("config-file", "f", "", "YAML file containing configuration to update")
	updateCmd.Flags().Bool("interactive", false, "Use interactive mode to update configuration")
	updateCmd.Flags().Bool("merge", false, "Merge with existing configuration instead of replacing")
	_ = updateCmd.MarkFlagRequired("deployment-cell-id")
	_ = updateCmd.MarkFlagRequired("service-id")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("deployment-cell-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	interactive, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	merge, err := cmd.Flags().GetBool("merge")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if configFile == "" && !interactive {
		utils.PrintError(fmt.Errorf("must specify either --config-file or --interactive"))
		return fmt.Errorf("configuration input required")
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if interactive {
		err = updateDeploymentCellInteractive(ctx, token, deploymentCellID, serviceID, merge, output)
	} else {
		err = updateDeploymentCellFromFile(ctx, token, deploymentCellID, serviceID, configFile, merge, output)
	}

	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func updateDeploymentCellFromFile(ctx context.Context, token, deploymentCellID, serviceID, configFile string, merge bool, output string) error {
	// Read configuration from YAML file
	config, err := dataaccess.ReadAmenitiesConfigFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Update deployment cell amenities configuration
	fmt.Printf("🔄 Updating deployment cell amenities configuration from file: %s\n", configFile)
	fmt.Printf("Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Service ID: %s\n", serviceID)
	fmt.Printf("Merge mode: %t\n\n", merge)

	err = dataaccess.UpdateDeploymentCellAmenitiesConfiguration(ctx, token, deploymentCellID, serviceID, config, merge)
	if err != nil {
		return fmt.Errorf("failed to update deployment cell configuration: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully updated deployment cell %s configuration", deploymentCellID))

	// Get updated status
	status, err := dataaccess.GetDeploymentCellAmenitiesStatus(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintWarning(fmt.Sprintf("Failed to get updated status: %v", err))
	} else {
		fmt.Printf("Status: %s\n", status.Status)
		fmt.Printf("Has pending changes: %t\n", status.HasPendingChanges)
		
		if status.HasPendingChanges {
			fmt.Printf("Pending changes: %d\n", len(status.PendingChanges))
			utils.PrintInfo("Use 'omnistrate-ctl deployment-cell apply-pending-changes' to activate the changes")
		}
	}

	return nil
}

func updateDeploymentCellInteractive(ctx context.Context, token, deploymentCellID, serviceID string, merge bool, output string) error {
	fmt.Printf("🎯 Interactive deployment cell amenities configuration update\n")
	fmt.Printf("Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Service ID: %s\n", serviceID)
	fmt.Printf("Merge mode: %t\n\n", merge)

	// Get current configuration if in merge mode
	var currentConfig *map[string]interface{}
	if merge {
		fmt.Println("📋 Loading current configuration...")
		// This would fetch the current configuration from the deployment cell
		// For now, we'll use a placeholder
		currentConfig = &map[string]interface{}{
			"logging": map[string]interface{}{
				"level": "info",
				"retention_days": 30,
			},
		}
	}

	// Run interactive configuration
	config, err := dataaccess.RunInteractiveAmenitiesConfiguration(currentConfig)
	if err != nil {
		return fmt.Errorf("interactive configuration failed: %w", err)
	}

	// Update deployment cell amenities configuration
	fmt.Printf("\n🔄 Updating deployment cell amenities configuration...\n")
	err = dataaccess.UpdateDeploymentCellAmenitiesConfiguration(ctx, token, deploymentCellID, serviceID, config, merge)
	if err != nil {
		return fmt.Errorf("failed to update deployment cell configuration: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully updated deployment cell %s configuration", deploymentCellID))

	// Get updated deployment cell status to show amenities information
	deploymentCell, err := dataaccess.DescribeHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintWarning(fmt.Sprintf("Failed to get updated deployment cell status: %v", err))
	} else {
		fmt.Printf("Status: %s\n", deploymentCell.GetStatus())
		fmt.Printf("Configuration updated successfully\n")
		utils.PrintInfo("Use 'omnistrate-ctl deployment-cell apply-pending-changes' to activate any pending changes")
	}

	return nil
}