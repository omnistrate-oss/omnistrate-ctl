package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func TestLogout(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// FAIL: logout without login
	cmd.RootCmd.SetArgs([]string{"logout"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: logout after login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{"logout"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
