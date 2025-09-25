package customnetwork

import (
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:          "custom-network [operation] [flags]",
	Short:        "List and describe custom networks of your customers",
	Long:         `This command helps you explore custom networks used by your customers.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}

func formatCustomNetwork(network *openapiclientfleet.FleetCustomNetwork) model.CustomNetwork {
	networkModel := model.CustomNetwork{
		CustomNetworkID:   network.Id,
		CustomNetworkName: utils.FromPtr(network.Name),
		CloudProvider:     network.CloudProviderName,
		Region:            network.CloudProviderRegion,
		CIDR:              network.Cidr,
		OwningOrgID:       network.OwningOrgID,
		OwningOrgName:     network.OwningOrgName,
		Status:            utils.FromPtr(network.Status),
	}

	if len(network.NetworkInstances) > 0 {
		networkInstance := network.NetworkInstances[0]
		networkModel.AwsAccountID = utils.FromPtr(networkInstance.AwsAccountID)
		networkModel.CloudProviderNativeNetworkId = utils.FromPtr(networkInstance.CloudProviderNativeNetworkId)
		networkModel.GcpProjectID = utils.FromPtr(networkInstance.GcpProjectID)
		networkModel.GcpProjectNumber = utils.FromPtr(networkInstance.GcpProjectNumber)
		networkModel.HostClusterID = utils.FromPtr(networkInstance.HostClusterID)
	}

	return networkModel
}
