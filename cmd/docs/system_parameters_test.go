package docs

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemParametersCommand(t *testing.T) {
	require.Equal(t, "system-parameters", systemParametersCmd.Use)
	require.Equal(t, "Get the JSON schema for system parameters", systemParametersCmd.Short)
	require.NotNil(t, systemParametersCmd.RunE)
	require.True(t, systemParametersCmd.SilenceUsage)
}

func TestSystemParametersCommandFlags(t *testing.T) {
	cmd := systemParametersCmd

	// Check that output flag exists
	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag, "Expected flag 'output' not found")
	require.Equal(t, "o", outputFlag.Shorthand)
	require.Equal(t, "table", outputFlag.DefValue)
}

func TestSystemParametersExample(t *testing.T) {
	require.NotEmpty(t, systemParametersCmd.Example)
	require.Contains(t, systemParametersCmd.Example, "omnistrate-ctl docs system-parameters")
	require.Contains(t, systemParametersCmd.Example, "--output json")
}

func TestSystemParametersLongDescription(t *testing.T) {
	require.NotEmpty(t, systemParametersCmd.Long)
	require.Contains(t, systemParametersCmd.Long, "JSON schema")
	require.Contains(t, systemParametersCmd.Long, "system parameters")
}

func TestSystemParametersCommandStructure(t *testing.T) {
	// Verify command has proper structure
	require.NotNil(t, systemParametersCmd.RunE, "RunE function should be defined")
	require.NotEmpty(t, systemParametersCmd.Use, "Use field should not be empty")
	require.NotEmpty(t, systemParametersCmd.Short, "Short description should not be empty")
	require.NotEmpty(t, systemParametersCmd.Long, "Long description should not be empty")
	require.NotEmpty(t, systemParametersCmd.Example, "Example should not be empty")
}

func TestSystemParametersCommandExecution(t *testing.T) {
	// Buffer to capture output
	buf := new(bytes.Buffer)

	// Create the parent docs command
	docsCmd := Cmd
	docsCmd.SetOut(buf)
	docsCmd.SetErr(buf)

	// Test help for system-parameters subcommand
	docsCmd.SetArgs([]string{"system-parameters", "--help"})
	err := docsCmd.ExecuteContext(context.Background())
	require.NoError(t, err, "Help command should work")
	require.Contains(t, buf.String(), "JSON schema")
	require.Contains(t, buf.String(), "system parameters")
	require.Contains(t, buf.String(), "--output")
}

func TestSystemParametersCommandWithJSONOutput(t *testing.T) {
	// Skip this test if not connected to API
	t.Skip("Skipping TestSystemParametersCommandWithJSONOutput as it requires API connection")

	// Create a fresh command instance for testing
	cmd := systemParametersCmd

	// Buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test with JSON output
	cmd.SetArgs([]string{"--output", "json"})
	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, "Command should execute successfully")

	// Verify output is valid JSON
	var result interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "Output should be valid JSON")
	require.NotNil(t, result, "Result should not be nil")
}
