package docs

import (
	"fmt"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	validateExample = `# Validate a compose spec (the spec type is detected from the file)
omnistrate-ctl docs validate --file docker-compose.yaml

# Validate a ServicePlanSpec
omnistrate-ctl docs validate --file spec.yaml

# Force a spec type when detection is ambiguous
omnistrate-ctl docs validate --file spec.yaml --spec-type service-plan

# Machine-readable output, for a pre-commit hook or CI step
omnistrate-ctl docs validate --file spec.yaml --output json
`
)

// specTypeAliases maps the spellings people actually type onto the schema type the
// API serves. `--spec-type` on `build` uses ServicePlanSpec/DockerCompose, so accept
// those here too rather than making callers learn a second vocabulary.
var specTypeAliases = map[string]string{
	"compose":         "compose",
	"dockercompose":   "compose",
	"docker-compose":  "compose",
	"service-plan":    "service-plan",
	"serviceplan":     "service-plan",
	"serviceplanspec": "service-plan",
	"plan":            "service-plan",
	"plan-spec":       "service-plan",
}

var validateCmd = &cobra.Command{
	Use:   "validate --file <spec.yaml>",
	Short: "Validate a spec file against the authoritative JSON schema",
	Long: `This command validates a Docker Compose spec or a ServicePlanSpec against the JSON
schema served by the Omnistrate API, and reports every field that violates it.

Use it to check a spec before building. It catches unknown and misplaced fields,
wrong types, and missing required fields without creating anything in your account.

Two limits are worth knowing. The schema does not enumerate allowed values, so a
field with a valid type but an invalid value (a wrong tenancyType or cloudProvider)
passes — confirm values with 'docs compose-spec' or 'docs plan-spec'. And the compose
schema accepts any 'x-' key, so a misspelled extension name passes here; check
spelling against 'docs compose-spec'.`,
	Example:      validateExample,
	Args:         cobra.NoArgs,
	RunE:         runValidate,
	SilenceUsage: true,
}

func init() {
	validateCmd.Flags().StringP("file", "f", "", "Path to the spec file to validate (required)")
	validateCmd.Flags().String("spec-type", "", "Spec type: compose|service-plan (detected from the file when omitted)")
	validateCmd.Flags().String("schema-file", "", "Validate against a schema on disk instead of fetching one, for offline or pinned-schema use")
	validateCmd.Flags().StringP("output", "o", outputMarkdown, docsOutputUsage)
	if err := validateCmd.MarkFlagRequired("file"); err != nil {
		panic(err)
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	file, err := cmd.Flags().GetString("file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	requestedType, err := cmd.Flags().GetString("spec-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	specYAML, err := os.ReadFile(file)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	schemaFile, err := cmd.Flags().GetString("schema-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// With an explicit schema file the spec type is only a label, so detection is
	// allowed to come up empty rather than failing the run.
	var specType string
	if schemaFile != "" {
		specType = normalizeSpecType(requestedType)
		if specType == "" && strings.TrimSpace(requestedType) != "" {
			err = fmt.Errorf("unsupported --spec-type %q; use compose or service-plan", requestedType)
			utils.PrintError(err)
			return err
		}
		if specType == "" {
			specType = dataaccess.DetectSpecType(specYAML)
		}
	} else if specType, err = resolveSpecType(requestedType, specYAML); err != nil {
		utils.PrintError(err)
		return err
	}

	var result *dataaccess.SpecValidationResult
	if schemaFile != "" {
		result, err = dataaccess.ValidateSpecWithSchemaFile(file, specYAML, specType, schemaFile)
	} else {
		result, err = dataaccess.ValidateSpec(cmd.Context(), file, specYAML, specType)
	}
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if output == outputMarkdown {
		renderValidationResultMarkdown(stdout(), result)
	} else if err = utils.PrintTextTableJsonOutput(schemaOutput(output), result); err != nil {
		utils.PrintError(err)
		return err
	}

	// A spec that violates its schema is a failed check, so exit non-zero to make
	// this usable in a pre-commit hook or a CI step.
	if !result.Valid {
		return fmt.Errorf("%s does not satisfy the %s schema (%d violation(s))",
			file, specType, len(result.Violations))
	}
	return nil
}

// normalizeSpecType resolves an alias to a schema type, or "" if it is unknown or empty.
func normalizeSpecType(requested string) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed == "" {
		return ""
	}
	return specTypeAliases[strings.ToLower(trimmed)]
}

// resolveSpecType honours an explicit --spec-type, otherwise infers it from the file.
func resolveSpecType(requested string, specYAML []byte) (string, error) {
	if trimmed := strings.TrimSpace(requested); trimmed != "" {
		if resolved := normalizeSpecType(trimmed); resolved != "" {
			return resolved, nil
		}
		return "", fmt.Errorf("unsupported --spec-type %q; use compose or service-plan", requested)
	}

	if detected := dataaccess.DetectSpecType(specYAML); detected != "" {
		return detected, nil
	}

	return "", fmt.Errorf("could not tell whether this is a compose spec or a ServicePlanSpec; pass --spec-type compose or --spec-type service-plan")
}
