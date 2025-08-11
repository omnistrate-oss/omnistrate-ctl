package deploymentcell

import (
	"context"
	"fmt"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var updateTemplateCmd = &cobra.Command{
	Use:   "update-config-template",
	Short: "Update deployment cell configuration template",
	Long: `Update the deployment cell configuration template for your organization or a specific deployment cell.

This command allows you to:
1. Update the organization-level template that applies to new deployment cells
2. Update configuration for a specific deployment cell
3. Sync a deployment cell with the organization template

When updating the organization template, you must specify the environment and cloud provider.
When updating a specific deployment cell, provide the deployment cell ID as an argument or use the --id flag.

Examples:
  # Update organization template for PROD environment and AWS
  omnistrate-ctl deployment-cell update-config-template -e PROD --cloud aws -f template-aws.yaml

  # Update specific deployment cell with configuration file using flag
  omnistrate-ctl deployment-cell update-config-template --id hc-12345 -f deployment-cell-config.yaml

  # Sync deployment cell with organization template
  omnistrate-ctl deployment-cell update-config-template --id hc-12345 --sync-with-template`,
	RunE:         runUpdateTemplate,
	SilenceUsage: true,
}

func init() {
	updateTemplateCmd.Flags().StringP("environment", "e", "", "Environment type (e.g., PROD, STAGING) - required for organization template updates")
	updateTemplateCmd.Flags().StringP("cloud", "c", "", "Cloud provider (aws, azure, gcp) - required for organization template updates")
	updateTemplateCmd.Flags().StringP("file", "f", "", "Configuration file path (YAML format)")
	updateTemplateCmd.Flags().StringP("id", "i", "", "Deployment cell ID")
	updateTemplateCmd.Flags().Bool("sync-with-template", false, "Sync deployment cell with organization template")
}

func runUpdateTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	cloudProvider, err := cmd.Flags().GetString("cloud")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	configFile, err := cmd.Flags().GetString("file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// ID from flag
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Check for ID as positional argument if not provided via flag
	if id == "" && len(args) > 0 {
		id = args[0]
	}

	syncWithTemplate, err := cmd.Flags().GetBool("sync-with-template")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if syncWithTemplate && configFile != "" {
		utils.PrintError(fmt.Errorf("cannot use --sync-with-template with a configuration file"))
		return fmt.Errorf("invalid arguments")
	}

	// Cannot specify both environment/cloud and deployment cell ID
	if id != "" && (environment != "" || cloudProvider != "") {
		utils.PrintError(fmt.Errorf("cannot specify both deployment cell ID and environment/cloud provider"))
		return fmt.Errorf("invalid arguments")
	}

	// Validate environment if provided
	if environment != "" {
		if !isValidEnvironment(environment) {
			utils.PrintError(fmt.Errorf("invalid environment '%s'. Valid values are: %v", environment, validEnvironments))
			return fmt.Errorf("invalid environment type")
		}
	}

	// Validate cloud provider if provided
	if cloudProvider != "" {
		if !isValidCloudProvider(cloudProvider) {
			utils.PrintError(fmt.Errorf("invalid cloud provider '%s'. Valid values are: aws, azure, gcp", cloudProvider))
			return fmt.Errorf("invalid cloud provider")
		}
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Check if deployment cell ID is provided
	if id != "" {
		fmt.Printf("Deployment Cell Configuration Update\n")
		fmt.Printf("Deployment Cell ID: %s\n", id)
		fmt.Printf("Configuration File: %s\n\n", configFile)

		utils.PrintInfo("Updating deployment cell configuration...")
		return updateDeploymentCellConfiguration(ctx, token, id, configFile, syncWithTemplate)
	}

	// Update organization template
	return updateOrganizationTemplate(ctx, token, environment, cloudProvider, configFile)
}

func updateOrganizationTemplate(ctx context.Context, token string, environment string, cloudProvider string, configFile string) error {
	// Validate required flags
	if environment == "" {
		err := fmt.Errorf("environment flag is required for organization template updates")
		utils.PrintError(err)
		return err
	}

	if cloudProvider == "" {
		err := fmt.Errorf("cloud flag is required for organization template updates")
		utils.PrintError(err)
		return err
	}

	if configFile == "" {
		err := fmt.Errorf("configuration file is required for organization template updates")
		utils.PrintError(err)
		return err
	}

	// Read and parse configuration file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to read configuration file %s: %w", configFile, err))
		return err
	}

	// Parse as DeploymentCellTemplate directly (no cloud provider wrapper)
	var templateConfig model.DeploymentCellTemplate
	err = yaml.Unmarshal(configData, &templateConfig)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to parse configuration file %s: %w", configFile, err))
		return err
	}

	err = dataaccess.UpdateServiceProviderOrganization(ctx, token, templateConfig, environment, cloudProvider)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to update organization template: %w", err))
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully updated organization template for environment '%s' and cloud provider '%s'", environment, cloudProvider))

	return nil
}

func updateDeploymentCellConfiguration(ctx context.Context, token string, deploymentCellID string, configFile string, syncWithTemplate bool) error {
	if syncWithTemplate {
		return syncDeploymentCellWithTemplate(ctx, token, deploymentCellID)
	}

	if configFile == "" {
		err := fmt.Errorf("configuration file is required when not using --sync-with-template")
		utils.PrintError(err)
		return err
	}

	return updateDeploymentCellFromFile(ctx, token, deploymentCellID, configFile)
}

func syncDeploymentCellWithTemplate(ctx context.Context, token string, deploymentCellID string) error {
	// Get current deployment cell state before sync
	fmt.Printf("Checking current deployment cell configuration...\n")
	currentHc, err := dataaccess.DescribeHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get current deployment cell details: %w", err))
		return err
	}

	// Count current amenities
	currentManagedCount := 0
	currentCustomCount := 0
	if currentHc.Amenities != nil {
		for _, amenity := range currentHc.Amenities {
			if amenity.IsManaged != nil && *amenity.IsManaged {
				currentManagedCount++
			} else {
				currentCustomCount++
			}
		}
	}

	fmt.Printf("Current configuration: %d amenities (%d managed, %d custom)\n",
		currentManagedCount+currentCustomCount, currentManagedCount, currentCustomCount)

	envType := currentHc.GetEnvironmentType()
	if envType == "" {
		utils.PrintError(fmt.Errorf("deployment cell %s does not have an environment type set",
			deploymentCellID))
		return fmt.Errorf("missing environment type for deployment cell %s", deploymentCellID)
	}

	// Perform sync with organization template
	fmt.Printf("Syncing deployment cell with organization template for environment '%s'...\n", envType)
	err = dataaccess.UpdateHostCluster(ctx, token, deploymentCellID, nil, utils.ToPtr(true))
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to sync deployment cell with organization template: %w", err))
		return err
	}

	// Get updated deployment cell state after sync
	fmt.Printf("Checking updated deployment cell configuration...\n")
	updatedHc, err := dataaccess.DescribeHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get updated deployment cell details: %w", err))
		return err
	}

	// Count updated amenities
	updatedManagedCount := 0
	updatedCustomCount := 0
	if updatedHc.PendingAmenities != nil {
		for _, amenity := range updatedHc.PendingAmenities {
			if amenity.IsManaged != nil && *amenity.IsManaged {
				updatedManagedCount++
			} else {
				updatedCustomCount++
			}
		}
	}

	// Show the results
	totalBefore := currentManagedCount + currentCustomCount
	totalAfter := updatedManagedCount + updatedCustomCount

	if totalAfter == 0 && updatedManagedCount == 0 && updatedCustomCount == 0 {
		fmt.Printf("Deployment cell is already synchronized with organization template\n")
		fmt.Printf("Configuration remains: %d amenities (%d managed, %d custom)\n",
			totalBefore, currentManagedCount, currentCustomCount)
	} else {
		fmt.Printf("Successfully synchronized deployment cell with organization template\n")
		fmt.Printf("Configuration updated: %d → %d amenities\n", totalBefore, totalAfter)
		fmt.Printf("   Managed amenities: %d → %d\n", currentManagedCount, updatedManagedCount)
		fmt.Printf("   Custom amenities: %d → %d\n", currentCustomCount, updatedCustomCount)
	}

	utils.PrintSuccess(fmt.Sprintf("Deployment cell %s successfully synchronized with organization template", deploymentCellID))

	return nil
}

func updateDeploymentCellFromFile(ctx context.Context, token string, deploymentCellID string, configFile string) error {
	// Read and parse configuration file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to read configuration file %s: %w", configFile, err))
		return err
	}

	// Handle empty file case - should remove all amenities
	var templateConfig model.DeploymentCellTemplate
	if len(configData) == 0 || string(configData) == "" {
		// Empty file means remove all amenities
		fmt.Printf("Configuration file is empty - removing all amenities from deployment cell\n")
		templateConfig = model.DeploymentCellTemplate{
			ManagedAmenities: []model.Amenity{},
			CustomAmenities:  []model.Amenity{},
		}
	} else {
		// Parse as DeploymentCellTemplate directly (no cloud provider wrapper)
		err = yaml.Unmarshal(configData, &templateConfig)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to parse configuration file %s: %w", configFile, err))
			return err
		}
	}

	// Update deployment cell configuration
	var pendingChanges []fleet.Amenity

	for _, a := range templateConfig.ManagedAmenities {
		pendingChanges = append(pendingChanges, fleet.Amenity{
			Name:        utils.ToPtr(a.Name),
			Description: a.Description,
			Type:        a.Type,
			Properties:  a.Properties,
			IsManaged:   utils.ToPtr(true),
		})
	}

	for _, a := range templateConfig.CustomAmenities {
		pendingChanges = append(pendingChanges, fleet.Amenity{
			Name:        utils.ToPtr(a.Name),
			Description: a.Description,
			Type:        a.Type,
			Properties:  a.Properties,
			IsManaged:   utils.ToPtr(false),
		})
	}

	// Show what we're doing
	if len(pendingChanges) == 0 {
		fmt.Printf("Removing all amenities from deployment cell\n")
	} else {
		fmt.Printf("Updating with %d amenities (%d managed, %d custom)\n",
			len(pendingChanges), len(templateConfig.ManagedAmenities), len(templateConfig.CustomAmenities))
	}

	err = dataaccess.UpdateHostCluster(ctx, token, deploymentCellID, pendingChanges, nil)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to update deployment cell configuration: %w", err))
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Successfully updated configuration for deployment cell %s", deploymentCellID))

	// Get updated deployment cell details
	var hc *fleet.HostCluster
	hc, err = dataaccess.DescribeHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get updated deployment cell details: %w", err))
		return err
	}

	// Print updated details in a readable format
	fmt.Printf("\nUpdated Deployment Cell Details:\n")
	fmt.Printf("ID: %s\n", hc.GetId())

	return nil
}
