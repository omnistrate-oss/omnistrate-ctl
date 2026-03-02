package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshotSubcommandsRegistered(t *testing.T) {
	require := require.New(t)

	expectedCommands := []string{
		"list",
		"delete",
		"restore",
	}

	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range expectedCommands {
		require.Contains(actualCommands, expected, "Expected subcommand %s not found", expected)
	}
}

func TestDeleteCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(deleteCmd.Use, "delete")
	require.NotEmpty(deleteCmd.Example)
	require.True(deleteCmd.SilenceUsage)

	serviceIDFlag := deleteCmd.Flags().Lookup("service-id")
	require.NotNil(serviceIDFlag, "Expected flag 'service-id' not found")

	environmentIDFlag := deleteCmd.Flags().Lookup("environment-id")
	require.NotNil(environmentIDFlag, "Expected flag 'environment-id' not found")
}

func TestListCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(listCmd.Use, "list")
	require.Equal("List all snapshots for a service environment", listCmd.Short)
	require.NotEmpty(listCmd.Example)

	serviceIDFlag := listCmd.Flags().Lookup("service-id")
	require.NotNil(serviceIDFlag, "Expected flag 'service-id' not found")

	environmentIDFlag := listCmd.Flags().Lookup("environment-id")
	require.NotNil(environmentIDFlag, "Expected flag 'environment-id' not found")
}

func TestRestoreCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(restoreCmd.Use, "restore")
	require.NotEmpty(restoreCmd.Example)

	flags := []string{"service-id", "environment-id", "snapshot-id", "param", "param-file", "tierversion-override", "network-type"}
	for _, flagName := range flags {
		flag := restoreCmd.Flags().Lookup(flagName)
		require.NotNil(flag, "Expected flag '%s' not found", flagName)
	}
}

func TestFormatSnapshotDisplayTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid RFC3339", "2024-01-15T10:30:00Z", "2024-01-15 10:30:00 UTC"},
		{"empty string", "", ""},
		{"invalid format returns raw", "not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSnapshotDisplayTime(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
