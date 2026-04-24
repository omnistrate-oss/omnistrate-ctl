package vpc

import (
	"fmt"
	"os"
	"text/tabwriter"

	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

func printVPCOutput(output string, result *openapiclientv1.ListAccountConfigCloudNativeNetworksResult) error {
	switch output {
	case "json":
		return utils.PrintTextTableJsonOutput(output, result)
	case "table", "":
		return printVPCTable(result)
	default:
		// Delegate "text" and any unknown values to the shared printer so the
		// repo's "text|table|json" output contract is honored consistently.
		return utils.PrintTextTableJsonOutput(output, result)
	}
}

func printVPCTable(result *openapiclientv1.ListAccountConfigCloudNativeNetworksResult) error {
	if result == nil || len(result.CloudNativeNetworks) == 0 {
		fmt.Println("No VPCs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VPC ID\tREGION\tNAME\tCIDR\tSTATUS\tPRIVATE SUBNETS\tPUBLIC SUBNETS")
	fmt.Fprintln(w, "------\t------\t----\t----\t------\t---------------\t--------------")

	for _, vpc := range result.CloudNativeNetworks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			vpc.CloudNativeNetworkId,
			vpc.Region,
			derefString(vpc.Name),
			derefString(vpc.Cidr),
			vpc.Status,
			len(vpc.PrivateSubnets),
			len(vpc.PublicSubnets),
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
