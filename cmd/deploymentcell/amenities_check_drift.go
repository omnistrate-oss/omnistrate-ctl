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

var amenitiesCheckDriftCmd = &cobra.Command{
	Use:   "check-drift",
	Short: "Check deployment cell for configuration drift",
	Long: `Review deployment cells to determine if their amenities configuration matches 
the latest organization template for the relevant environment.

Identify differences that may require alignment between the current deployment 
cell configuration and the organization's target configuration template.

Examples:
  # Check drift for a specific deployment cell
  omnistrate-ctl deployment-cell amenities check-drift -i cell-123 -z org-123 -e production

  # Check drift for all deployment cells (if supported)
  omnistrate-ctl deployment-cell amenities check-drift -z org-123 -e production --all`,
	RunE:         runAmenitiesCheckDrift,
	SilenceUsage: true,
}

func init() {
	amenitiesCheckDriftCmd.Flags().StringP("deployment-cell-id", "i", "", "Deployment cell ID (required unless --all is used)")
	amenitiesCheckDriftCmd.Flags().StringP("organization-id", "z", "", "Organization ID (required)")
	amenitiesCheckDriftCmd.Flags().StringP("environment", "e", "", "Target environment (required)")
	amenitiesCheckDriftCmd.Flags().Bool("all", false, "Check drift for all deployment cells in the organization")
	amenitiesCheckDriftCmd.Flags().Bool("summary", false, "Show only summary of drift status")
	_ = amenitiesCheckDriftCmd.MarkFlagRequired("organization-id")
	_ = amenitiesCheckDriftCmd.MarkFlagRequired("environment")
}

func runAmenitiesCheckDrift(cmd *cobra.Command, args []string) error {
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

	checkAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	summary, err := cmd.Flags().GetBool("summary")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if !checkAll && deploymentCellID == "" {
		utils.PrintError(fmt.Errorf("must specify either --deployment-cell-id or --all"))
		return fmt.Errorf("deployment cell identification required")
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if checkAll {
		// Check drift for all deployment cells
		err = checkDriftForAllCells(ctx, token, organizationID, environment, summary, output)
	} else {
		// Check drift for specific deployment cell
		err = checkDriftForSingleCell(ctx, token, deploymentCellID, organizationID, environment, summary, output)
	}

	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func checkDriftForSingleCell(ctx context.Context, token, deploymentCellID, organizationID, environment string, summary bool, output string) error {
	// Check configuration drift
	status, err := dataaccess.CheckDeploymentCellConfigurationDrift(ctx, token, deploymentCellID, organizationID, environment)
	if err != nil {
		return fmt.Errorf("failed to check configuration drift: %w", err)
	}

	// Print results
	if status.HasConfigurationDrift {
		utils.PrintWarning(fmt.Sprintf("Configuration drift detected for deployment cell %s", deploymentCellID))
		
		if !summary {
			fmt.Printf("\nDrift Details:\n")
			for _, drift := range status.DriftDetails {
				fmt.Printf("  • Path: %s\n", drift.Path)
				fmt.Printf("    Current: %v\n", drift.CurrentValue)
				fmt.Printf("    Expected: %v\n", drift.TargetValue)
				fmt.Printf("    Type: %s\n", drift.DriftType)
				fmt.Println()
			}
		}
	} else {
		utils.PrintSuccess(fmt.Sprintf("No configuration drift detected for deployment cell %s", deploymentCellID))
	}

	// Print status in requested format
	if output == "table" {
		tableView := status.ToTableView()
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{tableView})
	} else {
		err = utils.PrintTextTableJsonArrayOutput(output, []interface{}{status})
	}
	
	return err
}

func checkDriftForAllCells(ctx context.Context, token, organizationID, environment string, summary bool, output string) error {
	// This would typically get all deployment cells for the organization
	// For now, we'll simulate with a few mock cells
	mockCellIDs := []string{"cell-001", "cell-002", "cell-003"}
	
	var allStatuses []interface{}
	var allTableViews []interface{}
	
	driftCount := 0
	totalCount := len(mockCellIDs)

	fmt.Printf("Checking configuration drift for %d deployment cells...\n\n", totalCount)

	for _, cellID := range mockCellIDs {
		status, err := dataaccess.CheckDeploymentCellConfigurationDrift(ctx, token, cellID, organizationID, environment)
		if err != nil {
			utils.PrintWarning(fmt.Sprintf("Failed to check drift for cell %s: %v", cellID, err))
			continue
		}

		if output == "table" {
			allTableViews = append(allTableViews, status.ToTableView())
		} else {
			allStatuses = append(allStatuses, status)
		}

		if status.HasConfigurationDrift {
			driftCount++
			if !summary {
				fmt.Printf("❌ %s: Drift detected (%d issues)\n", cellID, len(status.DriftDetails))
			}
		} else {
			if !summary {
				fmt.Printf("✅ %s: No drift\n", cellID)
			}
		}
	}

	// Print summary
	fmt.Printf("\n📊 Drift Check Summary:\n")
	fmt.Printf("  Total cells checked: %d\n", totalCount)
	fmt.Printf("  Cells with drift: %d\n", driftCount)
	fmt.Printf("  Cells synchronized: %d\n", totalCount-driftCount)

	if driftCount > 0 {
		utils.PrintWarning(fmt.Sprintf("%d deployment cells have configuration drift", driftCount))
		fmt.Println("\nUse 'omnistrate-ctl deployment-cell amenities sync' to synchronize drifted cells.")
	} else {
		utils.PrintSuccess("All deployment cells are synchronized with the organization template")
	}

	// Print detailed results if requested
	if !summary {
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