package servicesorchestration

import (
	"errors"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe an services orchestration deployment
omctl services-orchestration describe so-abcd1234`
)

var describeCmd = &cobra.Command{
	Use:          "describe [so-id]",
	Short:        "DEPRECATED - Describe an services orchestration deployment",
	Long:         `DEPRECATED - This command helps you describe a services orchestration deployment.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("services orchestration id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	soID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate output flag
	if output != "json" {
		err = errors.New("only json output is supported")
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Describing instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Describe services orchestration
	servicesOrchestration, err := dataaccess.DescribeServicesOrchestration(
		cmd.Context(),
		token,
		soID,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully described services orchestration deployment")
	// Print output
	err = utils.PrintTextTableJsonOutput(output, servicesOrchestration)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
