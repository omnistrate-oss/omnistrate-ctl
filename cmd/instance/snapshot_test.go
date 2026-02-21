package instance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshotCommandsRegistered(t *testing.T) {
	require := require.New(t)

	// Verify all snapshot-related subcommands are registered
	snapshotCommands := []string{
		"create-snapshot",
		"delete-snapshot",
		"copy-snapshot",
		"list-snapshots",
		"describe-snapshot",
		"restore",
		"trigger-backup",
	}

	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range snapshotCommands {
		require.Contains(actualCommands, expected, "Expected subcommand %s not found", expected)
	}
}

func TestCreateSnapshotCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("create-snapshot [instance-id]", createSnapshotCmd.Use)
	require.Equal("Create a snapshot for an instance", createSnapshotCmd.Short)
	require.Contains(createSnapshotCmd.Long, "on-demand snapshot")
	require.NotEmpty(createSnapshotCmd.Example)
	require.True(createSnapshotCmd.SilenceUsage)

	// Verify target-region flag exists
	targetRegionFlag := createSnapshotCmd.Flags().Lookup("target-region")
	require.NotNil(targetRegionFlag, "Expected flag 'target-region' not found")
	require.Equal("string", targetRegionFlag.Value.Type())

	// Verify source-snapshot-id flag exists
	sourceSnapshotIDFlag := createSnapshotCmd.Flags().Lookup("source-snapshot-id")
	require.NotNil(sourceSnapshotIDFlag, "Expected flag 'source-snapshot-id' not found")
	require.Equal("string", sourceSnapshotIDFlag.Value.Type())
}

func TestDeleteSnapshotCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("delete-snapshot [snapshot-id] --service-id <service-id> --environment-id <environment-id>", deleteSnapshotCmd.Use)
	require.Equal("Delete an instance snapshot", deleteSnapshotCmd.Short)
	require.NotEmpty(deleteSnapshotCmd.Example)
	require.True(deleteSnapshotCmd.SilenceUsage)

	// Verify service-id and environment-id flags exist
	serviceIDFlag := deleteSnapshotCmd.Flags().Lookup("service-id")
	require.NotNil(serviceIDFlag, "Expected flag 'service-id' not found")
	require.Equal("string", serviceIDFlag.Value.Type())

	environmentIDFlag := deleteSnapshotCmd.Flags().Lookup("environment-id")
	require.NotNil(environmentIDFlag, "Expected flag 'environment-id' not found")
	require.Equal("string", environmentIDFlag.Value.Type())
}

func TestCopySnapshotCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(copySnapshotCmd.Use, "copy-snapshot")
	require.NotEmpty(copySnapshotCmd.Example)

	// Verify flags
	snapshotIDFlag := copySnapshotCmd.Flags().Lookup("snapshot-id")
	require.NotNil(snapshotIDFlag, "Expected flag 'snapshot-id' not found")
	require.Equal("string", snapshotIDFlag.Value.Type())

	targetRegionFlag := copySnapshotCmd.Flags().Lookup("target-region")
	require.NotNil(targetRegionFlag, "Expected flag 'target-region' not found")
	require.Equal("string", targetRegionFlag.Value.Type())
}

func TestListSnapshotsCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("list-snapshots [instance-id]", listSnapshotsCmd.Use)
	require.Equal("List all snapshots for an instance", listSnapshotsCmd.Short)
	require.NotEmpty(listSnapshotsCmd.Example)
}

func TestDescribeSnapshotCommand(t *testing.T) {
	require := require.New(t)

	require.Equal("describe-snapshot [instance-id] [snapshot-id]", describeSnapshotCmd.Use)
	require.Equal("Describe a specific instance snapshot", describeSnapshotCmd.Short)
	require.NotEmpty(describeSnapshotCmd.Example)
}

func TestRestoreCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(restoreCmd.Use, "restore")
	require.NotEmpty(restoreCmd.Example)

	// Verify flags
	flags := []string{"snapshot-id", "param", "param-file", "tierversion-override", "network-type"}
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
