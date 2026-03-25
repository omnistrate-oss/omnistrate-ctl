package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var debugHelmLogsCmd = &cobra.Command{
	Use:   "helm-logs [instance-id]",
	Short: "Get Helm installation logs for instance resources",
	Long:  `Get Helm installation logs for instance resources. Use --resource-id or --resource-key to filter by specific resource.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDebugHelmLogs,
	Example: `  omnistrate-ctl instance debug helm-logs <instance-id>
  omnistrate-ctl instance debug helm-logs <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug helm-logs <instance-id> --resource-id abc123`,
}

type HelmLogsOutput struct {
	InstanceID string             `json:"instanceId"`
	Resources  []HelmLogsResource `json:"resources"`
}

type HelmLogsResource struct {
	ResourceID  string `json:"resourceId"`
	ResourceKey string `json:"resourceKey"`
	InstallLog  string `json:"installLog"`
	Namespace   string `json:"namespace,omitempty"`
	ReleaseName string `json:"releaseName,omitempty"`
}

func runDebugHelmLogs(cmd *cobra.Command, args []string) error {
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

	output := HelmLogsOutput{
		InstanceID: instanceID,
		Resources:  []HelmLogsResource{},
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

			// Check if it's a helm resource
			if _, hasChart := actualDebugData["chartRepoName"]; !hasChart {
				continue
			}

			// Parse helm data
			helmData := parseHelmData(actualDebugData)

			helmLogsResource := HelmLogsResource{
				ResourceID:  actualResourceID,
				ResourceKey: resourceKey,
				InstallLog:  helmData.InstallLog,
				Namespace:   helmData.Namespace,
				ReleaseName: helmData.ReleaseName,
			}

			output.Resources = append(output.Resources, helmLogsResource)
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
	debugHelmLogsCmd.Flags().String("resource-id", "", "Filter by resource ID")
	debugHelmLogsCmd.Flags().String("resource-key", "", "Filter by resource key")
}
