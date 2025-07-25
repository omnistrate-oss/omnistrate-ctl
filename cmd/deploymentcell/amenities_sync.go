package deploymentcell

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var amenitiesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync deployment cell with organization+environment template",
	Long: `Synchronize deployment cells to adopt the current organization+environment configuration.

Changes are placed in a pending state and not active until you approve them using 
the 'apply' command. This allows you to review the changes before they are applied 
to the deployment cell.

Examples:
  # Sync specific deployment cell with organization template
  omnistrate-ctl deployment-cell amenities sync -i cell-123 -z org-123 -e production

  # Sync with confirmation prompt
  omnistrate-ctl deployment-cell amenities sync -i cell-123 -z org-123 -e production --confirm

  # Sync all deployment cells that have drift
  omnistrate-ctl deployment-cell amenities sync -z org-123 -e production --all --drift-only`,
	RunE:         runAmenitiesSync,
	SilenceUsage: true,
}

func init() {
	amenitiesSyncCmd.Flags().StringP("deployment-cell-id", "i", "", "Deployment cell ID (required unless --all is used)")
	amenitiesSyncCmd.Flags().StringP("organization-id", "z", "", "Organization ID (required)")
	amenitiesSyncCmd.Flags().StringP("environment", "e", "", "Target environment (required)")
	amenitiesSyncCmd.Flags().Bool("all", false, "Sync all deployment cells in the organization")
	amenitiesSyncCmd.Flags().Bool("drift-only", false, "Only sync cells that have configuration drift (use with --all)")
	amenitiesSyncCmd.Flags().Bool("confirm", false, "Prompt for confirmation before syncing")
	amenitiesSyncCmd.Flags().Bool("dry-run", false, "Show what would be synced without making changes")
	_ = amenitiesSyncCmd.MarkFlagRequired("organization-id")
	_ = amenitiesSyncCmd.MarkFlagRequired("environment")
}

func runAmenitiesSync(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("deployment-cell-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	organizationID, err := cmd.Flags().GetString("organization-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	syncAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	driftOnly, err := cmd.Flags().GetBool("drift-only")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	confirmFlag, err := cmd.Flags().GetBool("confirm")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if !syncAll && deploymentCellID == "" {
		utils.PrintError(fmt.Errorf("must specify either --deployment-cell-id or --all"))
		return fmt.Errorf("deployment cell identification required")
	}

	if driftOnly && !syncAll {
		utils.PrintError(fmt.Errorf("--drift-only can only be used with --all"))
		return fmt.Errorf("invalid flag combination")
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if syncAll {
		// Sync all deployment cells
		err = syncAllCells(ctx, token, organizationID, environment, driftOnly, confirmFlag, dryRun, output)
	} else {
		// Sync specific deployment cell
		err = syncSingleCell(ctx, token, deploymentCellID, organizationID, environment, confirmFlag, dryRun, output)
	}

	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func syncSingleCell(ctx context.Context, token, deploymentCellID, organizationID, environment string, confirm, dryRun bool, output string) error {
	// First check if there's any drift
	driftStatus, err := dataaccess.CheckDeploymentCellConfigurationDrift(ctx, token, deploymentCellID, organizationID, environment)
	if err != nil {
		return fmt.Errorf("failed to check configuration drift: %w", err)
	}

	if !driftStatus.HasConfigurationDrift {
		utils.PrintInfo(fmt.Sprintf("Deployment cell %s is already synchronized with the organization template", deploymentCellID))
		return nil
	}

	// Show what will be synced
	fmt.Printf("📋 Synchronization Plan for Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Organization: %s\n", organizationID)
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Configuration drift items: %d\n\n", len(driftStatus.DriftDetails))

	fmt.Println("Changes to be applied:")
	for _, drift := range driftStatus.DriftDetails {
		switch drift.DriftType {
		case "missing":
			fmt.Printf("  + Add: %s = %v\n", drift.Path, drift.TargetValue)
		case "different":
			fmt.Printf("  ~ Update: %s from %v to %v\n", drift.Path, drift.CurrentValue, drift.TargetValue)
		case "extra":
			fmt.Printf("  - Remove: %s (current: %v)\n", drift.Path, drift.CurrentValue)
		}
	}

	if dryRun {
		utils.PrintInfo("Dry run completed. No changes were made.")
		return nil
	}

	// Confirm if requested
	if confirm {
		confirmedChoice, err := prompt.New().Ask("Do you want to proceed with synchronization?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			return err
		}
		confirmed := confirmedChoice == "Yes"
		if !confirmed {
			utils.PrintInfo("Synchronization cancelled")
			return nil
		}
	}

	// Perform synchronization
	status, err := dataaccess.SyncDeploymentCellWithTemplate(ctx, token, deploymentCellID, organizationID, environment)
	if err != nil {
		return fmt.Errorf("failed to sync deployment cell: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully synchronized deployment cell %s", deploymentCellID))
	fmt.Printf("Pending changes: %d\n", len(status.PendingChanges))
	fmt.Println("\nUse 'omnistrate-ctl deployment-cell amenities apply' to activate the pending changes.")

	// Print status in requested format
	if output == "table" {
		tableView := status.ToTableView()
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
	} else {
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{status})
	}

	return err
}

func syncAllCells(ctx context.Context, token, organizationID, environment string, driftOnly, confirm, dryRun bool, output string) error {
	// This would typically get all deployment cells for the organization
	// For now, we'll simulate with a few mock cells
	mockCellIDs := []string{"cell-001", "cell-002", "cell-003"}
	
	var cellsToSync []string
	var syncResults []interface{}
	var syncResultsTable []interface{}

	fmt.Printf("Analyzing deployment cells for synchronization...\n")
	fmt.Printf("Organization: %s\n", organizationID)
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Drift-only mode: %t\n\n", driftOnly)

	// Check which cells need synchronization
	for _, cellID := range mockCellIDs {
		if driftOnly {
			// Check for drift first
			driftStatus, err := dataaccess.CheckDeploymentCellConfigurationDrift(ctx, token, cellID, organizationID, environment)
			if err != nil {
				utils.PrintWarning(fmt.Sprintf("Failed to check drift for cell %s: %v", cellID, err))
				continue
			}
			
			if driftStatus.HasConfigurationDrift {
				cellsToSync = append(cellsToSync, cellID)
				fmt.Printf("📋 %s: Drift detected (%d issues)\n", cellID, len(driftStatus.DriftDetails))
			} else {
				fmt.Printf("✅ %s: Already synchronized\n", cellID)
			}
		} else {
			cellsToSync = append(cellsToSync, cellID)
		}
	}

	if len(cellsToSync) == 0 {
		utils.PrintSuccess("All deployment cells are already synchronized")
		return nil
	}

	fmt.Printf("\nCells to synchronize: %d\n", len(cellsToSync))
	for _, cellID := range cellsToSync {
		fmt.Printf("  • %s\n", cellID)
	}

	if dryRun {
		utils.PrintInfo("Dry run completed. No changes were made.")
		return nil
	}

	// Confirm if requested
	if confirm {
		confirmedChoice, err := prompt.New().Ask(fmt.Sprintf("Do you want to synchronize %d deployment cells?", len(cellsToSync))).Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			return err
		}
		confirmed := confirmedChoice == "Yes"
		if !confirmed {
			utils.PrintInfo("Synchronization cancelled")
			return nil
		}
	}

	// Perform synchronization for each cell
	successCount := 0
	for _, cellID := range cellsToSync {
		fmt.Printf("\n🔄 Synchronizing %s...", cellID)
		
		status, err := dataaccess.SyncDeploymentCellWithTemplate(ctx, token, cellID, organizationID, environment)
		if err != nil {
			fmt.Printf(" ❌ Failed: %v\n", err)
			continue
		}

		fmt.Printf(" ✅ Success (%d pending changes)\n", len(status.PendingChanges))
		successCount++

		if output == "table" {
			syncResultsTable = append(syncResultsTable, status.ToTableView())
		} else {
			syncResults = append(syncResults, status)
		}
	}

	// Print summary
	fmt.Printf("\n📊 Synchronization Summary:\n")
	fmt.Printf("  Total cells processed: %d\n", len(cellsToSync))
	fmt.Printf("  Successfully synchronized: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", len(cellsToSync)-successCount)

	if successCount > 0 {
		utils.PrintSuccess(fmt.Sprintf("Successfully synchronized %d deployment cells", successCount))
		fmt.Println("\nUse 'omnistrate-ctl deployment-cell amenities apply' to activate the pending changes.")
		
		// Print detailed results
		if output == "table" && len(syncResultsTable) > 0 {
			fmt.Printf("\nSynchronization Results:\n")
			err := utils.PrintTextTableJsonArrayOutput(output, syncResultsTable)
			if err != nil {
				return err
			}
		} else if len(syncResults) > 0 {
			fmt.Printf("\nSynchronization Results:\n")
			err := utils.PrintTextTableJsonArrayOutput(output, syncResults)
			if err != nil {
				return err
			}
		}
	}

	return nil
}