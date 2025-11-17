package compose

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

	// Step 2: get compose files
	composeFiles, err := os.ReadDir("../../composefiles")
	require.NoError(err)

	if len(composeFiles) == 0 {
		t.Error("No compose files found")
	}

	// Step 3: test build service on all compose files
	for _, f := range composeFiles {
		if f.IsDir() {
			continue
		}

		if f.Name() == ".env" {
			continue
		}

		cmd.RootCmd.SetArgs([]string{"build",
			"-f", "../../composefiles/" + f.Name(),
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

func Test_build_update_service(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// PASS: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: create mysql cluster service
	serviceName := "mysql cluster" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--name", serviceName, "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png", "--release"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "describe", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_variation_apiparam_image_infra_capability.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update back to original mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_variation_account_integration_resource.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update back to original mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: create postgres cluster service
	serviceName = "postgres cluster" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/postgrescluster_original.yaml", "--name", serviceName, "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png", "--release"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "describe", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update postgres cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/postgrescluster_variation_load_balancer.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: update back to original postgres cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/postgrescluster_original.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: add new service plan to postgres cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/postgrescluster_variation_new_tier.yaml", "--name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_build_output_format(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// PASS: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: json output
	serviceName := "mysql cluster" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--product-name", serviceName, "--release", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: table output
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--product-name", serviceName, "--release", "--output", "table"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: text output
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--product-name", serviceName, "--release", "--output", "text"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: remove service
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_build_duplicate_service_plan_name(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// PASS: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	serviceName := "mysql cluster" + uuid.NewString()
	// PASS: create mysql cluster service in dev environment
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--product-name", serviceName, "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png", "--release"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	devEnvironmentID := build.EnvironmentID
	devProductTierID := build.ProductTierID
	require.NotEmpty(devEnvironmentID)
	require.NotEmpty(devProductTierID)

	// PASS: create mysql cluster service in prod environment
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_original.yaml", "--product-name", serviceName, "--environment", "prod", "--environment-type", "prod", "--release"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	prodEnvironmentID := build.EnvironmentID
	prodProductTierID := build.ProductTierID
	require.NotEmpty(prodEnvironmentID)
	require.NotEmpty(prodEnvironmentID)
	require.NotEqual(devEnvironmentID, prodEnvironmentID)
	require.NotEqual(devProductTierID, prodProductTierID)

	// PASS: update dev mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_variation_apiparam_image_infra_capability.yaml", "--product-name", serviceName, "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Equal(devEnvironmentID, build.EnvironmentID)
	require.Equal(devProductTierID, build.ProductTierID)

	// PASS: update prod mysql cluster service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/variations/mysqlcluster_variation_apiparam_image_infra_capability.yaml", "--product-name", serviceName, "--environment", "prod", "--environment-type", "prod", "--release-as-preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.Equal(prodEnvironmentID, build.EnvironmentID)
	require.Equal(prodProductTierID, build.ProductTierID)

	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_build_invalid_file(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"build", "-f", "invalid_file.yaml", "--product-name", "My Service" + uuid.NewString(), "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "does not exist")
}

func Test_build_no_file(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"build", "--product-name", "My Service" + uuid.NewString(), "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "no omnistrate-compose.yaml or spec.yaml found")
}

func Test_build_create_no_name(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/postgresql.yaml", "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "name is required")
}

func Test_build_create_no_description(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/postgresql.yaml", "--product-name", "postgresql" + uuid.NewString(), "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_build_create_no_service_logo_url(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"build", "-f", "../../composefiles/postgresql.yaml", "--product-name", "postgresql" + uuid.NewString(), "--description", "My Service Description"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
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
	serviceName := "build-dry-run-test" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{
		"build",
		"-f", "../../composefiles/postgresql.yaml",
		"--product-name", serviceName,
		"--description", "Test pgres service for dry run",
		"--environment", "dev",
		"--environment-type", "dev",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(build.ServiceID)

	// Store initial state for comparison
	initialServiceID := build.ServiceID
	cmd.RootCmd.SetArgs([]string{
		"service-plan",
		"describe",
		serviceName, "postgresql",
		"--service-id", initialServiceID,
		"--output", "json",
	})
	initialJsonOutput := utils.LastPrintedString

	// Step 3a: Test dry-run mode - Should not modify service
	cmd.RootCmd.SetArgs([]string{
		"build",
		"-f", "../../composefiles/postgresql_dryrun.yaml",
		"--product-name", serviceName,
		"--environment", "dev",
		"--environment-type", "dev",
		"--dry-run",
		"--release",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Verify dry-run didn't change the service configuration
	cmd.RootCmd.SetArgs([]string{
		"service-plan",
		"describe",
		serviceName, "postgresql",
		"--service-id", initialServiceID,
		"--output", "json",
	})
	require.Equal(initialJsonOutput, utils.LastPrintedString, "Service configuration should not change after dry-run")

	// Step 3b: Apply the actual changes - Should modify service
	cmd.RootCmd.SetArgs([]string{
		"build",
		"-f", "../../composefiles/postgresql_dryrun.yaml",
		"--product-name", serviceName,
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
		serviceName, "postgresql",
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
