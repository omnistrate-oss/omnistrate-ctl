package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	listEndpointsExample = `# List endpoints for a specific instance
omnistrate-ctl instance list-endpoints instance-abcd1234`
)

// ResourceEndpoints represents the endpoints for a resource
type ResourceEndpoints struct {
	ClusterEndpoint     string                                        `json:"cluster_endpoint"`
	ClusterPorts        []int64                                       `json:"cluster_ports,omitempty"`
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

var (
	endpointsFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#334155")).
				Foreground(lipgloss.Color("#D1D5DB")).
				Padding(1, 2).
				Width(120)
	endpointsTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F8FAFC"))
	endpointsResourceStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50C878"))
	endpointsLabelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	endpointsValueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	endpointsMutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
)

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
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Fetching endpoint information...")
		sm.Start()
	}

	// Check if instance exists and get details
	serviceID, environmentID, err := getInstanceWithResourceName(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Extract endpoint information
	resourceEndpoints, err := FetchEndpointsForInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

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
func getInstanceWithResourceName(ctx context.Context, token, instanceID string) (serviceID, environmentID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return
	}

	var found bool
	for _, instance := range searchRes.ResourceInstanceResults {
		if instance.Id == instanceID {
			serviceID = instance.ServiceId
			environmentID = instance.ServiceEnvironmentId
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

// FetchEndpointsForInstance fetches endpoint information for a known instance.
func FetchEndpointsForInstance(ctx context.Context, token, serviceID, environmentID, instanceID string) (map[string]ResourceEndpoints, error) {
	detailedInstance, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return nil, err
	}
	return extractEndpoints(detailedInstance), nil
}

// PrintEndpointsForInstance prints endpoint information grouped by resource.
func PrintEndpointsForInstance(ctx context.Context, token, serviceID, environmentID, instanceID string) error {
	resourceEndpoints, err := FetchEndpointsForInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return err
	}
	PrintResourceEndpoints(resourceEndpoints)
	return nil
}

// PrintResourceEndpoints renders endpoint information grouped by resource.
func PrintResourceEndpoints(resourceEndpoints map[string]ResourceEndpoints) {
	fmt.Println(renderResourceEndpoints(resourceEndpoints))
}

func renderResourceEndpoints(resourceEndpoints map[string]ResourceEndpoints) string {
	var body strings.Builder
	body.WriteString(endpointsTitleStyle.Render("Deployment endpoints"))

	if len(resourceEndpoints) == 0 {
		body.WriteString("\n\n")
		body.WriteString(endpointsMutedStyle.Render("No endpoints are published for this instance."))
		return endpointsFrameStyle.Render(body.String())
	}

	resourceNames := make([]string, 0, len(resourceEndpoints))
	for resourceName := range resourceEndpoints {
		resourceNames = append(resourceNames, resourceName)
	}
	sort.Strings(resourceNames)

	for _, resourceName := range resourceNames {
		endpoints := resourceEndpoints[resourceName]
		body.WriteString("\n\n")
		body.WriteString(endpointsResourceStyle.Render(resourceName + " ->"))

		if endpoints.ClusterEndpoint != "" {
			for _, formattedURL := range formatEndpointURLs(endpoints.ClusterEndpoint, endpoints.ClusterPorts) {
				body.WriteString("\n")
				body.WriteString(renderEndpointLine(resourceName+" endpoint", formattedURL))
			}
		}

		endpointNames := make([]string, 0, len(endpoints.AdditionalEndpoints))
		for endpointName := range endpoints.AdditionalEndpoints {
			endpointNames = append(endpointNames, endpointName)
		}
		sort.Strings(endpointNames)

		for _, endpointName := range endpointNames {
			endpoint := endpoints.AdditionalEndpoints[endpointName]
			url := utils.FromPtr(endpoint.Endpoint)
			if url == "" {
				continue
			}
			for _, formattedURL := range formatEndpointURLs(url, endpoint.OpenPorts) {
				body.WriteString("\n")
				body.WriteString(renderEndpointLine(endpointName, formattedURL))
			}
		}
	}

	return endpointsFrameStyle.Render(strings.TrimRight(body.String(), "\n"))
}

func renderEndpointLine(name, url string) string {
	return fmt.Sprintf("  %s  %s", endpointsLabelStyle.Width(22).Render(name), endpointsValueStyle.Render(url))
}

func formatEndpointURLs(endpoint string, ports []int64) []string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	if len(ports) == 0 {
		return []string{formatEndpointURL(endpoint, 0)}
	}

	sortedPorts := append([]int64(nil), ports...)
	sort.Slice(sortedPorts, func(i, j int) bool {
		return sortedPorts[i] < sortedPorts[j]
	})

	urls := make([]string, 0, len(sortedPorts))
	for _, port := range sortedPorts {
		urls = append(urls, formatEndpointURL(endpoint, port))
	}
	return urls
}

func formatEndpointURL(endpoint string, port int64) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}

	scheme := ""
	switch port {
	case 80:
		scheme = "http://"
	case 443:
		scheme = "https://"
	}

	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if port > 0 {
		endpoint = fmt.Sprintf("%s:%d", endpoint, port)
	}
	return scheme + endpoint
}

// extractEndpoints extracts endpoint information from the instance
func extractEndpoints(instance *openapiclientfleet.ResourceInstance) (resourceEndpoints map[string]ResourceEndpoints) {
	resourceEndpoints = make(map[string]ResourceEndpoints)

	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology == nil ||
		len(*instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology) == 0 {
		return nil
	}

	for _, resource := range *instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology {
		if shouldHideObservabilityEndpoints(resource) {
			continue
		}

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
			ClusterPorts:        append([]int64(nil), resource.ClusterPorts...),
			AdditionalEndpoints: additionalEndpoints,
		}
	}

	return resourceEndpoints
}

func shouldHideObservabilityEndpoints(resource openapiclientfleet.ResourceNetworkTopologyResult) bool {
	if resource.ResourceKey == "omnistrateobserv" {
		return true
	}

	return strings.EqualFold(strings.TrimSpace(resource.ResourceName), "Omnistrate Observability")
}

// convertToTableRows converts the nested endpoint structure to a flat table format
func convertToTableRows(resourceEndpoints map[string]ResourceEndpoints) []EndpointTableRow {
	var rows []EndpointTableRow

	for resourceName, endpoints := range resourceEndpoints {
		// Add cluster endpoint if present
		if endpoints.ClusterEndpoint != "" {
			for _, url := range formatEndpointURLs(endpoints.ClusterEndpoint, endpoints.ClusterPorts) {
				rows = append(rows, EndpointTableRow{
					ResourceName: resourceName,
					EndpointType: "cluster",
					EndpointName: "cluster_endpoint",
					URL:          url,
				})
			}
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
