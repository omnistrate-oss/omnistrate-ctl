package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var debugTerraformOutputsCmd = &cobra.Command{
	Use:   "terraform-output [instance-id]",
	Short: "Get Terraform logs for instance resources",
	Long:  `Get Terraform logs (apply, plan, etc.) for instance resources. Use --resource-id or --resource-key to filter by specific resource.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDebugTerraformOutputs,
	Example: `  omnistrate-ctl instance debug terraform-output <instance-id>
  omnistrate-ctl instance debug terraform-output <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug terraform-output <instance-id> --resource-id abc123`,
}

type TerraformOutputsOutput struct {
	InstanceID string                     `json:"instanceId"`
	Resources  []TerraformOutputsResource `json:"resources"`
}

type TerraformOutputsResource struct {
	ResourceID  string            `json:"resourceId"`
	ResourceKey string            `json:"resourceKey"`
	Logs        map[string]string `json:"logs"`
}

func runDebugTerraformOutputs(cmd *cobra.Command, args []string) error {
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

	output := TerraformOutputsOutput{
		InstanceID: instanceID,
		Resources:  []TerraformOutputsResource{},
	}

	for _, resource := range listTerraformResources(instanceData, resourceIndex, filter) {
		terraformData := &TerraformData{
			Files: make(map[string]string),
			Logs:  make(map[string]string),
		}
		if terraformConfigMapIndex != nil && resource.id != "" {
			terraformData = terraformConfigMapIndex.terraformDataForResource(resource.id)
		}

		terraformOutputsResource := TerraformOutputsResource{
			ResourceID:  resource.id,
			ResourceKey: resource.key,
			Logs:        terraformData.Logs,
		}

		output.Resources = append(output.Resources, terraformOutputsResource)
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
	debugTerraformOutputsCmd.Flags().String("resource-id", "", "Filter by resource ID")
	debugTerraformOutputsCmd.Flags().String("resource-key", "", "Filter by resource key")
	debugTerraformOutputsCmd.Flags().String("resource-name", "", "Filter by resource name")
}
