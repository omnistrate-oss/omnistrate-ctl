package cloudnativenetwork

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

func newHostClusterCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "host-cluster [operation] [flags]",
		Short:        "Manage imported host clusters for cloud-native networks",
		Long:         `This command helps you manage provider host clusters imported from cloud-native networks.`,
		Run:          run,
		SilenceUsage: true,
	}
	cmd.AddCommand(newHostClusterImportCmd(commandPath + " host-cluster"))
	return cmd
}

func newHostClusterImportCmd(commandPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [account-id] --region=[region] --network-id=[network-id] --host-cluster-name=[name]",
		Short: "Import a host cluster from a cloud-native network",
		Long:  `Imports a provider host cluster from an imported cloud-native network into Omnistrate.`,
		Example: fmt.Sprintf(`# Import a host cluster from a cloud-native network
omnistrate-ctl %s import [account-id] --region=[region] --network-id=[network-id] --host-cluster-name=[name]`, commandPath),
		Args:         cobra.ExactArgs(1),
		RunE:         runHostClusterImport,
		SilenceUsage: true,
	}
	cmd.Flags().String("region", "", "The cloud provider region of the cloud-native network (required)")
	cmd.Flags().String("network-id", "", "The cloud-native network ID that contains the host cluster (required)")
	cmd.Flags().String("host-cluster-name", "", "The provider host cluster name to import (required)")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("network-id")
	_ = cmd.MarkFlagRequired("host-cluster-name")
	return cmd
}

func runHostClusterImport(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accountID := args[0]
	region, _ := cmd.Flags().GetString("region")
	networkID, _ := cmd.Flags().GetString("network-id")
	hostClusterName, _ := cmd.Flags().GetString("host-cluster-name")
	output, _ := cmd.Flags().GetString("output")

	if err := validateHostClusterImportFlags(region, networkID, hostClusterName); err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Importing host cluster...")
		sm.Start()
	}

	result, err := dataaccess.ImportAccountConfigCloudNativeNetworkHostCluster(
		cmd.Context(), token, accountID, strings.TrimSpace(region), strings.TrimSpace(networkID), strings.TrimSpace(hostClusterName),
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Host cluster imported successfully")
	return printHostClusterImportOutput(output, result)
}

func validateHostClusterImportFlags(region, networkID, hostClusterName string) error {
	if strings.TrimSpace(region) == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if strings.TrimSpace(networkID) == "" {
		return fmt.Errorf("network-id cannot be empty")
	}
	if strings.TrimSpace(hostClusterName) == "" {
		return fmt.Errorf("host-cluster-name cannot be empty")
	}
	return nil
}

func printHostClusterImportOutput(output string, result *dataaccess.CloudNativeNetworkHostClusterImportResult) error {
	switch output {
	case "json":
		return utils.PrintTextTableJsonOutput(output, result)
	case "table", "":
		return printHostClusterImportTable(result)
	default:
		return utils.PrintTextTableJsonOutput(output, result)
	}
}

func printHostClusterImportTable(result *dataaccess.CloudNativeNetworkHostClusterImportResult) error {
	if result == nil {
		fmt.Println("No host cluster import result found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HOST CLUSTER ID\tCREATED")
	fmt.Fprintln(w, "---------------\t-------")
	fmt.Fprintf(w, "%s\t%t\n", result.HostClusterID, result.Created)
	return w.Flush()
}
