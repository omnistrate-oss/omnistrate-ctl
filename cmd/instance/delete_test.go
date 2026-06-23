package instance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteCommandFlags(t *testing.T) {
	yesFlag := deleteCmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag)
	require.Equal(t, "y", yesFlag.Shorthand)
	require.Equal(t, "false", yesFlag.DefValue)
	require.Equal(t, "bool", yesFlag.Value.Type())

	skipFinalSnapshotFlag := deleteCmd.Flags().Lookup("skip-final-snapshot")
	require.NotNil(t, skipFinalSnapshotFlag)
	require.Equal(t, "false", skipFinalSnapshotFlag.DefValue)
	require.Equal(t, "bool", skipFinalSnapshotFlag.Value.Type())
	require.Contains(t, skipFinalSnapshotFlag.Usage, "without prompting for confirmation")
}
