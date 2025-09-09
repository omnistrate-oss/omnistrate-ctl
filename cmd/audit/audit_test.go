package audit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditCommands(t *testing.T) {
	require.Equal(t, "audit", Cmd.Use)
	require.Equal(t, "Audit events and logging management", Cmd.Short)

	// Check that list subcommand is added
	subCommands := []string{"list"}
	for _, subCmd := range subCommands {
		found := false
		for _, cmd := range Cmd.Commands() {
			if cmd.Use == subCmd {
				found = true
				break
			}
		}
		require.True(t, found, "Expected subcommand %s not found", subCmd)
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd := listCmd

	require.Equal(t, "list", cmd.Use)
	require.Equal(t, "List audit events", cmd.Short)

	// Check flags
	flags := []string{
		"next-page-token", "page-size", "service-id", "environment-type",
		"event-source-types", "instance-id", "product-tier-id",
		"start-date", "end-date",
	}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found", flagName)
	}

	// Check shorthand flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentTypeFlag := cmd.Flags().Lookup("environment-type")
	require.Equal(t, "e", environmentTypeFlag.Shorthand)

	instanceIDFlag := cmd.Flags().Lookup("instance-id")
	require.Equal(t, "i", instanceIDFlag.Shorthand)
}