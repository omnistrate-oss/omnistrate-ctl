package cloudnativenetwork

import (
	"testing"

	"github.com/spf13/cobra"
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
	assert.True(t, subCmds["host-cluster"], "expected host-cluster subcommand")
}

func TestRemoveCommandRequiresRegionAndNetworkIDFlags(t *testing.T) {
	removeCmd := findSubCommand(t, Cmd, "remove")
	require.NotNil(t, removeCmd.Flags().Lookup("region"))
	require.NotNil(t, removeCmd.Flags().Lookup("network-id"))
}

func TestImportCommandRequiresRegionAndNetworkIDFlags(t *testing.T) {
	importCmd := findSubCommand(t, Cmd, "import")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
}

func TestSyncCommandSupportsRegionAndNetworkIDFlags(t *testing.T) {
	syncCmd := findSubCommand(t, Cmd, "sync")
	require.NotNil(t, syncCmd.Flags().Lookup("region"))
	require.NotNil(t, syncCmd.Flags().Lookup("network-id"))
	require.NotNil(t, syncCmd.Flags().Lookup("network"))
}

func TestHostClusterImportCommandRequiresFlags(t *testing.T) {
	hostClusterCmd := findSubCommand(t, Cmd, "host-cluster")
	importCmd := findSubCommand(t, hostClusterCmd, "import")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
	require.NotNil(t, importCmd.Flags().Lookup("host-cluster-name"))
}

func TestValidateHostClusterImportFlags(t *testing.T) {
	require.NoError(t, validateHostClusterImportFlags("us-east-1", "vpc-abc123", "customer-eks"))

	err := validateHostClusterImportFlags("", "vpc-abc123", "customer-eks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region cannot be empty")

	err = validateHostClusterImportFlags("us-east-1", "", "customer-eks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id cannot be empty")

	err = validateHostClusterImportFlags("us-east-1", "vpc-abc123", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host-cluster-name cannot be empty")
}

func TestImportTargetsFromFlags(t *testing.T) {
	targets, err := importTargetsFromFlags("us-east-1", []string{"vpc-abc123", "vpc-def456"})
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Equal(t, "vpc-abc123", targets[0].NetworkID)
	assert.Equal(t, "us-east-1", targets[1].Region)
	assert.Equal(t, "vpc-def456", targets[1].NetworkID)
}

func TestImportTargetsFromFlagsRejectsMissingRegion(t *testing.T) {
	_, err := importTargetsFromFlags("", []string{"vpc-abc123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region cannot be empty")
}

func TestImportTargetsFromFlagsRejectsMissingNetworkID(t *testing.T) {
	_, err := importTargetsFromFlags("us-east-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one network-id is required")
}

func TestSyncTargetsFromFlags(t *testing.T) {
	targets, err := syncTargetsFromFlags(
		[]string{"us-east-1"},
		nil,
		[]string{"us-west-2:vpc-abc123"},
	)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Empty(t, targets[0].NetworkID)
	assert.Equal(t, "us-west-2", targets[1].Region)
	assert.Equal(t, "vpc-abc123", targets[1].NetworkID)
}

func TestSyncTargetsFromFlagsWithRegionAndNetworkID(t *testing.T) {
	targets, err := syncTargetsFromFlags(
		[]string{"us-east-1"},
		[]string{"vpc-abc123", "vpc-def456"},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "us-east-1", targets[0].Region)
	assert.Equal(t, "vpc-abc123", targets[0].NetworkID)
	assert.Equal(t, "us-east-1", targets[1].Region)
	assert.Equal(t, "vpc-def456", targets[1].NetworkID)
}

func TestSyncTargetsFromFlagsRejectsNetworkIDWithoutSingleRegion(t *testing.T) {
	_, err := syncTargetsFromFlags(nil, []string{"vpc-abc123"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id requires exactly one region")

	_, err = syncTargetsFromFlags([]string{"us-east-1", "us-west-2"}, []string{"vpc-abc123"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network-id requires exactly one region")
}

func TestSyncTargetsFromFlagsRejectsMalformedNetwork(t *testing.T) {
	_, err := syncTargetsFromFlags(nil, nil, []string{"vpc-abc123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected region:network-id")
}

func findSubCommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	t.Fatalf("expected %s subcommand", name)
	return nil
}
