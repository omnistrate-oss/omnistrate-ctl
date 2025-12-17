package docs

import (
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	systemParametersExample = `# Get the JSON schema for system parameters
omnistrate-ctl docs system-parameters

# Get the JSON schema for system parameters with JSON output
omnistrate-ctl docs system-parameters --output json
`
)

var systemParametersCmd = &cobra.Command{
	Use:          "system-parameters",
	Short:        "Get the JSON schema for system parameters",
	Long:         "This command returns the JSON schema for system parameters from the Omnistrate API.",
	Example:      systemParametersExample,
	RunE:         runSystemParameters,
	SilenceUsage: true,
}

func init() {
	systemParametersCmd.Flags().StringP("output", "o", "table", "Output format (table|json)")
}

func runSystemParameters(cmd *cobra.Command, args []string) error {
	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Fetch JSON schema for system-parameters type
	schema, err := dataaccess.GetJSONSchema(cmd.Context(), "system-parameters")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print the schema
	err = utils.PrintTextTableJsonOutput(output, schema)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	return nil
}
