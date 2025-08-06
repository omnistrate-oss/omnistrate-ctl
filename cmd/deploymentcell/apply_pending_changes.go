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
	Short: "Apply pending configuration changes to a deployment cell",
	Long: `Apply pending configuration changes to a deployment cell.

When you update a deployment cell's configuration template, the changes are initially 
stored as "pending changes" and do not take effect immediately. This command reviews 
and applies those pending changes to make them active in the deployment cell.

This is useful for:
- Activating configuration changes made through update-config-template
- Reviewing pending changes before they take effect
- Confirming configuration updates in a controlled manner

Examples:
  # Apply pending changes to specific deployment cell
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345

  # Apply without confirmation prompt
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 --force`,
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
	fmt.Printf("Pending Changes for Deployment Cell: %s\n", deploymentCellID)

	pendingChanges := hc.GetPendingAmenities()
	if len(pendingChanges) == 0 {
		utils.PrintSuccess("No pending changes found.")
		utils.PrintInfo("Deployment cell is already up to date")
		return nil
	}

	fmt.Printf("Total pending changes: %d amenities\n\n", len(pendingChanges))

	// Display pending changes in a readable format
	// Separate amenities into managed and custom categories
	var managedAmenities []openapiclientfleet.Amenity
	var customAmenities []openapiclientfleet.Amenity

	for _, amenity := range pendingChanges {
		if amenity.GetIsManaged() {
			managedAmenities = append(managedAmenities, amenity)
		} else {
			customAmenities = append(customAmenities, amenity)
		}
	}

	// Display managed amenities section
	if len(managedAmenities) > 0 {
		fmt.Printf("Managed Amenities (%d):\n", len(managedAmenities))
		for i, amenity := range managedAmenities {
			fmt.Printf("  %d. Name: %s\n", i+1, amenity.GetName())
			if amenity.GetDescription() != "" {
				fmt.Printf("     Description: %s\n", amenity.GetDescription())
			}
			if amenity.GetType() != "" {
				fmt.Printf("     Type: %s\n", amenity.GetType())
			}
		}
	}

	// Display custom amenities section
	if len(customAmenities) > 0 {
		fmt.Printf("Custom Amenities (%d):\n", len(customAmenities))
		for i, amenity := range customAmenities {
			fmt.Printf("  %d. Name: %s\n", i+1, amenity.GetName())
			if amenity.GetDescription() != "" {
				fmt.Printf("     Description: %s\n", amenity.GetDescription())
			}
			if amenity.GetType() != "" {
				fmt.Printf("     Type: %s\n", amenity.GetType())
			}

			// Display properties if available
			if amenity.Properties != nil {
				fmt.Printf("     Properties:\n")
				for key, value := range amenity.Properties {
					fmt.Printf("       %s: %v\n", key, value)
				}
			}
			fmt.Println()
		}
	}

	// Confirm if not forced or if there are significant changes
	shouldConfirm := !forceFlag

	if shouldConfirm {
		utils.PrintWarning(fmt.Sprintf("You are about to apply the above pending changes to deployment cell %s", deploymentCellID))
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
	utils.PrintInfo(fmt.Sprintf("Applying pending changes to deployment cell %s...", deploymentCellID))

	err = dataaccess.ApplyPendingChangesToHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		return fmt.Errorf("failed to apply pending changes: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully applied pending changes to deployment cell %s", deploymentCellID))

	return nil
}
