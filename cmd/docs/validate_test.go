package docs

import (
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	require.Equal(t, "validate --file <spec.yaml>", validateCmd.Use)
	require.NotEmpty(t, validateCmd.Short)
	require.NotEmpty(t, validateCmd.Long)
	require.NotEmpty(t, validateCmd.Example)
	require.NotNil(t, validateCmd.RunE)
	require.True(t, validateCmd.SilenceUsage)
}

func TestValidateCommandFlags(t *testing.T) {
	fileFlag := validateCmd.Flags().Lookup("file")
	require.NotNil(t, fileFlag)
	require.Equal(t, "f", fileFlag.Shorthand)

	require.NotNil(t, validateCmd.Flags().Lookup("spec-type"))
	require.NotNil(t, validateCmd.Flags().Lookup("schema-file"))

	outputFlag := validateCmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	require.Equal(t, outputMarkdown, outputFlag.DefValue)
}

func TestValidateCommandTakesNoPositionalArgs(t *testing.T) {
	require.NoError(t, validateCmd.Args(validateCmd, []string{}))
	require.Error(t, validateCmd.Args(validateCmd, []string{"spec.yaml"}),
		"the spec is passed with --file, so a positional path should be rejected rather than ignored")
}

func TestValidateCommandIsRegistered(t *testing.T) {
	names := make([]string, 0, len(Cmd.Commands()))
	for _, sub := range Cmd.Commands() {
		names = append(names, sub.Name())
	}
	require.Contains(t, names, "validate")
}

func TestNormalizeSpecType(t *testing.T) {
	// Aliases exist so callers can reuse the vocabulary `build --spec-type` uses.
	for _, alias := range []string{"compose", "Compose", "DockerCompose", "docker-compose"} {
		assert.Equal(t, "compose", normalizeSpecType(alias), alias)
	}
	for _, alias := range []string{"service-plan", "ServicePlanSpec", "serviceplan", "plan", "plan-spec"} {
		assert.Equal(t, "service-plan", normalizeSpecType(alias), alias)
	}
	assert.Equal(t, "", normalizeSpecType(""))
	assert.Equal(t, "", normalizeSpecType("   "))
	assert.Equal(t, "", normalizeSpecType("nonsense"))
}

func TestResolveSpecTypeHonoursExplicitFlag(t *testing.T) {
	// An explicit flag wins over what the file looks like.
	composeLooking := []byte("services:\n  api:\n    image: nginx\n")
	got, err := resolveSpecType("service-plan", composeLooking)
	require.NoError(t, err)
	assert.Equal(t, "service-plan", got)
}

func TestResolveSpecTypeDetectsFromFile(t *testing.T) {
	got, err := resolveSpecType("", []byte("name: p\nservices:\n  - name: db\n"))
	require.NoError(t, err)
	assert.Equal(t, "service-plan", got)

	got, err = resolveSpecType("", []byte("services:\n  api:\n    image: nginx\n"))
	require.NoError(t, err)
	assert.Equal(t, "compose", got)
}

func TestResolveSpecTypeRejectsUnknownFlag(t *testing.T) {
	_, err := resolveSpecType("helm", []byte("services:\n  api:\n    image: nginx\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compose or service-plan")
}

func TestResolveSpecTypeAsksWhenAmbiguous(t *testing.T) {
	_, err := resolveSpecType("", []byte("name: just a name\n"))
	require.Error(t, err)
	// The error has to tell the caller how to proceed.
	assert.Contains(t, err.Error(), "--spec-type")
}

func TestRenderValidationResultMarkdownValid(t *testing.T) {
	var out strings.Builder
	renderValidationResultMarkdown(&out, &dataaccess.SpecValidationResult{
		File: "spec.yaml", SpecType: "service-plan", Valid: true,
	})

	rendered := out.String()
	assert.Contains(t, rendered, "spec.yaml is valid against the service-plan schema.")
	assert.NotContains(t, rendered, "violation")
}

func TestRenderValidationResultMarkdownInvalid(t *testing.T) {
	var out strings.Builder
	renderValidationResultMarkdown(&out, &dataaccess.SpecValidationResult{
		File:     "spec.yaml",
		SpecType: "compose",
		Valid:    false,
		Violations: []dataaccess.SpecViolation{
			{Path: "/services/api/x-omnistrate-api-params/0", Message: "additional properties 'min' not allowed"},
			{Path: "/x-omnistrate-service-plan/tenancyType", Message: "value must be one of 'CUSTOM_TENANCY'"},
		},
	})

	rendered := out.String()
	assert.Contains(t, rendered, "2 violation(s)")
	assert.Contains(t, rendered, "/services/api/x-omnistrate-api-params/0")
	assert.Contains(t, rendered, "additional properties 'min' not allowed")
	assert.Contains(t, rendered, "/x-omnistrate-service-plan/tenancyType")
	// The report must say what it cannot detect, so a clean run is not over-trusted.
	assert.Contains(t, rendered, "Invalid *values*")
}
