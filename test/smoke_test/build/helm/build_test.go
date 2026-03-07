package spec

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_build_basic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// Step 1: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 2: get spec files
	specFiles, err := os.ReadDir("../../specfiles/helm")
	require.NoError(err)

	if len(specFiles) == 0 {
		require.Fail("no spec files found")
	}

	// Step 3: test build service on all compose files
	for _, f := range specFiles {
		if f.IsDir() {
			continue
		}

		cmd.RootCmd.SetArgs([]string{"build",
			"--spec-type", "ServicePlanSpec",
			"-f", "../../specfiles/helm/" + f.Name(),
			"--name", f.Name() + uuid.NewString(),
			"--description", "My Service Description",
			"--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png",
			"--environment", "dev",
			"--environment-type", "dev",
			"--release-as-preferred",
			"--release-name", "v1.0.0-alpha",
		})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())

		cmd.RootCmd.SetArgs([]string{"service", "describe", "--id", build.ServiceID})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())

		cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())
	}
}

func Test_build_dry_run(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// Step 1: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 2: Create initial service with PostgreSQL configuration
	serviceName := "build-dry-run-helm-test" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{
		"build",
		"--spec-type", "ServicePlanSpec",
		"-f", "../../specfiles/helm/redis.yaml",
		"--name", serviceName,
		"--description", "Test redis helm service for dry run",
		"--environment", "dev",
		"--environment-type", "dev",
		"--release",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(build.ServiceID)

	// Store initial state for comparison
	initialServiceID := build.ServiceID
	cmd.RootCmd.SetArgs([]string{
		"service-plan",
		"describe",
		serviceName, "Redis Server",
		"--service-id", initialServiceID,
		"--output", "json",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	initialJsonOutput := utils.LastPrintedString

	// Step 3a: Test dry-run mode - Should not modify service
	cmd.RootCmd.SetArgs([]string{
		"build",
		"--spec-type", "ServicePlanSpec",
		"-f", "../../specfiles/helm/redis_dryrun.yaml",
		"--name", serviceName,
		"--description", "Test redis helm service for dry run",
		"--environment", "dev",
		"--environment-type", "dev",
		"--release",
		"--dry-run",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Verify dry-run didn't change the service configuration
	cmd.RootCmd.SetArgs([]string{
		"service-plan",
		"describe",
		serviceName, "Redis Server",
		"--service-id", initialServiceID,
		"--output", "json",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	dryRunOutput := utils.LastPrintedString
	require.Equal(initialJsonOutput, dryRunOutput, "Service configuration should not change after dry-run")

	// Step 3b: Apply the actual changes - Should modify service
	cmd.RootCmd.SetArgs([]string{
		"build",
		"--spec-type", "ServicePlanSpec",
		"-f", "../../specfiles/helm/redis_dryrun.yaml",
		"--name", serviceName,
		"--description", "Test redis helm service for dry run",
		"--environment", "dev",
		"--environment-type", "dev",
		"--release",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Verify the service configuration was actually modified
	cmd.RootCmd.SetArgs([]string{
		"service-plan",
		"describe",
		serviceName, "Redis Server",
		"--service-id", initialServiceID,
		"--output", "json",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEqual(initialJsonOutput, utils.LastPrintedString, "Service configuration should change after actual release")

	// Step 4: Cleanup - Delete the test service and associated resources
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
