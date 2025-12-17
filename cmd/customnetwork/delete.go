package customnetwork

import (
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	deleteExample = `# Delete a custom network by ID
omnistrate-ctl custom-network delete --custom-network-id [custom-network-id]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [custom-network-name] [flags]",
	Short:        "Deletes a custom network",
	Long:         `This command helps you delete an existing custom network.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().StringP(CustomNetworkIDFlag, "", "", "ID of the custom network")

	err := describeCmd.MarkFlagRequired(CustomNetworkIDFlag)
	if err != nil {
		return
	}
}

func runDelete(cmd *cobra.Command, args []string) (err error) {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	customNetworkId, _ := cmd.Flags().GetString(CustomNetworkIDFlag)

	// Validate input arguments
	if err = validateDeleteArguments(customNetworkId); err != nil {
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
		spinner = sm.AddSpinner(fmt.Sprintf("Deleting custom network %s...", customNetworkId))
		sm.Start()
	}

	// Delete
	err = dataaccess.FleetDeleteCustomNetwork(cmd.Context(), token, customNetworkId)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully deleted custom network %s", customNetworkId))
	return
}

func validateDeleteArguments(idFlag string) error {
	if len(idFlag) == 0 {
		return fmt.Errorf("invalid arguments: network ID is required")
	}
	return nil
}
