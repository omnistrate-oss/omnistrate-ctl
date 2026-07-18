package cloudnativenetwork

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const cloudNativeNetworkTableColumnCount = 9

func printCloudNativeNetworkOutput(output string, result *openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult) error {
	switch output {
	case "json":
		return utils.PrintTextTableJsonOutput(output, result)
	case "table", "":
		return printCloudNativeNetworkTable(result)
	default:
		// Delegate "text" and any unknown values to the shared printer so the
		// repo's "text|table|json" output contract is honored consistently.
		return utils.PrintTextTableJsonOutput(output, result)
	}
}

func printCloudNativeNetworkTable(result *openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult) error {
	if result == nil || len(result.CloudNativeNetworks) == 0 {
		fmt.Println("No cloud-native networks found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	includeHostClusters := hasHostClusters(result)
	header := "NETWORK ID\tREGION\tNAME\tCIDR\tSTATUS\tIMPORTED\tIN USE\tPRIVATE SUBNETS\tPUBLIC SUBNETS"
	separator := "------\t------\t----\t----\t------\t--------\t------\t---------------\t--------------"
	if includeHostClusters {
		header += "\tHOST CLUSTERS"
		separator += "\t-------------"
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, separator)

	for _, network := range result.CloudNativeNetworks {
		row := []string{network.CloudNativeNetworkId, network.Region, derefString(network.Name), derefString(network.Cidr), network.Status,
			fmt.Sprintf("%t", derefBool(network.Imported)), fmt.Sprintf("%t", derefBool(network.InUse)), fmt.Sprintf("%d", len(network.PrivateSubnets)), fmt.Sprintf("%d", len(network.PublicSubnets))}
		if includeHostClusters {
			row = append(row, formatHostClustersForTable(network.HostClusters))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return w.Flush()
}

func hasHostClusters(result *openapiclientfleet.FleetListAccountConfigCloudNativeNetworksResult) bool {
	for _, network := range result.CloudNativeNetworks {
		if len(network.HostClusters) > 0 {
			return true
		}
	}
	return false
}

func formatHostClusters(hostClusters []openapiclientfleet.FleetAccountConfigCloudNativeNetworkHostClusterResult) string {
	lines := make([]string, 0, len(hostClusters))
	for _, hostCluster := range hostClusters {
		if hostCluster.EligibleToImport {
			lines = append(lines, fmt.Sprintf("%s - eligible for import", hostCluster.Name))
			continue
		}

		line := fmt.Sprintf("%s - not eligible for import", hostCluster.Name)
		if hostCluster.IneligibilityReason != nil {
			line += " - " + *hostCluster.IneligibilityReason
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func formatHostClustersForTable(hostClusters []openapiclientfleet.FleetAccountConfigCloudNativeNetworkHostClusterResult) string {
	return strings.ReplaceAll(formatHostClusters(hostClusters), "\n", "\n"+strings.Repeat("\t", cloudNativeNetworkTableColumnCount))
}

func printDeploymentCellImportOutput(output string, result *dataaccess.CloudNativeNetworkDeploymentCellImportResult) error {
	switch output {
	case "json":
		return utils.PrintTextTableJsonOutput(output, result)
	case "table", "":
		return printDeploymentCellImportTable(result)
	default:
		return utils.PrintTextTableJsonOutput(output, result)
	}
}

func printDeploymentCellImportTable(result *dataaccess.CloudNativeNetworkDeploymentCellImportResult) error {
	if result == nil {
		fmt.Println("No deployment cell import result found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HOST CLUSTER ID\tCREATED")
	fmt.Fprintln(w, "---------------\t-------")
	fmt.Fprintf(w, "%s\t%t\n", result.HostClusterID, result.Created)

	return w.Flush()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
