package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	resourceID, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		return fmt.Errorf("failed to get resource-id flag: %w", err)
	}

	resourceKeyFilter, err := cmd.Flags().GetString("resource-key")
	if err != nil {
		return fmt.Errorf("failed to get resource-key flag: %w", err)
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

	// Get debug information
	debugResult, err := dataaccess.DebugResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get debug information: %w", err)
	}

	output := TerraformOutputsOutput{
		InstanceID: instanceID,
		Resources:  []TerraformOutputsResource{},
	}

	if debugResult.ResourcesDebug != nil {
		for resourceKey, resourceDebugInfo := range *debugResult.ResourcesDebug {
			// Skip omnistrateobserv
			if resourceKey == "omnistrateobserv" {
				continue
			}

			// Apply resource-key filter if specified
			if resourceKeyFilter != "" && resourceKeyFilter != resourceKey {
				continue
			}

			// Get actual resource ID if filter is specified
			var actualResourceID string
			if resourceID != "" {
				actualResourceID, _, err = getResourceFromInstance(ctx, token, instanceID, resourceKey)
				if err == nil && actualResourceID != "" {
					if resourceID != actualResourceID {
						continue
					}
				}
			}

			// Get debug data from DebugResourceResult
			debugDataInterface, ok := resourceDebugInfo.GetDebugDataOk()
			if !ok || debugDataInterface == nil {
				continue
			}

			// Convert to map
			actualDebugData, ok := (*debugDataInterface).(map[string]interface{})
			if !ok {
				continue
			}

			// Check if it's a terraform resource (has rendered/*.tf files or log/terraform*)
			isTerraform := false
			for key := range actualDebugData {
				if (strings.HasPrefix(key, "rendered/") && strings.HasSuffix(key, ".tf")) ||
					(strings.HasPrefix(key, "log/") && strings.Contains(key, "terraform")) {
					isTerraform = true
					break
				}
			}
			if !isTerraform {
				continue
			}

			// Parse terraform data
			terraformData := parseTerraformData(actualDebugData)

			terraformOutputsResource := TerraformOutputsResource{
				ResourceID:  actualResourceID,
				ResourceKey: resourceKey,
				Logs:        terraformData.Logs,
			}

			output.Resources = append(output.Resources, terraformOutputsResource)
		}
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
}
