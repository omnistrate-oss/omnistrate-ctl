package servicesorchestration

import (
	"errors"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	modifyExample = `# Modify a services orchestration deployment from a DSL file
omctl services-orchestration modify so-abcd1234 --dsl-file /path/to/dsl.yaml`
)

var modifyCmd = &cobra.Command{
	Use:          "modify [so-id] -dsl-file=[file-path]",
	Short:        "DEPRECATED - Modify a services orchestration deployment",
	Long:         `DEPRECATED - This command helps you modify a services orchestration deployment, coordinating the modification of multiple services.`,
	Example:      modifyExample,
	RunE:         runModify,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument

	modifyCmd.Flags().String("dsl-file", "", "Yaml file containing DSL for services orchestration deployment")

	if err := modifyCmd.MarkFlagRequired("dsl-file"); err != nil {
		return
	}
}

func runModify(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("services orchestration id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	soID := args[0]

	// Retrieve flags
	dslFilePath, err := cmd.Flags().GetString("dsl-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
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
		msg := "Modifying services orchestration..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Read DSL file
	dslFileContent, err := readDslFile(dslFilePath)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	err = dataaccess.ModifyServicesOrchestration(
		cmd.Context(),
		token,
		soID,
		dslFileContent,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully modified services orchestration")

	// Search for the orchestration
	searchRes, err := dataaccess.DescribeServicesOrchestration(
		cmd.Context(),
		token,
		soID,
	)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, searchRes); err != nil {
		return err
	}

	return nil
}
