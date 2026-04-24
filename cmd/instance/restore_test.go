package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreCommandFlags(t *testing.T) {
	require.Contains(t, restoreCmd.Use, "restore")
	require.NotEmpty(t, restoreCmd.Example)
	require.True(t, restoreCmd.SilenceUsage)
}

func TestRestoreCommandFlags_AllExpected(t *testing.T) {
	expectedFlags := []string{
		"snapshot-id", "param", "param-file",
		"tierversion-override", "network-type", "restore-to-source",
	}
	for _, flagName := range expectedFlags {
		flag := restoreCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Expected flag '%s' not found", flagName)
	}
}

func TestRestoreCommandFlags_RestoreToSource(t *testing.T) {
	flag := restoreCmd.Flags().Lookup("restore-to-source")
	require.NotNil(t, flag, "Expected flag 'restore-to-source' to be registered")
	assert.Contains(t, flag.Usage, "original source instance")
	assert.Equal(t, "false", flag.DefValue, "restore-to-source should default to false")
}

func TestRestoreCommandFlags_SnapshotIDRequired(t *testing.T) {
	flag := restoreCmd.Flags().Lookup("snapshot-id")
	require.NotNil(t, flag)

	annotations := flag.Annotations
	_, isRequired := annotations["cobra_annotation_bash_completion_one_required_flag"]
	assert.True(t, isRequired, "snapshot-id should be a required flag")
}

func TestRestoreCommandHelpText(t *testing.T) {
	assert.Equal(t, "Restore an instance from a snapshot", restoreCmd.Short)
	assert.Contains(t, restoreCmd.Long, "restore-to-source")
	assert.Contains(t, restoreCmd.Long, "original source instance")
}

func TestRestoreCommandExample_ContainsRestoreToSource(t *testing.T) {
	assert.Contains(t, restoreCmd.Example, "--restore-to-source")
	assert.Contains(t, restoreCmd.Example, "original source instance")
}

func TestRestoreCommandUse_IncludesRestoreToSource(t *testing.T) {
	assert.Contains(t, restoreCmd.Use, "--restore-to-source")
}
