package secret

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_secret_basic(t *testing.T) {
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

	// PASS: set db password secret
	cmd.RootCmd.SetArgs([]string{"secret", "set", "dev", "dbPassword", "password"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: get db password secret
	cmd.RootCmd.SetArgs([]string{"secret", "get", "dev", "dbPassword"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list secrets
	cmd.RootCmd.SetArgs([]string{"secret", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete the secret
	cmd.RootCmd.SetArgs([]string{"secret", "delete", "dev", "dbPassword"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
