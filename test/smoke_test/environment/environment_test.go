package environment

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/environment"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_environment_basic(t *testing.T) {
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

	// PASS: create postgresql service
	serviceName := "postgresql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/postgresql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	serviceID := build.ServiceID
	sourceEnvID := build.EnvironmentID

	// PASS: create an environment
	envName := "prod"
	cmd.RootCmd.SetArgs([]string{"environment", "create", serviceName, envName, "--type=PROD", "--source=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	envName2 := "qa"
	cmd.RootCmd.SetArgs([]string{"environment", "create", envName2, "--type=QA", "--service-id=" + serviceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	env2ID := environment.EnvironmentID

	// PASS: describe the environment
	cmd.RootCmd.SetArgs([]string{"environment", "describe", serviceName, envName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"environment", "describe", "--service-id", serviceID, "--environment-id", env2ID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list environments
	cmd.RootCmd.SetArgs([]string{"environment", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list environments with filter
	cmd.RootCmd.SetArgs([]string{"environment", "list", fmt.Sprintf("-f=service_name:%s", serviceName)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: promote the dev environment
	cmd.RootCmd.SetArgs([]string{"environment", "promote", serviceName, "dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"environment", "promote", "--service-id", serviceID, "--environment-id", sourceEnvID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: describe the environment after promotion
	cmd.RootCmd.SetArgs([]string{"environment", "describe", serviceName, envName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"environment", "describe", "--service-id", serviceID, "--environment-id", env2ID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete the environment
	cmd.RootCmd.SetArgs([]string{"environment", "delete", serviceName, envName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"environment", "delete", "--service-id", serviceID, "--environment-id", env2ID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete postgresql service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
