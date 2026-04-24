package cloudnativenetwork

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNetworkIDs(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected []string
	}{
		{
			name:     "single ID",
			raw:      "vpc-abc123",
			expected: []string{"vpc-abc123"},
		},
		{
			name:     "multiple IDs",
			raw:      "vpc-abc123,vpc-def456,vpc-ghi789",
			expected: []string{"vpc-abc123", "vpc-def456", "vpc-ghi789"},
		},
		{
			name:     "trims whitespace",
			raw:      " vpc-abc123 , vpc-def456 ",
			expected: []string{"vpc-abc123", "vpc-def456"},
		},
		{
			name:     "empty string",
			raw:      "",
			expected: nil,
		},
		{
			name:     "only commas",
			raw:      ",,,",
			expected: nil,
		},
		{
			name:     "trailing comma",
			raw:      "vpc-abc123,",
			expected: []string{"vpc-abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNetworkIDs(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloudNativeNetworkCommandStructure(t *testing.T) {
	// Verify the top-level cloud-native-network command has the expected subcommands
	require.NotNil(t, Cmd)
	assert.Equal(t, "cloud-native-network [operation] [flags]", Cmd.Use)

	subCmds := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	assert.True(t, subCmds["sync"], "expected sync subcommand")
	assert.True(t, subCmds["list"], "expected list subcommand")
	assert.True(t, subCmds["import"], "expected import subcommand")
	assert.True(t, subCmds["unimport"], "expected unimport subcommand")
	assert.True(t, subCmds["bulk-import"], "expected bulk-import subcommand")
}

func TestImportCommandRequiresNetworkIDFlag(t *testing.T) {
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
}

func TestUnimportCommandRequiresNetworkIDFlag(t *testing.T) {
	require.NotNil(t, unimportCmd.Flags().Lookup("network-id"))
}

func TestBulkImportCommandRequiresNetworkIDsFlag(t *testing.T) {
	require.NotNil(t, bulkImportCmd.Flags().Lookup("network-ids"))
}

func TestSyncCommandRequiresExactlyOneArg(t *testing.T) {
	require.Error(t, syncCmd.Args(syncCmd, []string{}))
	require.NoError(t, syncCmd.Args(syncCmd, []string{"ac-123"}))
	require.Error(t, syncCmd.Args(syncCmd, []string{"ac-123", "extra"}))
}

func TestListCommandRequiresExactlyOneArg(t *testing.T) {
	require.Error(t, listCmd.Args(listCmd, []string{}))
	require.NoError(t, listCmd.Args(listCmd, []string{"ac-123"}))
	require.Error(t, listCmd.Args(listCmd, []string{"ac-123", "extra"}))
}
