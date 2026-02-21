package snapshot

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestSnapshotListAndDeleteNonExistent(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// List snapshots with a fake service/environment — should return an error (not found)
	cmd.RootCmd.SetArgs([]string{"snapshot", "list", "--service-id", "s-fake123", "--environment-id", "se-fake456"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(t, err, "listing snapshots with non-existent service/environment should fail")

	// Delete a non-existent snapshot — should return an error (not found)
	cmd.RootCmd.SetArgs([]string{"snapshot", "delete", "snap-nonexistent", "--service-id", "s-fake123", "--environment-id", "se-fake456"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(t, err, "deleting a non-existent snapshot should fail")
}
