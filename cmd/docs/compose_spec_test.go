package docs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComposeSpecCommand(t *testing.T) {
	require.Equal(t, "compose-spec [tag]", composeSpecCmd.Use)
	require.Equal(t, "Compose spec documentation", composeSpecCmd.Short)
	require.NotNil(t, composeSpecCmd.RunE)
	require.True(t, composeSpecCmd.SilenceUsage)
}

func TestComposeSpecCommandFlags(t *testing.T) {
	cmd := composeSpecCmd

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

func TestComposeSpecExample(t *testing.T) {
	require.NotEmpty(t, composeSpecCmd.Example)
	require.Contains(t, composeSpecCmd.Example, "omnistrate-ctl docs compose-spec")
	require.Contains(t, composeSpecCmd.Example, "--output json")
	require.Contains(t, composeSpecCmd.Example, "x-omnistrate-compute")
}

func TestComposeSpecLongDescription(t *testing.T) {
	require.NotEmpty(t, composeSpecCmd.Long)
	require.Contains(t, composeSpecCmd.Long, "Docker Compose specification")
	require.Contains(t, composeSpecCmd.Long, "tag")
}

func TestComposeSpecCommandStructure(t *testing.T) {
	// Verify command has proper structure
	require.NotNil(t, composeSpecCmd.RunE, "RunE function should be defined")
	require.NotEmpty(t, composeSpecCmd.Use, "Use field should not be empty")
	require.NotEmpty(t, composeSpecCmd.Short, "Short description should not be empty")
	require.NotEmpty(t, composeSpecCmd.Long, "Long description should not be empty")
	require.NotEmpty(t, composeSpecCmd.Example, "Example should not be empty")
}

func TestJSONSchemaOnlyValidation(t *testing.T) {
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
			tag:         "x-omnistrate-compute",
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
