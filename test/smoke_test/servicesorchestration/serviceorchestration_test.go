package servicesorchestration

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestInstanceBasic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// PASS: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: list services orchestrations with default
	cmd.RootCmd.SetArgs([]string{"services-orchestration", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: list services orchestrations with environment type lowercase
	cmd.RootCmd.SetArgs([]string{"services-orchestration", "list", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: list services orchestrations with environment type uppercase
	cmd.RootCmd.SetArgs([]string{"services-orchestration", "list", "--environment-type=DEV"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}
