package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	listEndpointsExample = `# List endpoints for a specific instance
omctl instance list-endpoints instance-abcd1234`
)

// ResourceEndpoints represents the endpoints for a resource
type ResourceEndpoints struct {
	ClusterEndpoint     string         `json:"cluster_endpoint"`
	AdditionalEndpoints map[string]any `json:"additional_endpoints"`
}

// EndpointTableRow represents a single row in the table output
type EndpointTableRow struct {
	ResourceName string `json:"resource_name"`
	EndpointType string `json:"endpoint_type"`
	EndpointName string `json:"endpoint_name"`
	URL          string `json:"url"`
	Status       string `json:"status,omitempty"`
	NetworkType  string `json:"network_type,omitempty"`
	Ports        string `json:"ports,omitempty"`
}

var listEndpointsCmd = &cobra.Command{
	Use:          "list-endpoints [instance-id]",
	Short:        "List endpoints for a specific instance",
	Long:         `This command lists all additional endpoints and cluster endpoint for a specific instance by instance ID.`,
	Example:      listEndpointsExample,
	RunE:         runListEndpoints,
	SilenceUsage: true,
}

func init() {
	listEndpointsCmd.Args = cobra.ExactArgs(1) // Require exactly one argument (instance ID)
}

func runListEndpoints(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != common.OutputTypeJson {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Fetching endpoint information...")
		sm.Start()
	}

	// Check if instance exists and get details
	serviceID, environmentID, _, err := getInstanceWithResourceName(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get detailed instance information
	detailedInstance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Extract endpoint information
	resourceEndpoints := extractEndpoints(detailedInstance)

	if len(resourceEndpoints) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No endpoint information found for this instance.")
		// Print empty result for consistency
		if output == common.OutputTypeJson {
			err = utils.PrintTextTableJsonOutput(output, resourceEndpoints)
		} else {
			err = utils.PrintTextTableJsonArrayOutput(output, []EndpointTableRow{})
		}
		if err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved endpoint information")

	// Print output
	if output == common.OutputTypeJson {
		err = utils.PrintTextTableJsonOutput(output, resourceEndpoints)
	} else {
		// Convert to table format for better readability
		tableRows := convertToTableRows(resourceEndpoints)
		err = utils.PrintTextTableJsonArrayOutput(output, tableRows)
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// getInstanceWithResourceName gets instance details including resource name
func getInstanceWithResourceName(ctx context.Context, token, instanceID string) (serviceID, environmentID, productTierID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return
	}

	var found bool
	for _, instance := range searchRes.ResourceInstanceResults {
		if instance.Id == instanceID {
			serviceID = instance.ServiceId
			environmentID = instance.ServiceEnvironmentId
			productTierID = instance.ProductTierId
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("%s not found. Please check the instance ID and try again", instanceID)
		return
	}

	return
}

// extractEndpoints extracts endpoint information from the instance
func extractEndpoints(instance *openapiclientfleet.ResourceInstance) (resourceEndpoints map[string]ResourceEndpoints) {
	resourceEndpoints = make(map[string]ResourceEndpoints)

	if len(instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology) == 0 {
		return nil
	}

	for _, resourceIntfc := range instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology {
		resource, exists := resourceIntfc.(map[string]interface{})
		if !exists {
			continue // Skip if not a map
		}

		var data interface{}
		if data, exists = resource["resourceName"]; !exists {
			// If resourceName doesn't exist, skip this resource
			continue
		}

		var resourceName string
		if resourceName, exists = data.(string); !exists {
			// If resourceName is not a string, skip this resource
			continue
		}

		// Check for cluster endpoint
		var endpoints ResourceEndpoints
		if data, exists = resource["clusterEndpoint"]; exists {
			if endpoints.ClusterEndpoint, exists = data.(string); !exists {
				endpoints.ClusterEndpoint = "" // Default to empty string if not found
			}
		}

		// Check for additional endpoints
		if data, exists = resource["additionalEndpoints"]; exists {
			if endpoints.AdditionalEndpoints, exists = data.(map[string]interface{}); !exists {
				endpoints.AdditionalEndpoints = nil // Default to nil if not found
			}
		}

		if endpoints.ClusterEndpoint == "" && len(endpoints.AdditionalEndpoints) == 0 {
			// If both clusterEndpoint and additionalEndpoints are empty, skip this resource
			continue
		}

		// Add to the map with resourceName as key
		resourceEndpoints[resourceName] = endpoints
	}

	return
}

// convertToTableRows converts the nested endpoint structure to a flat table format
func convertToTableRows(resourceEndpoints map[string]ResourceEndpoints) []EndpointTableRow {
	var rows []EndpointTableRow

	for resourceName, endpoints := range resourceEndpoints {
		// Add cluster endpoint if present
		if endpoints.ClusterEndpoint != "" {
			rows = append(rows, EndpointTableRow{
				ResourceName: resourceName,
				EndpointType: "cluster",
				EndpointName: "cluster_endpoint",
				URL:          endpoints.ClusterEndpoint,
			})
		}

		// Add additional endpoints if present
		for endpointName, endpointData := range endpoints.AdditionalEndpoints {
			// Handle different endpoint data structures
			if endpointMap, ok := endpointData.(map[string]interface{}); ok {
				url := ""
				status := ""
				networkType := ""
				ports := ""

				// Extract URL/endpoint
				if endpointVal, exists := endpointMap["endpoint"]; exists {
					if endpointStr, ok := endpointVal.(string); ok {
						url = endpointStr
					}
				}

				// Extract health status
				if statusVal, exists := endpointMap["healthStatus"]; exists {
					if statusStr, ok := statusVal.(string); ok {
						status = statusStr
					}
				}

				// Extract network type
				if networkVal, exists := endpointMap["networkingType"]; exists {
					if networkStr, ok := networkVal.(string); ok {
						networkType = networkStr
					}
				}

				// Extract ports
				if portsVal, exists := endpointMap["openPorts"]; exists {
					if portsSlice, ok := portsVal.([]interface{}); ok {
						var portStrs []string
						for _, port := range portsSlice {
							if portNum, ok := port.(float64); ok {
								portStrs = append(portStrs, fmt.Sprintf("%.0f", portNum))
							} else if portStr, ok := port.(string); ok {
								portStrs = append(portStrs, portStr)
							}
						}
						ports = strings.Join(portStrs, ",")
					}
				}

				rows = append(rows, EndpointTableRow{
					ResourceName: resourceName,
					EndpointType: "additional",
					EndpointName: endpointName,
					URL:          url,
					Status:       status,
					NetworkType:  networkType,
					Ports:        ports,
				})
			} else {
				// Handle simple string endpoints
				if endpointStr, ok := endpointData.(string); ok {
					rows = append(rows, EndpointTableRow{
						ResourceName: resourceName,
						EndpointType: "additional",
						EndpointName: endpointName,
						URL:          endpointStr,
					})
				}
			}
		}
	}

	return rows
}
