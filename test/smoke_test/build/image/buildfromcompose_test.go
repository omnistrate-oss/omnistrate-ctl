package image

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_build_service_from_image(t *testing.T) {
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

	serviceName := "mysql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--image", "docker.io/mysql:latest", "--product-name", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	serviceName2 := "mysql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--image", "docker.io/mysql:latest", "--product-name", serviceName2, "--env-var", "MYSQL_ROOT_PASSWORD=secret", "--env-var", "MYSQL_DATABASE=mydb"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName2})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	serviceName3 := "mysql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--image", "docker.io/mysql:latest", "--product-name", serviceName3, "--env-var", "MYSQL_ROOT_PASSWORD=secret", "--env-var", "MYSQL_DATABASE=mydb", "--image-registry-auth-username", "test", "--image-registry-auth-password", "test"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
	require.Contains(err.Error(), "cannot read image")
}
