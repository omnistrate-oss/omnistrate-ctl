package build

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDeleteDeprecatedResourceCommandStructure(t *testing.T) {
	cmd, _, err := BuildCmd.Find([]string{"delete-deprecated-resource"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "delete-deprecated-resource", cmd.Name())
	require.Contains(t, cmd.Short, "Delete a deprecated resource")
	require.Contains(t, cmd.Long, "already deprecated")

	serviceFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceFlag)
	require.Equal(t, "", serviceFlag.DefValue)

	require.Nil(t, cmd.Flags().Lookup("product-tier-id"))

	resourceFlag := cmd.Flags().Lookup("resource-id")
	require.NotNil(t, resourceFlag)
	require.Equal(t, "", resourceFlag.DefValue)

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	require.Equal(t, "false", dryRunFlag.DefValue)

	yesFlag := cmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag)
	require.Equal(t, "false", yesFlag.DefValue)
	require.Equal(t, "y", yesFlag.Shorthand)
}

func TestConfirmDeleteDeprecatedResource(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		confirmed bool
	}{
		{name: "yes", input: "yes\n", confirmed: true},
		{name: "y", input: "y\n", confirmed: true},
		{name: "no", input: "no\n", confirmed: false},
		{name: "empty", input: "\n", confirmed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confirmed, err := confirmDeleteDeprecatedResource(strings.NewReader(tt.input), io.Discard, "r-123")
			require.NoError(t, err)
			require.Equal(t, tt.confirmed, confirmed)
		})
	}
}
