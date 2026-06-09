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

	assert.True(t, subCmds["remove"], "expected remove subcommand")
	assert.True(t, subCmds["deployment-cell"], "expected deployment-cell subcommand")
}

func TestRemoveCommandRequiresNetworkIDFlag(t *testing.T) {
	require.NotNil(t, removeCmd.Flags().Lookup("network-id"))
}

func TestDeploymentCellCommandStructure(t *testing.T) {
	require.NotNil(t, deploymentCellCmd)

	subCmds := make(map[string]*cobra.Command)
	for _, sub := range deploymentCellCmd.Commands() {
		subCmds[sub.Name()] = sub
	}

	importCmd := subCmds["import"]
	require.NotNil(t, importCmd, "expected import subcommand")
	require.NotNil(t, importCmd.Flags().Lookup("region"))
	require.NotNil(t, importCmd.Flags().Lookup("network-id"))
	require.NotNil(t, importCmd.Flags().Lookup("name"))
}
