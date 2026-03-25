package deploymentcell

import (
	"context"
	"fmt"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var deleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "Delete a deployment cell",
	Long:         `Delete a deployment cell by ID.`,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")
	deleteCmd.Flags().StringP("customer-email", "c", "", "Customer email to filter by (required)")
	_ = deleteCmd.MarkFlagRequired("id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	id, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	customerEmail, err := cmd.Flags().GetString("customer-email")
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

	if !force {
		fmt.Printf("Are you sure you want to delete deployment cell '%s'? This action cannot be undone.\n", id)
		fmt.Print("Type 'yes' to confirm: ")
		var confirmation string
		if _, err := fmt.Scanln(&confirmation); err != nil {
			utils.PrintError(fmt.Errorf("failed to read confirmation: %w", err))
			return err
		}
		if confirmation != "yes" {
			fmt.Println("Delete operation cancelled.")
			return nil
		}
	}

	fmt.Printf("Deleting deployment cell: %s\n", id)

	// List all deployment cells to find the one to delete
	hostClusters, err := dataaccess.ListHostClusters(ctx, token, nil, nil)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to list deployment cells: %w", err))
		return err
	}

	// Find the deployment cell by ID / key and customer email
	var hostClusterToDelete openapiclientfleet.HostCluster
	var found bool
	for _, cluster := range hostClusters.GetHostClusters() {
		if (cluster.GetId() == id || cluster.GetKey() == id) && cluster.GetCustomerEmail() == customerEmail {
			hostClusterToDelete = cluster
			found = true
			break
		}
	}

	if !found {
		utils.PrintError(fmt.Errorf("deployment cell with ID '%s' and customer email '%s' not found", id, customerEmail))
		return fmt.Errorf("deployment cell not found")
	}

	err = dataaccess.DeleteHostCluster(ctx, token, hostClusterToDelete.Id)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to delete deployment cell: %w", err))
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Deployment cell '%s' deleted successfully!", id))
	return nil
}
