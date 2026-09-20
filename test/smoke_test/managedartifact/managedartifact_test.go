package managedartifact

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_managed_artifact_read_only(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{
		"login",
		fmt.Sprintf("--email=%s", testEmail),
		fmt.Sprintf("--password=%s", testPassword),
	})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	tests := []struct {
		name string
		args []string
	}{
		{name: "list releases as table", args: []string{"managed-artifact", "release", "list", "--limit", "1"}},
		{name: "list releases as JSON", args: []string{"managed-artifact", "release", "list", "--limit", "1", "--output", "json"}},
		{name: "list syncs as JSON", args: []string{"managed-artifact", "sync", "list", "--limit", "1", "--output", "json"}},
		{name: "describe development AWS policy", args: []string{"managed-artifact", "policy", "describe", "--environment-type", "dev", "--cloud-provider", "aws"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd.RootCmd.SetArgs(test.args)
			require.NoError(t, cmd.RootCmd.ExecuteContext(ctx), "command failed: %v", test.args)
		})
	}
}
