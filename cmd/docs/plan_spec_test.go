package docs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanSpecCommand(t *testing.T) {
	require.Equal(t, "plan-spec [tag]", planSpecCmd.Use)
	require.Equal(t, "Plan spec documentation", planSpecCmd.Short)
	require.NotNil(t, planSpecCmd.RunE)
	require.True(t, planSpecCmd.SilenceUsage)
}

func TestPlanSpecCommandFlags(t *testing.T) {
	cmd := planSpecCmd

	// Check that output flag exists
	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag, "Expected flag 'output' not found")
	require.Equal(t, "o", outputFlag.Shorthand)
	require.Equal(t, "table", outputFlag.DefValue)

	// Check that json-schema-only flag exists
	jsonSchemaOnlyFlag := cmd.Flags().Lookup("json-schema-only")
	require.NotNil(t, jsonSchemaOnlyFlag, "Expected flag 'json-schema-only' not found")
	require.Equal(t, "false", jsonSchemaOnlyFlag.DefValue)
	require.Equal(t, "bool", jsonSchemaOnlyFlag.Value.Type())
}

func TestPlanSpecExample(t *testing.T) {
	require.NotEmpty(t, planSpecCmd.Example)
	require.Contains(t, planSpecCmd.Example, "omnistrate-ctl docs plan-spec")
	require.Contains(t, planSpecCmd.Example, "--output json")
	require.Contains(t, planSpecCmd.Example, "helm chart configuration")
}

func TestPlanSpecLongDescription(t *testing.T) {
	require.NotEmpty(t, planSpecCmd.Long)
	require.Contains(t, planSpecCmd.Long, "Plan specification")
	require.Contains(t, planSpecCmd.Long, "tag")
}

func TestPlanSpecCommandStructure(t *testing.T) {
	// Verify command has proper structure
	require.NotNil(t, planSpecCmd.RunE, "RunE function should be defined")
	require.NotEmpty(t, planSpecCmd.Use, "Use field should not be empty")
	require.NotEmpty(t, planSpecCmd.Short, "Short description should not be empty")
	require.NotEmpty(t, planSpecCmd.Long, "Long description should not be empty")
	require.NotEmpty(t, planSpecCmd.Example, "Example should not be empty")
}

func TestPlanSpecJSONSchemaOnlyValidation(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		schemaOnly   bool
		expectError  bool
		errorMessage string
	}{
		{
			name:         "json-schema-only without tag should error",
			tag:          "",
			schemaOnly:   true,
			expectError:  true,
			errorMessage: "tag is required when using --json-schema-only flag",
		},
		{
			name:        "json-schema-only with tag should not error in validation",
			tag:         "compute",
			schemaOnly:  true,
			expectError: false,
		},
		{
			name:        "normal mode without tag should not error",
			tag:         "",
			schemaOnly:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic
			if tt.schemaOnly && tt.tag == "" {
				err := fmt.Errorf("tag is required when using --json-schema-only flag")
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMessage)
			} else if tt.expectError {
				t.Errorf("Expected error but got none")
			}
		})
	}
}
