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

	output := TerraformFilesOutput{
		InstanceID: instanceID,
		Resources:  []TerraformFilesResource{},
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

			terraformFilesResource := TerraformFilesResource{
				ResourceID:  actualResourceID,
				ResourceKey: resourceKey,
				Files:       terraformData.Files,
			}

			output.Resources = append(output.Resources, terraformFilesResource)
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
	debugTerraformFilesCmd.Flags().String("resource-id", "", "Filter by resource ID")
	debugTerraformFilesCmd.Flags().String("resource-key", "", "Filter by resource key")
}
