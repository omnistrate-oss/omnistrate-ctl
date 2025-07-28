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

var applyPendingChangesCmd = &cobra.Command{
	Use:   "apply-pending-changes",
	Short: "Apply pending changes to deployment cell",
	Long: `Review and confirm the pending configuration changes for deployment cells.

Pending changes are activated and become the live configuration for those cells.
This command allows you to review the pending changes before applying them to 
ensure they are correct.

Examples:
  # Apply pending changes to specific deployment cell
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 -s service-id

  # Apply without confirmation prompt
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 -s service-id --force`,
	RunE:         runApplyPendingChanges,
	SilenceUsage: true,
}

func init() {
	applyPendingChangesCmd.Flags().StringP("deployment-cell-id", "i", "", "Deployment cell ID (format: hc-xxxxx)")
	applyPendingChangesCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	applyPendingChangesCmd.Flags().Bool("force", false, "Skip confirmation prompt and apply changes immediately")
	_ = applyPendingChangesCmd.MarkFlagRequired("deployment-cell-id")
	_ = applyPendingChangesCmd.MarkFlagRequired("service-id")
}

func runApplyPendingChanges(cmd *cobra.Command, args []string) error {
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

	forceFlag, err := cmd.Flags().GetBool("force")
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

	// Get current status to check for pending changes
	status, err := dataaccess.GetDeploymentCellAmenitiesStatus(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get deployment cell status: %w", err))
		return err
	}

	if !status.HasPendingChanges {
		utils.PrintInfo(fmt.Sprintf("No pending changes found for deployment cell %s", deploymentCellID))
		return nil
	}

	// Display pending changes
	fmt.Printf("📋 Pending Changes for Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Service ID: %s\n", serviceID)
	fmt.Printf("Total pending changes: %d\n\n", len(status.PendingChanges))

	fmt.Println("Changes to be applied:")
	for i, change := range status.PendingChanges {
		fmt.Printf("  %d. Path: %s\n", i+1, change.Path)
		
		switch change.Operation {
		case "add":
			fmt.Printf("     Operation: Add\n")
			fmt.Printf("     New Value: %v\n", change.NewValue)
		case "update":
			fmt.Printf("     Operation: Update\n")
			fmt.Printf("     Current Value: %v\n", change.OldValue)
			fmt.Printf("     New Value: %v\n", change.NewValue)
		case "delete":
			fmt.Printf("     Operation: Delete\n")
			fmt.Printf("     Current Value: %v\n", change.OldValue)
		}
		fmt.Println()
	}

	// Confirm if not forced or if there are significant changes
	shouldConfirm := !forceFlag && len(status.PendingChanges) > 0

	if shouldConfirm {
		fmt.Printf("\n⚠️  You are about to apply %d configuration changes to deployment cell %s.\n", len(status.PendingChanges), deploymentCellID)
		fmt.Println("This will modify the live configuration of the deployment cell.")
		
		confirmedChoice, err := prompt.New().Ask("Do you want to proceed with applying these changes?").Choose([]string{"Yes", "No"}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			utils.PrintError(err)
			return err
		}
		
		confirmed := confirmedChoice == "Yes"
		
		if !confirmed {
			utils.PrintInfo("Apply operation cancelled")
			return nil
		}
	}

	// Apply the pending changes using the existing API
	fmt.Printf("🔄 Applying pending changes to deployment cell %s...\n", deploymentCellID)
	
	err = dataaccess.ApplyPendingChangesToDeploymentCell(ctx, token, serviceID, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to apply pending changes: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully applied %d pending changes to deployment cell %s", len(status.PendingChanges), deploymentCellID))

	// Get updated status
	updatedStatus, err := dataaccess.GetDeploymentCellAmenitiesStatus(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintWarning(fmt.Sprintf("Failed to get updated status: %v", err))
		// Don't return error here as the apply operation succeeded
	} else {
		fmt.Printf("Updated status: %s\n", updatedStatus.Status)
		fmt.Printf("Remaining pending changes: %d\n", len(updatedStatus.PendingChanges))
		
		// Print the updated status
		if output == "table" {
			tableView := updatedStatus.ToTableView()
			err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
		} else {
			err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{updatedStatus})
		}
		
		if err != nil {
			utils.PrintWarning(fmt.Sprintf("Failed to print updated status: %v", err))
		}
	}

	// Provide next steps guidance
	fmt.Println("\n📝 Next Steps:")
	fmt.Println("  • Use 'omnistrate-ctl deployment-cell status' to verify the changes")
	fmt.Println("  • Use 'omnistrate-ctl deployment-cell sync' to ensure synchronization")
	
	return nil
}