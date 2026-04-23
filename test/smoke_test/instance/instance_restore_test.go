package instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

// TestInstanceUndeleteWithInstanceID tests the --instance-id flag on instance create.
// It creates an instance, deletes it, then restores it using create --instance-id,
// verifying the same instance ID is returned.
func TestInstanceUndeleteWithInstanceID(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Build a service
	serviceName := "undelete-test-" + uuid.NewString()[:8]
	log.Debug().Msgf("Building service %s...", serviceName)
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, build.ServiceID)
	require.NotEmpty(t, build.EnvironmentID)

	// Create instance
	log.Debug().Msg("Creating instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--param", `{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}`,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	originalInstanceID := instance.InstanceID
	require.NotEmpty(t, originalInstanceID, "expected instance ID to be set after create")

	// Wait for instance to be RUNNING
	log.Debug().Msg("Waiting for instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, originalInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Delete the instance
	log.Debug().Msg("Deleting instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", originalInstanceID, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Wait for the instance to be fully deleted
	log.Debug().Msg("Waiting for instance to be deleted...")
	waitForInstanceDeletion(t, ctx, originalInstanceID)

	// Restore (undelete) the instance using --instance-id
	log.Debug().Msg("Restoring instance using --instance-id...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		fmt.Sprintf("--instance-id=%s", originalInstanceID),
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	restoredInstanceID := instance.InstanceID
	require.Equal(t, originalInstanceID, restoredInstanceID, "undeleted instance should have the same ID")

	// Wait for the restored instance to reach RUNNING
	log.Debug().Msg("Waiting for restored instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, restoredInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Cleanup: delete instance and service
	log.Debug().Msg("Cleaning up...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", restoredInstanceID, "--yes"})
	_ = cmd.RootCmd.ExecuteContext(ctx)
	waitForInstanceDeletion(t, ctx, restoredInstanceID)

	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	_ = cmd.RootCmd.ExecuteContext(ctx)
}

// TestSnapshotRestoreToSource tests the --restore-to-source flag on snapshot restore.
// It creates an instance, triggers a backup, deletes the instance, then restores
// from snapshot with --restore-to-source, verifying the same instance ID is restored.
func TestSnapshotRestoreToSource(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	token, err := config.GetToken()
	require.NoError(t, err)

	// Build a service
	serviceName := "restore-src-" + uuid.NewString()[:8]
	log.Debug().Msgf("Building service %s...", serviceName)
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	serviceID := build.ServiceID
	environmentID := build.EnvironmentID
	require.NotEmpty(t, serviceID)
	require.NotEmpty(t, environmentID)

	// Create instance
	log.Debug().Msg("Creating instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--param", `{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}`,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	originalInstanceID := instance.InstanceID
	require.NotEmpty(t, originalInstanceID)

	// Wait for RUNNING
	log.Debug().Msg("Waiting for instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, originalInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Trigger backup
	log.Debug().Msg("Triggering backup...")
	backupResult, err := dataaccess.TriggerResourceInstanceAutoBackup(ctx, token, serviceID, environmentID, originalInstanceID)
	require.NoError(t, err)
	snapshotID := backupResult.GetSnapshotId()
	require.NotEmpty(t, snapshotID, "expected snapshot ID from backup trigger")
	log.Debug().Msgf("Snapshot ID: %s", snapshotID)

	// Wait for snapshot to complete
	log.Debug().Msg("Waiting for snapshot to complete...")
	waitForSnapshotCompletion(t, ctx, token, serviceID, environmentID, originalInstanceID, snapshotID)

	// Delete the instance
	log.Debug().Msg("Deleting instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", originalInstanceID, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	waitForInstanceDeletion(t, ctx, originalInstanceID)

	// Restore from snapshot with --restore-to-source
	log.Debug().Msg("Restoring from snapshot with --restore-to-source...")
	cmd.RootCmd.SetArgs([]string{"snapshot", "restore",
		"--service-id", serviceID,
		"--environment-id", environmentID,
		"--snapshot-id", snapshotID,
		"--restore-to-source",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Wait for restored instance to reach RUNNING
	log.Debug().Msg("Waiting for restored instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, originalInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Verify describe still works and returns the same instance ID
	cmd.RootCmd.SetArgs([]string{"instance", "describe", originalInstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Cleanup: delete instance and service
	log.Debug().Msg("Cleaning up...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", originalInstanceID, "--yes"})
	_ = cmd.RootCmd.ExecuteContext(ctx)
	waitForInstanceDeletion(t, ctx, originalInstanceID)

	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	_ = cmd.RootCmd.ExecuteContext(ctx)
}

// TestInstanceRestoreToSource tests the --restore-to-source flag on instance restore.
// Uses the instance-based restore path (instance restore <id> --snapshot-id <sid> --restore-to-source).
func TestInstanceRestoreToSource(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	token, err := config.GetToken()
	require.NoError(t, err)

	// Build a service
	serviceName := "inst-restore-" + uuid.NewString()[:8]
	log.Debug().Msgf("Building service %s...", serviceName)
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	serviceID := build.ServiceID
	environmentID := build.EnvironmentID
	require.NotEmpty(t, serviceID)
	require.NotEmpty(t, environmentID)

	// Create instance
	log.Debug().Msg("Creating instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--param", `{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}`,
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	originalInstanceID := instance.InstanceID
	require.NotEmpty(t, originalInstanceID)

	// Wait for RUNNING
	log.Debug().Msg("Waiting for instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, originalInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Trigger backup
	log.Debug().Msg("Triggering backup...")
	backupResult, err := dataaccess.TriggerResourceInstanceAutoBackup(ctx, token, serviceID, environmentID, originalInstanceID)
	require.NoError(t, err)
	snapshotID := backupResult.GetSnapshotId()
	require.NotEmpty(t, snapshotID)

	// Wait for snapshot to complete
	waitForSnapshotCompletion(t, ctx, token, serviceID, environmentID, originalInstanceID, snapshotID)

	// Delete the instance
	log.Debug().Msg("Deleting instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", originalInstanceID, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	waitForInstanceDeletion(t, ctx, originalInstanceID)

	// Restore using instance restore with --restore-to-source
	log.Debug().Msg("Restoring with instance restore --restore-to-source...")
	cmd.RootCmd.SetArgs([]string{"instance", "restore", originalInstanceID,
		"--snapshot-id", snapshotID,
		"--restore-to-source",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Wait for restored instance to reach RUNNING
	log.Debug().Msg("Waiting for restored instance to reach RUNNING...")
	err = testutils.WaitForInstanceToReachStatus(ctx, originalInstanceID, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// Cleanup
	log.Debug().Msg("Cleaning up...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", originalInstanceID, "--yes"})
	_ = cmd.RootCmd.ExecuteContext(ctx)
	waitForInstanceDeletion(t, ctx, originalInstanceID)

	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	_ = cmd.RootCmd.ExecuteContext(ctx)
}

// waitForInstanceDeletion waits until the instance is no longer found via describe.
func waitForInstanceDeletion(t *testing.T, ctx context.Context, instanceID string) {
	t.Helper()
	for i := 0; i < 30; i++ {
		cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID})
		err := cmd.RootCmd.ExecuteContext(ctx)
		if err != nil {
			return // Instance no longer found
		}
		time.Sleep(60 * time.Second)
	}
	t.Fatalf("instance %s was not deleted within the timeout", instanceID)
}

// waitForSnapshotCompletion polls snapshot status until it reaches "Complete".
func waitForSnapshotCompletion(t *testing.T, ctx context.Context, token, serviceID, environmentID, instanceID, snapshotID string) {
	t.Helper()
	for i := 0; i < 30; i++ {
		result, err := dataaccess.DescribeResourceInstanceSnapshot(ctx, token, serviceID, environmentID, instanceID, snapshotID)
		require.NoError(t, err)

		status := result.GetStatus()
		log.Debug().Msgf("Snapshot %s status: %s", snapshotID, status)
		if status == "Complete" {
			return
		}
		if status == "Failed" {
			t.Fatalf("snapshot %s failed", snapshotID)
		}
		time.Sleep(30 * time.Second)
	}
	t.Fatalf("snapshot %s did not complete within the timeout", snapshotID)
}
