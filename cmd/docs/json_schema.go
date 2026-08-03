package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	jsonSchemaExample = `# List the schema types that can be requested
omnistrate-ctl docs json-schema

# Get the full Docker Compose spec schema, including every Omnistrate extension
omnistrate-ctl docs json-schema compose --output json

# Get the full Plan spec (ServicePlanSpec) schema
omnistrate-ctl docs json-schema service-plan --output json

# Get the schema for a single compose extension
omnistrate-ctl docs json-schema x-omnistrate-compute --output json
`
)

var jsonSchemaCmd = &cobra.Command{
	Use:          "json-schema [type]",
	Short:        "Get the JSON schema for a spec type",
	Long:         "This command returns the JSON schema for a spec type from the Omnistrate API. If no type is provided, it lists the types that can be requested. Use this to validate a compose or plan spec against the authoritative schema.",
	Example:      jsonSchemaExample,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runJSONSchema,
	SilenceUsage: true,
}

func init() {
	jsonSchemaCmd.Flags().StringP("output", "o", outputJSON, schemaOutputUsage)
}

func runJSONSchema(cmd *cobra.Command, args []string) error {
	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// No type provided, list the types that can be requested. The listing is genuinely
	// tabular, unlike a schema, so it renders as a table unless the caller asked for
	// something specific.
	if len(args) == 0 {
		listOutput := output
		if !cmd.Flags().Changed("output") {
			listOutput = "table"
		}
		if err = utils.PrintTextTableJsonArrayOutput(listOutput, dataaccess.ListJSONSchemaTypes()); err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	schemaType := strings.TrimSpace(args[0])

	// Fetch JSON schema. The type is passed through rather than rejected locally so
	// that types added to the API before this CLI knows about them still work; an
	// unrecognized type gets the known list appended to whatever the API reports.
	schema, err := dataaccess.GetJSONSchema(cmd.Context(), schemaType)
	if err != nil {
		if !dataaccess.IsValidJSONSchemaType(schemaType) {
			err = fmt.Errorf("%w (known types: %s)", err, strings.Join(knownSchemaTypeNames(), ", "))
		}
		utils.PrintError(err)
		return err
	}

	// Print the schema
	if err = utils.PrintTextTableJsonOutput(schemaOutput(output), schema); err != nil {
		utils.PrintError(err)
		return err
	}
	return nil
}

func knownSchemaTypeNames() []string {
	types := dataaccess.ListJSONSchemaTypes()
	names := make([]string, 0, len(types))
	for _, t := range types {
		names = append(names, t.Type)
	}
	return names
}
