package instance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBreakpointCommands(t *testing.T) {
	require.NotNil(t, breakpointCmd)
	require.Equal(t, "breakpoint [operation] [flags]", breakpointCmd.Use)
	require.NotNil(t, breakpointListCmd)
	require.Equal(t, "list [instance-id]", breakpointListCmd.Use)

	found := false
	for _, cmd := range breakpointCmd.Commands() {
		if cmd.Name() == "list" {
			found = true
			break
		}
	}
	require.True(t, found, "expected breakpoint list subcommand to be registered")
}
