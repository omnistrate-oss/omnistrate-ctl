package deploymentcell

import (
	"context"
	"fmt"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
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
	applyPendingChangesCmd.Flags().StringP("id", "i", "", "Deployment cell ID (format: hc-xxxxx)")
	applyPendingChangesCmd.Flags().Bool("force", false, "Skip confirmation prompt and apply changes immediately")
	_ = applyPendingChangesCmd.MarkFlagRequired("id")
}

func runApplyPendingChanges(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	forceFlag, err := cmd.Flags().GetBool("force")
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

	var hc *openapiclientfleet.HostCluster
	if hc, err = dataaccess.DescribeHostCluster(ctx, token, deploymentCellID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Display pending changes
	fmt.Printf("📋 Pending Changes for Deployment Cell: %s\n", deploymentCellID)
	fmt.Printf("Total pending changes: %v\n", hc.GetPendingAmenities())

	// Confirm if not forced or if there are significant changes
	shouldConfirm := !forceFlag

	if shouldConfirm {
		fmt.Printf("\n⚠️  You are about to apply the following pending changes to deployment cell %s:\n", deploymentCellID)
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

	err = dataaccess.ApplyPendingChangesToHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to apply pending changes: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully applied pending changes to deployment cell %s", deploymentCellID))

	return nil
}
