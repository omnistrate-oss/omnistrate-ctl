package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONSchemaCommand(t *testing.T) {
	require.Equal(t, "json-schema [type]", jsonSchemaCmd.Use)
	require.NotEmpty(t, jsonSchemaCmd.Short)
	require.NotEmpty(t, jsonSchemaCmd.Long)
	require.NotEmpty(t, jsonSchemaCmd.Example)
	require.NotNil(t, jsonSchemaCmd.RunE)
	require.True(t, jsonSchemaCmd.SilenceUsage)
}

func TestJSONSchemaCommandFlags(t *testing.T) {
	outputFlag := jsonSchemaCmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag, "Expected flag 'output' not found")
	require.Equal(t, "o", outputFlag.Shorthand)
	// A JSON schema has no table representation, so json is the default.
	require.Equal(t, "json", outputFlag.DefValue)
}

func TestJSONSchemaCommandAcceptsAtMostOneType(t *testing.T) {
	require.NoError(t, jsonSchemaCmd.Args(jsonSchemaCmd, []string{}))
	require.NoError(t, jsonSchemaCmd.Args(jsonSchemaCmd, []string{"compose"}))
	require.Error(t, jsonSchemaCmd.Args(jsonSchemaCmd, []string{"compose", "service-plan"}))
}

func TestJSONSchemaCommandIsRegistered(t *testing.T) {
	names := make([]string, 0, len(Cmd.Commands()))
	for _, sub := range Cmd.Commands() {
		names = append(names, sub.Name())
	}
	require.Contains(t, names, "json-schema")
	require.Contains(t, names, "compose-spec")
	require.Contains(t, names, "plan-spec")
	require.Contains(t, names, "system-parameters")
	require.Contains(t, names, "search")
}

func TestKnownSchemaTypeNames(t *testing.T) {
	names := knownSchemaTypeNames()
	require.NotEmpty(t, names)
	require.Contains(t, names, "compose")
	require.Contains(t, names, "service-plan")
}
