package cloudnativenetwork

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudNativeNetworkCommandStructure(t *testing.T) {
	// Verify the top-level cloud-native-network command has the expected subcommands
	require.NotNil(t, Cmd)
	assert.Equal(t, "cloud-native-network [operation] [flags]", Cmd.Use)

	subCmds := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	assert.True(t, subCmds["list"], "expected list subcommand")
	assert.True(t, subCmds["sync"], "expected sync subcommand")
	assert.True(t, subCmds["import"], "expected import subcommand")
	assert.True(t, subCmds["remove"], "expected remove subcommand")
}

func TestRemoveCommandRequiresNetworkIDFlag(t *testing.T) {
	require.NotNil(t, removeCmd.Flags().Lookup("network-id"))
}

func TestImportCommandRequiresNetworkIDFlag(t *testing.T) {
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
}

func TestSyncTargetsFromFlags(t *testing.T) {
	targets, err := syncTargetsFromFlags(
		[]string{"us-east-1"},
		[]string{"us-west-2:vpc-abc123"},
	)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Empty(t, targets[0].NetworkID)
	assert.Equal(t, "us-west-2", targets[1].Region)
	assert.Equal(t, "vpc-abc123", targets[1].NetworkID)
}

func TestSyncTargetsFromFlagsRejectsMalformedNetwork(t *testing.T) {
	_, err := syncTargetsFromFlags(nil, []string{"vpc-abc123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected region:network-id")
}
