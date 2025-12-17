package instance

import (
	"slices"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/input"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deleteExample = `# Delete an instance deployment
omnistrate-ctl instance delete instance-abcd1234`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [instance-id] [flags]",
	Short:        "Delete an instance deployment",
	Long:         `This command helps you delete an instance from your account.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Flags().BoolP("yes", "y", false, "Pre-approve the deletion of the instance without prompting for confirmation")
	deleteCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, _ := cmd.Flags().GetString("output")
	yes, _ := cmd.Flags().GetBool("yes")

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Confirm deletion
	if !yes {
		ok, err := prompt.New().Ask("Are you sure you want to delete this instance? (y/n)").
			Input("", input.WithValidateFunc(
				func(input string) error {
					if slices.Contains([]string{"y", "yes", "n", "no"}, strings.ToLower(input)) {
						return nil
					} else {
						return errors.New("invalid input")
					}
				}))
		if err != nil {
			utils.PrintError(err)
			return err
		}

		if !slices.Contains([]string{"y", "yes"}, strings.ToLower(ok)) {
			return nil
		}
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Deleting instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, resourceID, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Delete the instance
	err = dataaccess.DeleteResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully deleted instance")

	return nil
}
