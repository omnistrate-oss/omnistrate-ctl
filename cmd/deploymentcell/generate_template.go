package deploymentcell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var generateTemplateCmd = &cobra.Command{
	Use:   "generate-config-template",
	Short: "Generate deployment cell configuration template",
	Long: `Generate a deployment cell configuration template with available amenities for a specific cloud provider.

This command creates a YAML template file containing all available amenities (Helm charts) 
that can be configured for deployment cells. The template includes both managed amenities 
(maintained by Omnistrate) and custom amenities based on the organization's current configuration.

The generated template can be customized and used with the update-config-template command 
to configure deployment cell amenities for your organization.

Examples:
  # Generate template for AWS cloud provider
  omnistrate-ctl deployment-cell generate-config-template --cloud aws --output template-aws.yaml

  # Generate template for Azure cloud provider
  omnistrate-ctl deployment-cell generate-config-template --cloud azure --output template-azure.yaml

  # Generate template and display to stdout
  omnistrate-ctl deployment-cell generate-config-template --cloud aws`,
	RunE:         runGenerateTemplate,
	SilenceUsage: true,
}

func init() {
	generateTemplateCmd.Flags().StringP("cloud", "c", "", "Cloud provider to generate template for (aws,azure,gcp).")
	generateTemplateCmd.Flags().StringP("output", "o", "", "Output file path for the template (if not specified, outputs to stdout)")
	_ = generateTemplateCmd.MarkFlagRequired("cloud")
}

func runGenerateTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	cloudProvider, err := cmd.Flags().GetString("cloud")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	outputFile, err := cmd.Flags().GetString("output")
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

	// Generate template for specified cloud providers
	template, err := generateDeploymentCellTemplate(ctx, token, cloudProvider)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to generate template: %w", err))
		return err
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(template)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to marshal template to YAML: %w", err))
		return err
	}

	// Output to file or stdout
	if outputFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(outputFile)
		if dir != "." {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				utils.PrintError(fmt.Errorf("failed to create directory %s: %w", dir, err))
				return err
			}
		}

		err = os.WriteFile(outputFile, yamlData, 0600)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to write template to file %s: %w", outputFile, err))
			return err
		}

		utils.PrintSuccess(fmt.Sprintf("Template generated successfully and saved to %s", outputFile))
	} else {
		fmt.Print(string(yamlData))
	}

	return nil
}

func generateDeploymentCellTemplate(ctx context.Context, token string, cloudProviderName string) (map[string]model.DeploymentCellTemplate, error) {
	res, err := dataaccess.GetServiceProviderOrganization(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get service provider organization: %w", err)
	}

	// Convert the API response to model.DeploymentCellConfigurationTemplate
	template := make(map[string]model.DeploymentCellTemplate)

	if res.DefaultDeploymentCellConfigurations == nil {
		return template, fmt.Errorf("no default deployment cell configurations found in service provider organization")
	}

	for cloudProvider, cellConfig := range res.DefaultDeploymentCellConfigurations.DeploymentCellConfigurationPerCloudProvider {
		if cloudProvider != cloudProviderName {
			continue // Skip if the cloud provider does not match
		}
		var convertedConfig model.DeploymentCellTemplate
		convertedConfig, err = convertToDeploymentCellConfiguration(cellConfig)
		if err != nil {
			return template, fmt.Errorf("failed to convert deployment cell configuration for cloud provider '%s': %w", cloudProvider, err)
		}
		template[cloudProvider] = convertedConfig
	}
	return template, nil
}

func convertToDeploymentCellConfiguration(cellConfig interface{}) (model.DeploymentCellTemplate, error) {
	result := model.DeploymentCellTemplate{}

	// Handle the case where apiResponse is nil
	if cellConfig == nil {
		return result, nil
	}

	internalAmenities, err := dataaccess.ConvertToInternalAmenitiesList(cellConfig)
	if err != nil {
		return result, fmt.Errorf("failed to convert amenities list: %w", err)
	}

	var managedAmenities []model.Amenity
	var customAmenities []model.Amenity
	for _, amenity := range internalAmenities {
		externalModel := model.Amenity{
			Name:        amenity.Name,
			Description: amenity.Description,
			Type:        amenity.Type,
			Properties:  amenity.Properties,
		}
		if utils.FromPtr(amenity.IsManaged) {
			managedAmenities = append(managedAmenities, externalModel)
		} else {
			customAmenities = append(customAmenities, externalModel)
		}
	}

	result.ManagedAmenities = managedAmenities
	result.CustomAmenities = customAmenities
	return result, nil
}
