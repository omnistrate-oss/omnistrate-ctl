package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestInstanceCreateDescribeAndList(t *testing.T) {
	ctx, serviceName := setupInstanceSmokeTest(t, "mysql-create")

	instanceID := createMySQLInstance(t, ctx, serviceName,
		"--tags", "environment=dev,owner=platform",
		"--param", newMySQLParamsJSON(t),
	)

	cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "list"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "-f", "environment:DEV,cloud_provider:gcp", "-f", "environment:Dev,cloud_provider:aws"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "-f", fmt.Sprintf("service:%s", serviceName)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
}

func TestInstanceModifyTagsAndFilters(t *testing.T) {
	ctx, serviceName := setupInstanceSmokeTest(t, "mysql-tags")

	instanceID := createMySQLInstance(t, ctx, serviceName,
		"--tags", "environment=dev,owner=platform",
		"--param", newMySQLParamsJSON(t),
	)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "modify", instanceID, "--tags", "environment=prod,owner=platform"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	time.Sleep(60 * time.Second)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "environment=prod"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "environment=prod", "--tag", "owner=platform"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "-f", fmt.Sprintf("service:%s", serviceName), "--tag", "environment=prod"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
}

func TestInstanceCreateFromParamFileAndUpdate(t *testing.T) {
	ctx, serviceName := setupInstanceSmokeTest(t, "mysql-param-file")

	instanceID := createMySQLInstance(t, ctx, serviceName,
		"--tags", "source=file",
		"--param-file", writeMySQLParamsFile(t),
	)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "update", instanceID, "--param-file", writeMySQLParamsFile(t)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	time.Sleep(60 * time.Second)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "source=file"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
}

func TestInstanceStopStartRestart(t *testing.T) {
	ctx, serviceName := setupInstanceSmokeTest(t, "mysql-power")

	instanceID := createMySQLInstance(t, ctx, serviceName, "--param", newMySQLParamsJSON(t))
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "stop", instanceID})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusStopped))

	cmd.RootCmd.SetArgs([]string{"instance", "start", instanceID})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "restart", instanceID})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	time.Sleep(60 * time.Second)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))
}

func TestInstanceUpdateParameters(t *testing.T) {
	ctx, serviceName := setupInstanceSmokeTest(t, "mysql-update")

	instanceID := createMySQLInstance(t, ctx, serviceName, "--param", newMySQLParamsJSON(t))
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))

	cmd.RootCmd.SetArgs([]string{"instance", "update", instanceID, "--param", newMySQLParamsJSON(t)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	time.Sleep(60 * time.Second)
	require.NoError(t, testutils.WaitForInstanceToReachStatus(ctx, instanceID, instance.InstanceStatusRunning))
}

func setupInstanceSmokeTest(t *testing.T, servicePrefix string) (context.Context, string) {
	t.Helper()
	testutils.SmokeTest(t)
	resetGlobals()

	ctx := context.TODO()
	t.Cleanup(testutils.Cleanup)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	serviceName := fmt.Sprintf("%s-%s", servicePrefix, uuid.NewString()[:8])
	cmd.RootCmd.SetArgs([]string{"build", "--file", writeRenderedMySQLComposeFile(t), "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	t.Cleanup(func() {
		cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
		if err := cmd.RootCmd.ExecuteContext(ctx); err != nil && !isNotFoundError(err) {
			t.Errorf("cleanup service %s: %v", serviceName, err)
		}
	})

	return ctx, serviceName
}

func createMySQLInstance(t *testing.T, ctx context.Context, serviceName string, extraArgs ...string) string {
	t.Helper()
	instance.InstanceID = ""

	args := []string{
		"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--network-type=INTERNAL",
	}
	args = append(args, extraArgs...)

	createInstanceWithRetry(t, ctx, args)
	instanceID := instance.InstanceID
	require.NotEmpty(t, instanceID)

	t.Cleanup(func() {
		deleteInstanceIfPresent(t, ctx, instanceID)
	})

	return instanceID
}

func newMySQLParamsJSON(t *testing.T) string {
	t.Helper()

	params, err := json.Marshal(map[string]string{
		"databaseName": "default",
		"password":     testutils.RandomTestPassword(),
		"rootPassword": testutils.RandomTestPassword(),
		"username":     "user",
	})
	require.NoError(t, err)
	return string(params)
}

func writeMySQLParamsFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mysql-params.json")
	require.NoError(t, os.WriteFile(path, []byte(newMySQLParamsJSON(t)), 0o600))
	return path
}

func writeRenderedMySQLComposeFile(t *testing.T) string {
	t.Helper()

	content, err := os.ReadFile("../composefiles/mysql.yaml")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "mysql.yaml")
	rendered := testutils.RenderTestCredentials(string(content))
	require.NoError(t, os.WriteFile(path, []byte(rendered), 0o600)) //nolint:gosec // destination is confined to t.TempDir
	return path
}

func deleteInstanceIfPresent(t *testing.T, ctx context.Context, instanceID string) {
	t.Helper()
	if instanceID == "" {
		return
	}

	cmd.RootCmd.SetArgs([]string{"instance", "delete", instanceID, "--yes"})
	if err := cmd.RootCmd.ExecuteContext(ctx); err != nil {
		if isNotFoundError(err) {
			return
		}
		t.Errorf("cleanup instance %s: %v", instanceID, err)
		return
	}

	waitForInstanceDeletion(t, ctx, instanceID)
}

func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
