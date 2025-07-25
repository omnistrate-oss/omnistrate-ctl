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

var amenitiesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show amenities status for deployment cell",
	Long: `Show the current amenities configuration status for a deployment cell.

This command displays:
- Current amenities configuration status
- Configuration drift information
- Pending changes information
- Last synchronization check time

Examples:
  # Show status for specific deployment cell
  omnistrate-ctl deployment-cell amenities status -i cell-123

  # Show detailed status including configuration details
  omnistrate-ctl deployment-cell amenities status -i cell-123 --detailed

  # Show status for multiple deployment cells
  omnistrate-ctl deployment-cell amenities status -i cell-123,cell-456,cell-789`,
	RunE:         runAmenitiesStatus,
	SilenceUsage: true,
}

func init() {
	amenitiesStatusCmd.Flags().StringP("deployment-cell-id", "i", "", "Deployment cell ID(s) - comma-separated for multiple cells (required)")
	amenitiesStatusCmd.Flags().Bool("detailed", false, "Show detailed configuration information")
	amenitiesStatusCmd.Flags().Bool("show-config", false, "Include current configuration in output")
	_ = amenitiesStatusCmd.MarkFlagRequired("deployment-cell-id")
}

func runAmenitiesStatus(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellIDs, err := cmd.Flags().GetString("deployment-cell-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	detailed, err := cmd.Flags().GetBool("detailed")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	showConfig, err := cmd.Flags().GetBool("show-config")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Parse deployment cell IDs (support comma-separated list)
	cellIDs := utils.ParseCommaSeparatedList(deploymentCellIDs)
	
	if len(cellIDs) == 1 {
		// Single cell status
		err = showSingleCellStatus(ctx, token, cellIDs[0], detailed, showConfig, output)
	} else {
		// Multiple cells status
		err = showMultipleCellsStatus(ctx, token, cellIDs, detailed, showConfig, output)
	}

	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func showSingleCellStatus(ctx context.Context, token, deploymentCellID string, detailed, showConfig bool, output string) error {
	// Get current status
	status, err := dataaccess.GetDeploymentCellAmenitiesStatus(ctx, token, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to get deployment cell status: %w", err)
	}

	// Display status information
	fmt.Printf("📊 Amenities Status for Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Status: %s\n", getStatusEmoji(status.Status))
	fmt.Printf("Last Check: %s\n", status.LastCheck.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Configuration Drift: %s\n", getBooleanEmoji(status.HasConfigurationDrift))
	fmt.Printf("Pending Changes: %s\n", getBooleanEmoji(status.HasPendingChanges))

	if detailed || status.HasConfigurationDrift {
		fmt.Printf("\n🔍 Configuration Drift Details:\n")
		if len(status.DriftDetails) > 0 {
			for i, drift := range status.DriftDetails {
				fmt.Printf("  %d. Path: %s\n", i+1, drift.Path)
				fmt.Printf("     Type: %s\n", drift.DriftType)
				fmt.Printf("     Current: %v\n", drift.CurrentValue)
				fmt.Printf("     Expected: %v\n", drift.TargetValue)
				fmt.Println()
			}
		} else {
			fmt.Printf("  No configuration drift detected.\n")
		}
	}

	if detailed || status.HasPendingChanges {
		fmt.Printf("\n⏳ Pending Changes:\n")
		if len(status.PendingChanges) > 0 {
			for i, change := range status.PendingChanges {
				fmt.Printf("  %d. Path: %s\n", i+1, change.Path)
				fmt.Printf("     Operation: %s\n", change.Operation)
				if change.Operation == "update" {
					fmt.Printf("     From: %v\n", change.OldValue)
					fmt.Printf("     To: %v\n", change.NewValue)
				} else if change.Operation == "add" {
					fmt.Printf("     Value: %v\n", change.NewValue)
				} else if change.Operation == "delete" {
					fmt.Printf("     Current: %v\n", change.OldValue)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("  No pending changes.\n")
		}
	}

	if showConfig {
		fmt.Printf("\n⚙️  Current Configuration:\n")
		if status.CurrentConfiguration != nil {
			configJSON, _ := utils.FormatJSON(status.CurrentConfiguration)
			fmt.Println(configJSON)
		} else {
			fmt.Printf("  Configuration not available.\n")
		}
		
		if status.TargetConfiguration != nil {
			fmt.Printf("\n🎯 Target Configuration:\n")
			targetConfigJSON, _ := utils.FormatJSON(status.TargetConfiguration)
			fmt.Println(targetConfigJSON)
		}
	}

	// Provide recommendations
	fmt.Printf("\n💡 Recommendations:\n")
	if status.HasConfigurationDrift {
		fmt.Printf("  • Run 'omnistrate-ctl deployment-cell amenities sync -i %s' to synchronize configuration\n", deploymentCellID)
	}
	if status.HasPendingChanges {
		fmt.Printf("  • Run 'omnistrate-ctl deployment-cell amenities apply -i %s' to apply pending changes\n", deploymentCellID)
	}
	if !status.HasConfigurationDrift && !status.HasPendingChanges {
		fmt.Printf("  • Configuration is synchronized and up-to-date\n")
	}

	// Print in requested output format
	if output != "text" {
		fmt.Printf("\nStructured Output:\n")
		if output == "table" {
			tableView := status.ToTableView()
			err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
		} else {
			err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{status})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func showMultipleCellsStatus(ctx context.Context, token string, cellIDs []string, detailed, showConfig bool, output string) error {
	var allStatuses []interface{}
	var allTableViews []interface{}
	
	fmt.Printf("📊 Amenities Status Summary (%d deployment cells)\n\n", len(cellIDs))

	driftCount := 0
	pendingCount := 0
	errorCount := 0

	for _, cellID := range cellIDs {
		status, err := dataaccess.GetDeploymentCellAmenitiesStatus(ctx, token, cellID)
		if err != nil {
			fmt.Printf("❌ %s: Failed to get status (%v)\n", cellID, err)
			errorCount++
			continue
		}

		// Count statuses
		if status.HasConfigurationDrift {
			driftCount++
		}
		if status.HasPendingChanges {
			pendingCount++
		}

		// Display basic status
		statusIcon := getStatusEmoji(status.Status)
		driftIcon := getBooleanEmoji(status.HasConfigurationDrift)
		pendingIcon := getBooleanEmoji(status.HasPendingChanges)
		
		fmt.Printf("%s %s: %s | Drift: %s | Pending: %s\n", 
			statusIcon, cellID, status.Status, driftIcon, pendingIcon)

		if detailed {
			if status.HasConfigurationDrift {
				fmt.Printf("    Drift items: %d\n", len(status.DriftDetails))
			}
			if status.HasPendingChanges {
				fmt.Printf("    Pending changes: %d\n", len(status.PendingChanges))
			}
			fmt.Printf("    Last check: %s\n", status.LastCheck.Format("2006-01-02 15:04:05"))
		}

		// Collect for structured output
		if output == "table" {
			allTableViews = append(allTableViews, status.ToTableView())
		} else {
			allStatuses = append(allStatuses, status)
		}
	}

	// Print summary
	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("  Total cells: %d\n", len(cellIDs))
	fmt.Printf("  With drift: %d\n", driftCount)
	fmt.Printf("  With pending changes: %d\n", pendingCount)
	fmt.Printf("  Errors: %d\n", errorCount)

	syncNeeded := len(cellIDs) - errorCount - (len(cellIDs) - driftCount - pendingCount)
	if syncNeeded > 0 {
		fmt.Printf("  Needing attention: %d\n", syncNeeded)
	}

	// Provide bulk recommendations
	if driftCount > 0 || pendingCount > 0 {
		fmt.Printf("\n💡 Bulk Operations:\n")
		if driftCount > 0 {
			fmt.Printf("  • Run 'omnistrate-ctl deployment-cell amenities sync --all --drift-only' to sync all drifted cells\n")
		}
		if pendingCount > 0 {
			fmt.Printf("  • Apply pending changes individually using the 'apply' command\n")
		}
	}

	// Print structured output
	if output != "text" {
		fmt.Printf("\nDetailed Results:\n")
		if output == "table" && len(allTableViews) > 0 {
			err := utils.PrintTextTableJsonArrayOutput(output, allTableViews)
			if err != nil {
				return err
			}
		} else if len(allStatuses) > 0 {
			err := utils.PrintTextTableJsonArrayOutput(output, allStatuses)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getStatusEmoji(status string) string {
	switch status {
	case "synchronized":
		return "✅ " + status
	case "drift_detected":
		return "⚠️  " + status
	case "pending_changes":
		return "⏳ " + status
	case "error":
		return "❌ " + status
	default:
		return "❓ " + status
	}
}

func getBooleanEmoji(value bool) string {
	if value {
		return "❌ Yes"
	}
	return "✅ No"
}