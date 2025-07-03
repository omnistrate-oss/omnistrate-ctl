package customnetwork

import (
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/ctl/cmd/common"
	"github.com/omnistrate-oss/ctl/internal/config"
	"github.com/omnistrate-oss/ctl/internal/dataaccess"
	"github.com/omnistrate-oss/ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe a custom network by ID
omctl custom-network describe --custom-network-id [custom-network-id]`
)

var describeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a custom network",
	Long:         `This command helps you describe an existing custom network.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Flags().StringP(CustomNetworkIDFlag, "", "", "ID of the custom network")

	err := describeCmd.MarkFlagRequired(CustomNetworkIDFlag)
	if err != nil {
		return
	}
}

func runDescribe(cmd *cobra.Command, args []string) (err error) {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	customNetworkId, _ := cmd.Flags().GetString(CustomNetworkIDFlag)

	// Validate input arguments
	if err = validateDescribeArguments(customNetworkId); err != nil {
		utils.PrintError(err)
		return
	}

	// Validate user is logged in
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
		spinner = sm.AddSpinner(fmt.Sprintf("Describing custom network %s...", customNetworkId))
		sm.Start()
	}

	// Describe
	var customNetwork *openapiclientfleet.FleetCustomNetwork
	customNetwork, err = dataaccess.FleetDescribeCustomNetwork(cmd.Context(), token, customNetworkId)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully described custom network %s", customNetwork.Id))

	// Format and print the custom network details
	formattedCustomNetwork := formatCustomNetwork(customNetwork)

	err = utils.PrintTextTableJsonOutput(output, formattedCustomNetwork)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return
}

func validateDescribeArguments(idFlag string) error {
	if len(idFlag) == 0 {
		return fmt.Errorf("invalid arguments: network ID is required")
	}
	return nil
}
