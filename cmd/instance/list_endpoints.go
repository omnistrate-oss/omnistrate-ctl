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
	ClusterEndpoint     string                                        `json:"cluster_endpoint"`
	AdditionalEndpoints map[string]openapiclientfleet.ClusterEndpoint `json:"additional_endpoints"`
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

	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology == nil ||
		len(*instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology) == 0 {
		return nil
	}

	for _, resource := range *instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology {
		if resource.ClusterEndpoint == "" {
			if resource.AdditionalEndpoints == nil || len(*resource.AdditionalEndpoints) == 0 {
				// If both clusterEndpoint and additionalEndpoints are empty, skip this resource
				continue
			}
		}

		// Add to the map with resourceName as key
		var additionalEndpoints map[string]openapiclientfleet.ClusterEndpoint
		if resource.AdditionalEndpoints != nil {
			additionalEndpoints = *resource.AdditionalEndpoints
		} else {
			additionalEndpoints = make(map[string]openapiclientfleet.ClusterEndpoint)
		}
		resourceEndpoints[resource.ResourceName] = ResourceEndpoints{
			ClusterEndpoint:     resource.ClusterEndpoint,
			AdditionalEndpoints: additionalEndpoints,
		}
	}

	return resourceEndpoints
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
		for endpointName, endpoint := range endpoints.AdditionalEndpoints {
			url := endpoint.Endpoint
			status := endpoint.HealthStatus
			networkType := endpoint.NetworkingType
			ports := ""

			// Extract ports
			var portStrs []string
			for _, port := range endpoint.OpenPorts {
				portStrs = append(portStrs, fmt.Sprintf("%d", port))
			}
			ports = strings.Join(portStrs, ",")

			rows = append(rows, EndpointTableRow{
				ResourceName: resourceName,
				EndpointType: "additional",
				EndpointName: endpointName,
				URL:          utils.FromPtr(url),
				Status:       utils.FromPtr(status),
				NetworkType:  utils.FromPtr(networkType),
				Ports:        ports,
			})
		}
	}

	return rows
}
