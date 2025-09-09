package upgrade

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/upgrade/status"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/upgrade"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_upgrade_basic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: build service
	serviceName := "mysql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	serviceID := build.ServiceID
	productTierID := build.ProductTierID

	// PASS: create instance with param
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--param", `{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}`})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	instanceID := instance.InstanceID
	require.NotEmpty(t, instanceID)

	// PASS: wait for instance to reach running status
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: release mysql service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "release", "--service-id", serviceID, "--plan-id", productTierID, "--release-as-preferred", "--release-description", "v1.0.0-alpha"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: upgrade instance with latest version
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "latest"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)
	upgradeID := upgrade.UpgradePathIDs[0]

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID, "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", "detail", upgradeID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", "detail", upgradeID, "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: wait for instance to reach running status
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance to version 1.0
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "1.0"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance to preferred version
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)
	// PASS: scheduled upgrade
	err = validateScheduledAndCancel(ctx, instanceID, "1.0", false)
	require.NoError(err)
	err = validateScheduledAndCancel(ctx, instanceID, "1.0", true)
	require.NoError(err)

	// PASS: scheduled upgrade with max-concurrent-upgrades
	err = validateScheduledAndCancelWithMaxConcurrency(ctx, instanceID, "1.0", false, 10)
	require.NoError(err)
	// PASS: upgrade instance to version 1.0
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "1.0"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance to "v1.0.0-alpha"
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version-name", "v1.0.0-alpha"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance with max-concurrent-upgrades
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "1.0", "--max-concurrent-upgrades", "5"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance with max-concurrent-upgrades at minimum value
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "preferred", "--max-concurrent-upgrades", "1"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance with max-concurrent-upgrades at maximum value
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "1.0", "--max-concurrent-upgrades", "25"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for instance to reach running status
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: upgrade instance with max-concurrent-upgrades as 0 (should use system default)
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", "1.0", "--max-concurrent-upgrades", "0"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Len(upgrade.UpgradePathIDs, 1)

	// PASS: wait for final upgrade to complete
	time.Sleep(5 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning, 900*time.Second)
	require.NoError(err)

	// PASS: delete instance
	cmd.RootCmd.SetArgs([]string{"instance", "delete", instanceID, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Wait for the instances to be deleted
	for {
		cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID})
		err1 := cmd.RootCmd.ExecuteContext(ctx)

		if err1 != nil {
			break
		}

		time.Sleep(5 * time.Second)
	}

	// PASS: delete service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func validateScheduledAndCancel(ctx context.Context, instanceID string, targetVersion string, shouldSkipInstance bool) error {
	// Upgrade instance with latest version
	scheduledDate := time.Now().Add(3 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", targetVersion, "--scheduled-date", scheduledDate})
	err := cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}
	if len(upgrade.UpgradePathIDs) != 1 {
		return fmt.Errorf("expected 1 upgrade path ID, got %d", len(upgrade.UpgradePathIDs))
	}
	upgradeID := upgrade.UpgradePathIDs[0]

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
	if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
		return err
	}
	if status.LastUpgradeStatus.NotifyCustomer == true {
		return fmt.Errorf("expected notify customer to be false, got %v", status.LastUpgradeStatus.NotifyCustomer)
	}
	// Test notify-customer
	cmd.RootCmd.SetArgs([]string{"upgrade", "notify-customer", upgradeID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}

	for {
		cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
		if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
			return err
		}

		if status.LastUpgradeStatus.Status != model.InProgress.String() {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if status.LastUpgradeStatus.NotifyCustomer != true {
		return fmt.Errorf("expected notify customer to be true, got %v", status.LastUpgradeStatus.NotifyCustomer)
	}
	if status.LastUpgradeStatus.Status != model.Scheduled.String() {
		return fmt.Errorf("expected status %s, got %s", model.Scheduled.String(), status.LastUpgradeStatus.Status)
	}
	cmdArgs := []string{"upgrade", "cancel", upgradeID}
	if shouldSkipInstance {
		cmdArgs = []string{"upgrade", "skip-instances", upgradeID, "--resource-ids", instanceID}
	}
	cmd.RootCmd.SetArgs(cmdArgs)
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}

	for {
		cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
		if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
			return err
		}

		if status.LastUpgradeStatus.Status != model.Scheduled.String() {
			break
		}
		time.Sleep(5 * time.Second)
	}
	expectedStatus := model.Cancelled.String()
	if shouldSkipInstance {
		expectedStatus = model.Complete.String()
	}
	if status.LastUpgradeStatus.Status != expectedStatus {
		return fmt.Errorf("expected status %s, got %s", expectedStatus, status.LastUpgradeStatus.Status)
	}
	return nil
}

func validateScheduledAndCancelWithMaxConcurrency(ctx context.Context, instanceID string, targetVersion string, shouldSkipInstance bool, maxConcurrentUpgrades int) error {
	// Upgrade instance with scheduled date and max-concurrent-upgrades
	scheduledDate := time.Now().Add(3 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
	cmd.RootCmd.SetArgs([]string{"upgrade", instanceID, "--version", targetVersion, "--scheduled-date", scheduledDate, "--max-concurrent-upgrades", fmt.Sprintf("%d", maxConcurrentUpgrades)})
	err := cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}
	if len(upgrade.UpgradePathIDs) != 1 {
		return fmt.Errorf("expected 1 upgrade path ID, got %d", len(upgrade.UpgradePathIDs))
	}
	upgradeID := upgrade.UpgradePathIDs[0]

	cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
	if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
		return err
	}
	if status.LastUpgradeStatus.NotifyCustomer == true {
		return fmt.Errorf("expected notify customer to be false, got %v", status.LastUpgradeStatus.NotifyCustomer)
	}

	for {
		cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
		if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
			return err
		}

		if status.LastUpgradeStatus.Status != model.InProgress.String() {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if status.LastUpgradeStatus.Status != model.Scheduled.String() {
		return fmt.Errorf("expected status %s, got %s", model.Scheduled.String(), status.LastUpgradeStatus.Status)
	}

	// Cancel the scheduled upgrade
	cmd.RootCmd.SetArgs([]string{"upgrade", "cancel", upgradeID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}

	for {
		cmd.RootCmd.SetArgs([]string{"upgrade", "status", upgradeID})
		if err = cmd.RootCmd.ExecuteContext(ctx); err != nil {
			return err
		}

		if status.LastUpgradeStatus.Status != model.Scheduled.String() {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if status.LastUpgradeStatus.Status != model.Cancelled.String() {
		return fmt.Errorf("expected status %s, got %s", model.Cancelled.String(), status.LastUpgradeStatus.Status)
	}
	return nil
}
