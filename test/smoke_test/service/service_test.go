package service

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

func Test_service_basic(t *testing.T) {
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

	serviceName := "postgresql" + uuid.NewString()

	// Build service
	cmd.RootCmd.SetArgs([]string{"build", "-f", "../composefiles/postgresql.yaml", "--name", serviceName, "--description", "My Service Description", "--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	serviceID := build.ServiceID

	// List services
	cmd.RootCmd.SetArgs([]string{"service", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// List services by name
	cmd.RootCmd.SetArgs([]string{"service", "list", "--filter", fmt.Sprintf("name:%s", serviceName)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Get services by ID
	cmd.RootCmd.SetArgs([]string{"service", "list", "--filter", fmt.Sprintf("id:%s", serviceID)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Describe services by name
	cmd.RootCmd.SetArgs([]string{"service", "describe", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Describe services by ID
	cmd.RootCmd.SetArgs([]string{"service", "describe", "--id", serviceID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Delete service by name
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Delete service by ID
	cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", "s-invalid"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "service not found")
}
