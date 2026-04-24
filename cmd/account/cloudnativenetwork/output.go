package cloudnativenetwork

import (
	"fmt"
	"os"
	"text/tabwriter"

	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

func printCloudNativeNetworkOutput(output string, result *openapiclientv1.ListAccountConfigCloudNativeNetworksResult) error {
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

func printCloudNativeNetworkTable(result *openapiclientv1.ListAccountConfigCloudNativeNetworksResult) error {
	if result == nil || len(result.CloudNativeNetworks) == 0 {
		fmt.Println("No cloud-native networks found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NETWORK ID\tREGION\tNAME\tCIDR\tSTATUS\tPRIVATE SUBNETS\tPUBLIC SUBNETS")
	fmt.Fprintln(w, "------\t------\t----\t----\t------\t---------------\t--------------")

	for _, network := range result.CloudNativeNetworks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			network.CloudNativeNetworkId,
			network.Region,
			derefString(network.Name),
			derefString(network.Cidr),
			network.Status,
			len(network.PrivateSubnets),
			len(network.PublicSubnets),
		)
	}

	return w.Flush()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
