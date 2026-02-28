package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	planSpecExample = `# List all H3 headers in the plan spec documentation with JSON output
omnistrate-ctl docs plan-spec --output json

# Search for a specific tag with JSON output
omnistrate-ctl docs plan-spec "compute" --output json

# Search for specific schema tags with JSON output
omnistrate-ctl docs plan-spec "helm chart configuration" --output json
`
)

var planSpecCmd = &cobra.Command{
	Use:          "plan-spec [tag]",
	Short:        "Plan spec documentation",
	Long:         "This command returns information about the Omnistrate Plan specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.",
	Example:      planSpecExample,
	RunE:         runPlanSpec,
	SilenceUsage: true,
}

func init() {
	planSpecCmd.Flags().StringP("output", "o", "table", "Output format (table|json)")
	planSpecCmd.Flags().Bool("json-schema-only", false, "Return only the JSON schema for the specified tag")
}

func runPlanSpec(cmd *cobra.Command, args []string) error {
	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	jsonSchemaOnly, err := cmd.Flags().GetBool("json-schema-only")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get the tag from args (optional)
	var tag string
	if len(args) > 0 {
		tag = strings.Join(args, " ")
	}

	// If json-schema-only flag is set, only fetch and return the JSON schema
	if jsonSchemaOnly {
		if tag == "" {
			err := fmt.Errorf("tag is required when using --json-schema-only flag")
			utils.PrintError(err)
			return err
		}

		// Fetch JSON schema
		schema, schemaErr := dataaccess.GetJSONSchema(cmd.Context(), tag)
		if schemaErr != nil {
			utils.PrintError(schemaErr)
			return schemaErr
		}

		// Print the schema
		err = utils.PrintTextTableJsonOutput(output, schema)
		if err != nil {
			utils.PrintError(err)
			return err
		}
		return nil
	}

	// Use the dataaccess layer to search plan spec sections
	results, err := dataaccess.SearchPlanSpecSections(tag)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(results) == 0 {
		availableTags, err := dataaccess.ListPlanSpecSections()
		if err != nil {
			utils.PrintError(err)
			return err
		}
		err = utils.PrintTextTableJsonArrayOutput(output, availableTags)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	} else {
		// Print results
		err = utils.PrintTextTableJsonArrayOutput(output, results)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	}
	return nil
}
