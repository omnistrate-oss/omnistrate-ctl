package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var debugTerraformFilesCmd = &cobra.Command{
	Use:   "terraform-files [instance-id]",
	Short: "Get Terraform files for instance resources",
	Long:  `Get Terraform files for instance resources. Use --resource-id or --resource-key to filter by specific resource.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDebugTerraformFiles,
	Example: `  omnistrate-ctl instance debug terraform-files <instance-id>
  omnistrate-ctl instance debug terraform-files <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug terraform-files <instance-id> --resource-id abc123`,
}

type TerraformFilesOutput struct {
	InstanceID string                   `json:"instanceId"`
	Resources  []TerraformFilesResource `json:"resources"`
}

type TerraformFilesResource struct {
	ResourceID  string            `json:"resourceId"`
	ResourceKey string            `json:"resourceKey"`
	Files       map[string]string `json:"files"`
}

func runDebugTerraformFiles(cmd *cobra.Command, args []string) error {
	instanceID := args[0]

	resourceIDFilter, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		return fmt.Errorf("failed to get resource-id flag: %w", err)
	}

	resourceKeyFilter, err := cmd.Flags().GetString("resource-key")
	if err != nil {
		return fmt.Errorf("failed to get resource-key flag: %w", err)
	}

	resourceNameFilter, err := cmd.Flags().GetString("resource-name")
	if err != nil {
		return fmt.Errorf("failed to get resource-name flag: %w", err)
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()

	// Get instance details
	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	instanceData, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID, true)
	if err != nil {
		return fmt.Errorf("failed to describe resource instance: %w", err)
	}

	resourceIndex, err := buildResourceIndex(ctx, token, serviceID, instanceData, resourceNameFilter != "")
	if err != nil {
		return fmt.Errorf("failed to build resource indexes: %w", err)
	}

	filter, err := resolveResourceFilter(rawResourceFilter{
		key:  resourceKeyFilter,
		name: resourceNameFilter,
		id:   resourceIDFilter,
	}, resourceIndex)
	if err != nil {
		return err
	}

	var terraformConfigMapIndex *terraformConfigMapIndex
	if resourceIndex.needsTerraformData(filter) {
		terraformConfigMapIndex, _, err = loadTerraformConfigMapIndexForInstance(ctx, token, instanceData, instanceID)
		if err != nil {
			return fmt.Errorf("failed to load terraform configmaps: %w", err)
		}
	}

	output := TerraformFilesOutput{
		InstanceID: instanceID,
		Resources:  []TerraformFilesResource{},
	}

	for _, resource := range listTerraformResources(instanceData, resourceIndex, filter) {
		terraformData := &TerraformData{
			Files: make(map[string]string),
			Logs:  make(map[string]string),
		}
		if terraformConfigMapIndex != nil && resource.id != "" {
			terraformData = terraformConfigMapIndex.terraformDataForResource(resource.id)
		}

		terraformFilesResource := TerraformFilesResource{
			ResourceID:  resource.id,
			ResourceKey: resource.key,
			Files:       terraformData.Files,
		}

		output.Resources = append(output.Resources, terraformFilesResource)
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func init() {
	debugTerraformFilesCmd.Flags().String("resource-id", "", "Filter by resource ID")
	debugTerraformFilesCmd.Flags().String("resource-key", "", "Filter by resource key")
	debugTerraformFilesCmd.Flags().String("resource-name", "", "Filter by resource name")
}
