package deploymentcell

import (
	"context"
	"fmt"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var describeTemplateCmd = &cobra.Command{
	Use:   "describe-config-template",
	Short: "Describe deployment cell configuration template",
	Long: `Describe the current deployment cell configuration template for your organization.

This command shows the current amenities configuration template that is applied to 
new deployment cells in the specified environment and cloud provider.

You can also describe the configuration of a specific deployment cell by providing 
its ID as an argument.

Examples:
  # Describe organization template for PROD environment and AWS
  omnistrate-ctl deployment-cell describe-config-template -e PROD --cloud aws

  # Describe specific deployment cell configuration
  omnistrate-ctl deployment-cell describe-config-template hc-12345

  # Get JSON output format
  omnistrate-ctl deployment-cell describe-config-template -e PROD --cloud aws --output json`,
	RunE:         runDescribeTemplate,
	SilenceUsage: true,
}

func init() {
	describeTemplateCmd.Flags().StringP("environment", "e", "", "Environment type (e.g., PROD, STAGING)")
	describeTemplateCmd.Flags().StringP("cloud", "c", "", "Cloud provider (aws, azure, gcp)")
	describeTemplateCmd.Flags().StringP("id", "i", "", "Deployment cell ID")
	describeTemplateCmd.Flags().StringP("output", "o", "yaml", "Output format (yaml, json, table)")
}

func runDescribeTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// ID
	id, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	cloudProvider, err := cmd.Flags().GetString("cloud")
	if err != nil {
		utils.PrintError(err)
		return err
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

	if id != "" {
		return describeDeploymentCellConfiguration(ctx, token, id, output)
	} else {
		// Validate required flags for organization template
		if environment == "" {
			err := fmt.Errorf("environment flag is required when not specifying a deployment cell ID")
			utils.PrintError(err)
			return err
		}

		if cloudProvider == "" {
			err := fmt.Errorf("cloud flag is required when not specifying a deployment cell ID")
			utils.PrintError(err)
			return err
		}

		return describeOrganizationTemplate(ctx, token, environment, cloudProvider, output)
	}
}

func describeOrganizationTemplate(ctx context.Context, token string, environment string, cloudProvider string, output string) error {
	// Get the service provider organization configuration
	spOrg, err := dataaccess.GetServiceProviderOrganization(ctx, token)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get service provider organization: %w", err))
		return err
	}

	// Extract deployment cell configurations for the specified environment
	template, err := dataaccess.GetOrganizationDeploymentCellTemplate(ctx, token, environment, cloudProvider)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get deployment cell template: %w", err))
		return err
	}

	if template == nil {
		utils.PrintInfo(fmt.Sprintf("No configuration template found for environment '%s' and cloud provider '%s'", environment, cloudProvider))
		return nil
	}

	totalAmenities := len(template.ManagedAmenities) + len(template.CustomAmenities)
	fmt.Printf("📋 Deployment Cell Configuration Template\n")
	fmt.Printf("Organization: %s\n", utils.FromPtr(spOrg.Id))
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Cloud Provider: %s\n", cloudProvider)
	fmt.Printf("Total Amenities: %d (Managed: %d, Custom: %d)\n\n", totalAmenities, len(template.ManagedAmenities), len(template.CustomAmenities))

	// Print output based on format
	switch output {
	case "table":
		return printTemplateAsTable(template)
	case "json":
		return utils.PrintTextTableJsonOutput(output, template)
	case "yaml":
		return utils.PrintTextTableYamlOutput(template)
	default:
		return utils.PrintTextTableYamlOutput(template)
	}
}

func describeDeploymentCellConfiguration(ctx context.Context, token string, deploymentCellID string, output string) error {
	// Get deployment cell details
	deploymentCell, err := dataaccess.DescribeHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get deployment cell details: %w", err))
		return err
	}

	// Create deployment cell template structure for consistent output
	deploymentCellTemplate := createDeploymentCellTemplate(deploymentCell)

	// Print output based on format - use the deployment cell template for consistent YAML output
	switch output {
	case "table":
		return printDeploymentCellTemplateAsTable(deploymentCellTemplate)
	case "json":
		return utils.PrintTextTableJsonOutput(output, deploymentCellTemplate)
	case "yaml":
		return utils.PrintTextTableYamlOutput(deploymentCellTemplate)
	default:
		return utils.PrintTextTableYamlOutput(deploymentCellTemplate)
	}
}

func printTemplateAsTable(template *model.DeploymentCellTemplate) error {
	type AmenityTableView struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		IsManaged   string `json:"is_managed"`
		Modifiable  string `json:"modifiable"`
		Description string `json:"description"`
	}

	var tableData []interface{}

	// Add managed amenities
	for _, amenity := range template.ManagedAmenities {
		view := AmenityTableView{
			Name:      amenity.Name,
			IsManaged: "Yes",
		}
		if amenity.Type != nil {
			view.Type = *amenity.Type
		}

		if amenity.Description != nil {
			view.Description = *amenity.Description
		}

		tableData = append(tableData, view)
	}

	// Add custom amenities
	for _, amenity := range template.CustomAmenities {
		view := AmenityTableView{
			Name:      amenity.Name,
			IsManaged: "No",
		}

		if amenity.Type != nil {
			view.Type = *amenity.Type
		}
		if amenity.Description != nil {
			view.Description = *amenity.Description
		}

		tableData = append(tableData, view)
	}

	return utils.PrintTextTableJsonArrayOutput("table", tableData)
}

func createDeploymentCellTemplate(deploymentCell *fleet.HostCluster) *model.DeploymentCellTemplate {
	// Convert fleet amenities to template amenities format and categorize by managed vs custom
	var managedAmenities []model.Amenity
	var customAmenities []model.Amenity

	for _, amenity := range deploymentCell.GetAmenities() {
		templateAmenity := model.Amenity{
			Name:        amenity.GetName(),
			Type:        amenity.Type,
			Description: amenity.Description,
		}

		// Add properties if available
		if amenity.Properties != nil {
			templateAmenity.Properties = amenity.Properties
		}

		// Categorize based on isManaged flag
		if amenity.IsManaged != nil && *amenity.IsManaged {
			managedAmenities = append(managedAmenities, templateAmenity)
		} else {
			customAmenities = append(customAmenities, templateAmenity)
		}
	}

	// Create a deployment cell template instance consistent with organization template format
	template := &model.DeploymentCellTemplate{
		ManagedAmenities: managedAmenities,
		CustomAmenities:  customAmenities,
	}

	return template
}

func printDeploymentCellTemplateAsTable(template *model.DeploymentCellTemplate) error {
	type AmenityTableView struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		IsManaged   string `json:"is_managed"`
		Modifiable  string `json:"modifiable"`
		Description string `json:"description"`
	}

	var tableData []interface{}

	// Add managed amenities
	for _, amenity := range template.ManagedAmenities {
		view := AmenityTableView{
			Name:      amenity.Name,
			IsManaged: "Yes",
		}

		if amenity.Type != nil {
			view.Type = *amenity.Type
		}
		if amenity.Description != nil {
			view.Description = *amenity.Description
		}

		tableData = append(tableData, view)
	}

	// Add custom amenities
	for _, amenity := range template.CustomAmenities {
		view := AmenityTableView{
			Name:      amenity.Name,
			IsManaged: "No",
		}

		if amenity.Type != nil {
			view.Type = *amenity.Type
		}
		if amenity.Description != nil {
			view.Description = *amenity.Description
		}

		tableData = append(tableData, view)
	}

	return utils.PrintTextTableJsonArrayOutput("table", tableData)
}
