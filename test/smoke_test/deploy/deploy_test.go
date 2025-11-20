package deploy

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/deploy"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Test_deploy_with_instance tests deployment with instance creation
func Test_deploy_with_instance(t *testing.T) {
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

	// Step 2: Deploy mysql with instance
	serviceName := "mysql-deploy-test-" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"deploy",
		"-f", "../composefiles/mysql.yaml",
		"--product-name", serviceName,
		"--environment", "prod",
		"--environment-type", "prod",
		"--deployment-type", "hosted",
		"--cloud-provider", "aws",
		"--region", "ap-south-1",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(deploy.ServiceID)
	require.NotEmpty(deploy.InstanceID)

	// Step 3: Verify instance was created
	cmd.RootCmd.SetArgs([]string{"instance", "describe", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 4: Delete instance
	cmd.RootCmd.SetArgs([]string{"instance", "delete", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 5: Create instance again with same service
	cmd.RootCmd.SetArgs([]string{"deploy",
		"-f", "../composefiles/mysql.yaml",
		"--product-name", serviceName,
		"--environment", "prod",
		"--environment-type", "prod",
		"--deployment-type", "hosted",
		"--cloud-provider", "aws",
		"--region", "ap-south-1",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(deploy.InstanceID)

	// Step 6: Verify new instance was created
	cmd.RootCmd.SetArgs([]string{"instance", "describe", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 7: Cleanup - delete instance
	cmd.RootCmd.SetArgs([]string{"instance", "delete", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 7: Cleanup - delete  service
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", deploy.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}



// Test_deploy_dry_run tests dry-run mode
// func Test_deploy_dry_run(t *testing.T) {
// 	testutils.SmokeTest(t)

// 	ctx := context.TODO()

// 	require := require.New(t)
// 	defer testutils.Cleanup()

// 	var err error

// 	// Step 1: login
// 	testEmail, testPassword, err := testutils.GetTestAccount()
// 	require.NoError(err)
// 	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
// 	err = cmd.RootCmd.ExecuteContext(ctx)
// 	require.NoError(err)

// 	// Step 2: Test dry-run mode - should validate without deploying
// 	serviceName := "deploy-dry-run-test-" + uuid.NewString()
// 	cmd.RootCmd.SetArgs([]string{"deploy",
// 		"-f", "../composefiles/mysql.yaml",
// 		"--product-name", serviceName,
// 		"--environment", "prod",
// 		"--environment-type", "prod",
// 		"--deployment-type", "hosted",
// 		"--dry-run",
// 		"--skip-docker-build",
// 	})
// 	err = cmd.RootCmd.ExecuteContext(ctx)
// 	require.NoError(err)

// 	// Step 3: Verify no service was created in dry-run mode
// 	// In dry-run mode, ServiceID should be empty as nothing is actually created
// 	// The command should succeed but not create any resources
// }

// Test_deploy_invalid_file tests deployment with invalid file
func Test_deploy_invalid_file(t *testing.T) {
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

	cmd.RootCmd.SetArgs([]string{"deploy",
		"--file", "invalid_file.yaml",
		"--product-name", "test-service-" + uuid.NewString(),
		"--deployment-type", "hosted",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "does not exist")
}

// Test_deploy_no_file tests deployment without specifying a file
func Test_deploy_no_file(t *testing.T) {
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

	// Change to temp directory without any compose files
	tempDir, err := os.MkdirTemp("", "deploy_test")
	require.NoError(err)
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	require.NoError(err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"deploy",
		"--product-name", "test-service-" + uuid.NewString(),
		"--deployment-type", "hosted",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "no omnistrate-compose.yaml or spec.yaml found")
}

// Test_deploy_no_name tests deployment without service name
func Test_deploy_no_name(t *testing.T) {
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

	cmd.RootCmd.SetArgs([]string{"deploy",
		"--file", "../../composefiles/mysql.yaml",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "name is required")
}

// Test_deploy_output_format tests different output formats
func Test_deploy_output_format(t *testing.T) {
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

	serviceName := "output-format-test-" + uuid.NewString()

	// Test JSON output - create service and instance
	cmd.RootCmd.SetArgs([]string{"deploy",
		"-f", "../composefiles/mysql.yaml",
		"--product-name", serviceName,
		"--environment", "prod",
		"--environment-type", "prod",
		"--deployment-type", "hosted",
		"--cloud-provider", "aws",
		"--region", "ap-south-1",
		"--output", "json",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(deploy.ServiceID)
	require.NotEmpty(deploy.InstanceID)

	// Verify new instance was created
	cmd.RootCmd.SetArgs([]string{"instance", "describe", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Cleanup - delete instance
	cmd.RootCmd.SetArgs([]string{"instance", "delete", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	
	// Cleanup - delete  service
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", deploy.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}


// Test_deploy_with_parameters tests deployment with custom parameters
func Test_deploy_with_parameters(t *testing.T) {
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

	serviceName := "param-test-" + uuid.NewString()

	// Deploy with parameters - test with valid mysql parameters
	cmd.RootCmd.SetArgs([]string{"deploy",
		"-f", "../composefiles/mysql.yaml",
		"--product-name", serviceName,
		"--deployment-type", "hosted",
		"--param", `{"rootPassword":"test_root_pass","password":"test_user_pass","username":"testuser","databaseName":"testdb"}`,
		"--cloud-provider", "gcp",
		"--region", "us-central1",
		"--skip-docker-build",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	require.NotEmpty(deploy.ServiceID)
	require.NotEmpty(deploy.InstanceID)

	// Verify new instance was created
	cmd.RootCmd.SetArgs([]string{"instance", "describe", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Cleanup - delete instance
	cmd.RootCmd.SetArgs([]string{"instance", "delete", "--id", deploy.InstanceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	
	// Cleanup - delete  service
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", deploy.ServiceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
