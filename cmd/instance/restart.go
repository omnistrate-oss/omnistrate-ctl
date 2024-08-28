package instance

import (
	"fmt"
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate/ctl/dataaccess"
	"github.com/omnistrate/ctl/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	restartExample = `  # Restart an instance deployment
  omctl instance restart instance-abcd1234`
)

var restartCmd = &cobra.Command{
	Use:          "restart [instance-id]",
	Short:        "Restart an instance deployment for your service",
	Long:         `This command helps you restart the instance for your service.`,
	Example:      restartExample,
	RunE:         runRestart,
	SilenceUsage: true,
}

func init() {

	restartCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runRestart(cmd *cobra.Command, args []string) error {
	defer utils.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := utils.GetToken()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Restarting instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, _, resourceID, err := getInstance(token, instanceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Restart instance
	err = dataaccess.RestartResourceInstance(token, serviceID, environmentID, resourceID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully restarted instance")

	// Search for the instance
	searchRes, err := dataaccess.SearchInventory(token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		err = errors.New("failed to find the restarted instance")
		utils.PrintError(err)
		return err
	}

	// Format instance
	formattedInstance := formatInstance(searchRes.ResourceInstanceResults[0], false)

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedInstance); err != nil {
		return err
	}

	return nil
}
