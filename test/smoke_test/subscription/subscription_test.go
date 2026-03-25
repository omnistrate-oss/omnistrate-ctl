package subscription

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_subscription_list(t *testing.T) {
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

	// Test subscription list with default (table) output
	cmd.RootCmd.SetArgs([]string{"subscription", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with default output should not error")

	// Test subscription list with table output explicitly
	cmd.RootCmd.SetArgs([]string{"subscription", "list", "--output", "table"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with table output should not error")

	// Test subscription list with json output
	cmd.RootCmd.SetArgs([]string{"subscription", "list", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with json output should not error")

	// Test subscription list with text output
	cmd.RootCmd.SetArgs([]string{"subscription", "list", "--output", "text"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with text output should not error")

	// Test subscription list with filters in table format
	cmd.RootCmd.SetArgs([]string{"subscription", "list", "--filter", "status:ACTIVE", "--output", "table"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with filter in table format should not error")

	// Test subscription list with filters in json format
	cmd.RootCmd.SetArgs([]string{"subscription", "list", "--filter", "status:ACTIVE", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err, "subscription list with filter in json format should not error")
}
