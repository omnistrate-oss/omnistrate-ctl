package snapshot

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestSnapshotListEmpty(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Build a service to get real service/environment IDs
	serviceName := "snapshot-test-" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, build.ServiceID)
	require.NotEmpty(t, build.EnvironmentID)

	// List snapshots — should succeed with empty results
	cmd.RootCmd.SetArgs([]string{"snapshot", "list", "--service-id", build.ServiceID, "--environment-id", build.EnvironmentID, "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Cleanup: delete the service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}

func TestSnapshotDeleteNonExistent(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Build a service to get real service/environment IDs
	serviceName := "snapshot-del-" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, build.ServiceID)
	require.NotEmpty(t, build.EnvironmentID)

	// Delete a non-existent snapshot — should return an error
	cmd.RootCmd.SetArgs([]string{"snapshot", "delete", "snap-nonexistent-id", "--service-id", build.ServiceID, "--environment-id", build.EnvironmentID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(t, err, "deleting a non-existent snapshot should fail")

	// Cleanup: delete the service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}
