package snapshot

import (
	"context"
	"fmt"
	"os"
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
	require.NotEmpty(t, build.ProductTierID)

	// List snapshots — should succeed with empty results
	cmd.RootCmd.SetArgs([]string{"snapshot", "list", "--service-id", build.ServiceID, "--environment-id", build.EnvironmentID, "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// List all snapshot types with a product tier filter — should also succeed with empty results
	cmd.RootCmd.SetArgs([]string{
		"snapshot", "list",
		"--service-id", build.ServiceID,
		"--environment-id", build.EnvironmentID,
		"--snapshot-type", "all",
		"--product-tier-id", build.ProductTierID,
		"--output", "json",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Cleanup: delete the service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}

func TestSnapshotRestoreWithTargetingFlags(t *testing.T) {
	testutils.SmokeTest(t)

	serviceID := os.Getenv("SNAPSHOT_RESTORE_TEST_SERVICE_ID")
	environmentID := os.Getenv("SNAPSHOT_RESTORE_TEST_ENVIRONMENT_ID")
	snapshotID := os.Getenv("SNAPSHOT_RESTORE_TEST_SNAPSHOT_ID")
	customNetworkID := os.Getenv("SNAPSHOT_RESTORE_TEST_CUSTOM_NETWORK_ID")
	subscriptionID := os.Getenv("SNAPSHOT_RESTORE_TEST_SUBSCRIPTION_ID")
	if serviceID == "" || environmentID == "" || snapshotID == "" || customNetworkID == "" || subscriptionID == "" {
		t.Skip("set SNAPSHOT_RESTORE_TEST_SERVICE_ID, SNAPSHOT_RESTORE_TEST_ENVIRONMENT_ID, SNAPSHOT_RESTORE_TEST_SNAPSHOT_ID, SNAPSHOT_RESTORE_TEST_CUSTOM_NETWORK_ID, and SNAPSHOT_RESTORE_TEST_SUBSCRIPTION_ID")
	}

	ctx := context.TODO()
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	restoreArgs := []string{
		"snapshot", "restore",
		"--service-id", serviceID,
		"--environment-id", environmentID,
		"--snapshot-id", snapshotID,
		"--custom-network-id", customNetworkID,
		"--subscription-id", subscriptionID,
		"--output", "json",
	}
	if rawParams := os.Getenv("SNAPSHOT_RESTORE_TEST_PARAM_JSON"); rawParams != "" {
		restoreArgs = append(restoreArgs, "--param", rawParams)
	}

	cmd.RootCmd.SetArgs(restoreArgs)
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
