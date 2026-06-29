package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	snapshotTypeFlag := listCmd.Flags().Lookup("snapshot-type")
	require.NotNil(snapshotTypeFlag, "Expected flag 'snapshot-type' not found")
	require.Equal("ManualSnapshot", snapshotTypeFlag.DefValue)

	productTierIDFlag := listCmd.Flags().Lookup("product-tier-id")
	require.NotNil(productTierIDFlag, "Expected flag 'product-tier-id' not found")
}

func TestNormalizeSnapshotType(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectError   bool
		errorContains string
	}{
		{"empty defaults to manual", "", "ManualSnapshot", false, ""},
		{"manual exact", "ManualSnapshot", "ManualSnapshot", false, ""},
		{"manual shorthand", "manual", "ManualSnapshot", false, ""},
		{"automated exact", "AutomatedSnapshot", "AutomatedSnapshot", false, ""},
		{"automated shorthand", "automated", "AutomatedSnapshot", false, ""},
		{"all disables snapshot type filter", "all", "", false, ""},
		{"invalid snapshot type", "FinalSnapshot", "", true, "invalid snapshot type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeSnapshotType(tt.input)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRestoreCommandFlags(t *testing.T) {
	require := require.New(t)

	require.Contains(restoreCmd.Use, "restore")
	require.NotEmpty(restoreCmd.Example)

	flags := []string{"service-id", "environment-id", "snapshot-id", "param", "param-file", "tierversion-override", "network-type", "custom-network-id", "subscription-id", "restore-to-source"}
	for _, flagName := range flags {
		flag := restoreCmd.Flags().Lookup(flagName)
		require.NotNil(flag, "Expected flag '%s' not found", flagName)
	}
}

func TestRestoreCommandFlags_RestoreToSource(t *testing.T) {
	flag := restoreCmd.Flags().Lookup("restore-to-source")
	require.NotNil(t, flag, "Expected flag 'restore-to-source' to be registered")
	assert.Contains(t, flag.Usage, "original source instance")
	assert.Equal(t, "false", flag.DefValue, "restore-to-source should default to false")
}

func TestRestoreCommandHelpText(t *testing.T) {
	assert.Equal(t, "Create an instance by restoring from a snapshot", restoreCmd.Short)
}

func TestRestoreCommandUse_IncludesRestoreToSource(t *testing.T) {
	assert.Contains(t, restoreCmd.Use, "--restore-to-source")
}

func TestRestoreCommandUse_IncludesSubscriptionID(t *testing.T) {
	assert.Contains(t, restoreCmd.Use, "--subscription-id")
}

func TestRestoreCommandUse_IncludesCustomNetworkID(t *testing.T) {
	assert.Contains(t, restoreCmd.Use, "--custom-network-id")
}
