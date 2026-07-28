package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	planSpecExample = `# List every section of the plan spec documentation
omnistrate-ctl docs plan-spec

# Read one section (full text, markdown tables preserved)
omnistrate-ctl docs plan-spec "Root schema"

# Read a section as JSON, for scripting
omnistrate-ctl docs plan-spec "compute" --output json

# Get the JSON schema that defines a section
omnistrate-ctl docs plan-spec "helm chart configuration" --json-schema-only
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
	planSpecCmd.Flags().StringP("output", "o", outputMarkdown, docsOutputUsage)
	planSpecCmd.Flags().Bool("json-schema-only", false, "Return only the JSON schema covering the specified tag. Plan spec tags are definitions within the ServicePlanSpec schema, so this returns that schema; use 'docs json-schema' to request a schema type directly")
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

		// Every plan spec heading documents a definition inside the service-plan
		// schema, so map the tag onto the type the schema API serves
		schemaType, err := dataaccess.ResolvePlanSpecJSONSchemaType(tag)
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
		warnNoTagMatch("plan spec", tag, len(availableTags))
		if output == outputMarkdown {
			tags := make([]string, 0, len(availableTags))
			for _, t := range availableTags {
				tags = append(tags, t.AvailableTag)
			}
			renderAvailableTagsMarkdown(stdout(), "Available plan spec sections", tags,
				`omnistrate-ctl docs plan-spec "%s"`)
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
			renderSpecSectionsMarkdown(stdout(), planSectionsToSpecSections(results))
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
