package cloudnativenetwork

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	importDeploymentCellExample = `# Import a deployment cell from an imported cloud-native network
omnistrate-ctl %s deployment-cell import ac-x9KpL2mQ7r --region=ap-south-1 --network-id=vpc-0f8a7c6d5e4b3a291 --name=imported-cell-r7x4q2`
)

func newDeploymentCellCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "deployment-cell [operation] [flags]",
		Short:        "Manage deployment cells backed by imported cloud-native networks",
		Long:         `This command helps you manage deployment cells backed by imported cloud-native networks.`,
		Run:          run,
		SilenceUsage: true,
	}

	cmd.AddCommand(newDeploymentCellImportCmd(commandPath))

	return cmd
}

func newDeploymentCellImportCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "import [account-id] --region=[region] --network-id=[network-id] --name=[name]",
		Short:        "Import a deployment cell from an imported cloud-native network",
		Long:         `Creates or returns an Omnistrate deployment cell record backed by an imported cloud-native network.`,
		Example:      fmt.Sprintf(importDeploymentCellExample, commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runDeploymentCellImport,
		SilenceUsage: true,
	}

	cmd.Flags().String("region", "", "The region of the imported cloud-native network (required)")
	cmd.Flags().String("network-id", "", "The cloud-native network ID to import the deployment cell from (required)")
	cmd.Flags().String("name", "", "The cloud provider deployment cell name to import (required)")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("network-id")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runDeploymentCellImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	region, _ := cmd.Flags().GetString("region")
	networkID, _ := cmd.Flags().GetString("network-id")
	name, _ := cmd.Flags().GetString("name")
	output, _ := cmd.Flags().GetString("output")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Importing deployment cell...")
		sm.Start()
	}

	result, err := dataaccess.ImportAccountConfigCloudNativeNetworkDeploymentCell(cmd.Context(), token, accountID, region, networkID, name)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Deployment cell imported successfully")

	return printDeploymentCellImportOutput(output, result)
}
