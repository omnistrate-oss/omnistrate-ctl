package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	composeSpecExample = `# List all H3 headers in the compose spec documentation with JSON output
omctl docs compose-spec --output json

# Search for a specific tag with JSON output
omctl docs compose-spec "networks" --output json

# Search for specific custom tags with JSON output
omctl docs compose-spec "x-omnistrate-compute" --output json
`
)

var composeSpecCmd = &cobra.Command{
	Use:          "compose-spec [tag]",
	Short:        "Compose spec documentation",
	Long:         "This command returns information about the Omnistrate Docker Compose specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.",
	Example:      composeSpecExample,
	RunE:         runComposeSpec,
	SilenceUsage: true,
}

func init() {
	composeSpecCmd.Flags().StringP("output", "o", "table", "Output format (table|json)")
	composeSpecCmd.Flags().Bool("json-schema-only", false, "Return only the JSON schema for the specified tag")
}

func runComposeSpec(cmd *cobra.Command, args []string) error {
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

	// Use the dataaccess layer to search compose spec sections
	results, err := dataaccess.SearchComposeSpecSections(tag)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(results) == 0 {
		availableTags, err := dataaccess.ListComposeSpecSections()
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
