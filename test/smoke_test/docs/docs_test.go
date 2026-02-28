package docs

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_docs_compose_spec(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// PASS: list all compose-spec tags
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec"})
	err := cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list all compose-spec tags with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "networks"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "networks", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_docs_plan_spec(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// PASS: list all plan-spec tags
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec"})
	err := cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list all plan-spec tags with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "compute"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "compute", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
