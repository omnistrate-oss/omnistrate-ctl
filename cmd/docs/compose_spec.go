package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	composeSpecExample = `# List every tag and extension in the compose spec documentation
omnistrate-ctl docs compose-spec

# Read one tag (full text, examples and markdown tables preserved)
omnistrate-ctl docs compose-spec "x-omnistrate-compute"

# Read a tag as JSON, for scripting
omnistrate-ctl docs compose-spec "networks" --output json

# Get the JSON schema covering a tag, including nested tags
omnistrate-ctl docs compose-spec "x-omnistrate-capabilities.sidecars" --json-schema-only
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
	composeSpecCmd.Flags().StringP("output", "o", outputMarkdown, docsOutputUsage)
	composeSpecCmd.Flags().Bool("json-schema-only", false, "Return only the JSON schema for the specified tag. Nested tags resolve to their root extension; use 'docs json-schema' to list every schema type or request one directly")
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

		// Documentation headings are not schema type names, so map the tag onto the
		// type the schema API serves before asking for it
		schemaType, err := dataaccess.ResolveComposeSpecJSONSchemaType(tag)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		// Fetch JSON schema
		schema, schemaErr := dataaccess.GetJSONSchema(cmd.Context(), schemaType)
		if schemaErr != nil {
			utils.PrintError(schemaErr)
			return schemaErr
		}

		// A JSON schema has no table representation, so markdown falls back to JSON
		err = utils.PrintTextTableJsonOutput(schemaOutput(output), schema)
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
		warnNoTagMatch("compose spec", tag, len(availableTags))
		if output == outputMarkdown {
			tags := make([]string, 0, len(availableTags))
			for _, t := range availableTags {
				tags = append(tags, t.AvailableTag)
			}
			renderAvailableTagsMarkdown(stdout(), "Available compose spec tags", tags,
				`omnistrate-ctl docs compose-spec "%s"`)
			return nil
		}
		err = utils.PrintTextTableJsonArrayOutput(output, availableTags)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	} else {
		// Print results
		if output == outputMarkdown {
			renderSpecSectionsMarkdown(stdout(), composeSectionsToSpecSections(results))
			return nil
		}
		err = utils.PrintTextTableJsonArrayOutput(output, results)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	}
	return nil
}
