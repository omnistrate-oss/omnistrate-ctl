package helm

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestHelmBasic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// Step 1: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Step 2: save helm chart
	cmd.RootCmd.SetArgs([]string{"helm", "save", "redis-smoke-test", "--version=20.0.1", "--namespace=default", "--repo-name=redis", "--repo-url=https://charts.bitnami.com/bitnami", "--values-file=./values"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Step 3: list helm charts
	cmd.RootCmd.SetArgs([]string{"helm", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Step 4: describe helm chart
	cmd.RootCmd.SetArgs([]string{"helm", "describe", "redis-smoke-test", "--version=20.0.1"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Step 5: list helm chart installations
	cmd.RootCmd.SetArgs([]string{"helm", "list-installations"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Step 6: delete helm chart
	cmd.RootCmd.SetArgs([]string{"helm", "delete", "redis-smoke-test", "--version=20.0.1"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}
