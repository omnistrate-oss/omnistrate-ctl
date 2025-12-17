package servicesorchestration

import (
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create a services orchestration deployment from a DSL file
omnistrate-ctl services-orchestration create --dsl-file /path/to/dsl.yaml`
)

var createCmd = &cobra.Command{
	Use:          "create --dsl-file=[file-path]",
	Short:        "Create a services orchestration deployment",
	Long:         `This command helps you create a services orchestration deployment, coordinating the creation of multiple services.`,
	Deprecated:   "This command is deprecated and will be removed in a future release.",
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().String("dsl-file", "", "Yaml file containing DSL for services orchestration deployment")

	if err := createCmd.MarkFlagRequired("dsl-file"); err != nil {
		return
	}

	createCmd.Args = cobra.NoArgs // Require no arguments
}

func runCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

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
		msg := "Creating services orchestration..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Read DSL file
	dslFileContent, err := readDslFile(dslFilePath)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	orchestration, err := dataaccess.CreateServicesOrchestration(
		cmd.Context(),
		token,
		dslFileContent,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if orchestration.Id == nil {
		err = errors.New("failed to create services orchestration")
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully created services orchestration")

	// Search for the orchestration
	searchRes, err := dataaccess.DescribeServicesOrchestration(
		cmd.Context(),
		token,
		*orchestration.Id,
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
