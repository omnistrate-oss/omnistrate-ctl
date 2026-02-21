package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshotSubcommandsRegistered(t *testing.T) {
	require := require.New(t)

	expectedCommands := []string{
		"copy",
		"list",
		"describe",
		"delete",
		"restore",
		"trigger-backup",
	}

	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range expectedCommands {
		require.Contains(actualCommands, expected, "Expected subcommand %s not found", expected)
	}
}

func TestCopyCommandHasCreateAlias(t *testing.T) {
	require := require.New(t)

	require.Contains(copyCmd.Aliases, "create", "copy command should have 'create' as an alias")
}

func TestCopyCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(copyCmd.Use, "copy")
	require.NotEmpty(copyCmd.Example)
	require.True(copyCmd.SilenceUsage)

	snapshotIDFlag := copyCmd.Flags().Lookup("snapshot-id")
	require.NotNil(snapshotIDFlag, "Expected flag 'snapshot-id' not found")
	require.Equal("string", snapshotIDFlag.Value.Type())

	targetRegionFlag := copyCmd.Flags().Lookup("target-region")
	require.NotNil(targetRegionFlag, "Expected flag 'target-region' not found")
	require.Equal("string", targetRegionFlag.Value.Type())
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

func TestListCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("list [instance-id]", listCmd.Use)
	require.Equal("List all snapshots for an instance", listCmd.Short)
	require.NotEmpty(listCmd.Example)
}

func TestDescribeCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("describe [instance-id] [snapshot-id]", describeCmd.Use)
	require.Equal("Describe a specific instance snapshot", describeCmd.Short)
	require.NotEmpty(describeCmd.Example)
}

func TestRestoreCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(restoreCmd.Use, "restore")
	require.NotEmpty(restoreCmd.Example)

	flags := []string{"snapshot-id", "param", "param-file", "tierversion-override", "network-type"}
	for _, flagName := range flags {
		flag := restoreCmd.Flags().Lookup(flagName)
		require.NotNil(flag, "Expected flag '%s' not found", flagName)
	}
}

func TestTriggerBackupCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("trigger-backup [instance-id]", triggerBackupCmd.Use)
	require.NotEmpty(triggerBackupCmd.Example)
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
