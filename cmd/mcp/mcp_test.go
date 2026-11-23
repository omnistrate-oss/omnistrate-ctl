package mcp

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestManagedArtifactSyncCopyIsNotExposed(t *testing.T) {
	root := &cobra.Command{Use: "omnistrate-ctl"}
	managedArtifact := &cobra.Command{Use: "managed-artifact"}
	sync := &cobra.Command{Use: "sync"}
	copyCmd := &cobra.Command{Use: "copy"}
	describeCmd := &cobra.Command{Use: "describe"}
	root.AddCommand(managedArtifact)
	managedArtifact.AddCommand(sync)
	sync.AddCommand(copyCmd, describeCmd)

	selector := config.Selectors[0].CmdSelector
	require.False(t, selector(copyCmd), "copy runs local hook executables and must not be an MCP tool")
	require.True(t, selector(describeCmd))
}
