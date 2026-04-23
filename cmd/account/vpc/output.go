package vpc

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

func printVPCOutput(output string, result *dataaccess.ListAccountConfigVPCsResult) error {
	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if len(result.CloudNativeNetworks) == 0 {
		fmt.Println("No VPCs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VPC ID\tREGION\tNAME\tCIDR\tSTATUS\tPRIVATE SUBNETS\tPUBLIC SUBNETS")
	fmt.Fprintln(w, "------\t------\t----\t----\t------\t---------------\t--------------")

	for _, vpc := range result.CloudNativeNetworks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			vpc.CloudNativeNetworkID,
			vpc.Region,
			vpc.Name,
			vpc.CIDR,
			vpc.Status,
			len(vpc.PrivateSubnets),
			len(vpc.PublicSubnets),
		)
	}

	return w.Flush()
}
