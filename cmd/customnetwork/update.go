package customnetwork

import (
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	updateExample = `# Update a custom network by ID
omnistrate-ctl custom-network update --custom-network-id [custom-network-id] --name [new-custom-network-name]`
)

var updateCmd = &cobra.Command{
	Use:          "update [flags]",
	Short:        "Update a custom network",
	Long:         `This command helps you update an existing custom network.`,
	Example:      updateExample,
	RunE:         runUpdate,
	SilenceUsage: true,
}

func init() {
	updateCmd.Flags().StringP(CustomNetworkIDFlag, "", "", "ID of the custom network")
	updateCmd.Flags().StringP(NameFlag, "", "", "New name of the custom network")

	err := updateCmd.MarkFlagRequired(CustomNetworkIDFlag)
	if err != nil {
		return
	}
}

func runUpdate(cmd *cobra.Command, args []string) (err error) {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	customNetworkId, _ := cmd.Flags().GetString(CustomNetworkIDFlag)
	nameFlag, _ := cmd.Flags().GetString(NameFlag)

	// Validate input arguments
	if err = validateUpdateArguments(customNetworkId); err != nil {
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
		spinner = sm.AddSpinner(fmt.Sprintf("Updating custom network %s...", customNetworkId))
		sm.Start()
	}

	// Gather parameters
	var updatedName *string
	if len(nameFlag) > 0 {
		updatedName = utils.ToPtr(nameFlag)
	}

	// Update
	var customNetwork *openapiclientfleet.FleetCustomNetwork
	customNetwork, err = dataaccess.FleetUpdateCustomNetwork(cmd.Context(), token, customNetworkId, updatedName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully updated custom network %s", customNetwork.Id))

	// Format and print the custom network details
	formattedCustomNetwork := formatCustomNetwork(customNetwork)

	err = utils.PrintTextTableJsonOutput(output, formattedCustomNetwork)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return
}

func validateUpdateArguments(idFlag string) error {
	if len(idFlag) == 0 {
		return fmt.Errorf("invalid arguments: network ID is required")
	}
	return nil
}
