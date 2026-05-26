package instance

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestInstanceCustomWorkflowCommandHelp(t *testing.T) {
	testutils.SmokeTest(t)
	t.Cleanup(func() {
		cmd.RootCmd.SetOut(os.Stdout)
		cmd.RootCmd.SetErr(os.Stderr)
	})

	for _, args := range [][]string{
		{"instance", "custom-workflow", "--help"},
		{"instance", "custom-workflow", "list", "--help"},
		{"instance", "custom-workflow", "describe", "--help"},
		{"instance", "custom-workflow", "trigger", "--help"},
	} {
		for _, output := range []string{"text", "table", "json"} {
			t.Run(fmt.Sprintf("%s/%s", fmt.Sprint(args), output), func(t *testing.T) {
				testArgs := append([]string{}, args...)
				testArgs = append(testArgs, "--output", output)

				cmd.RootCmd.SetOut(io.Discard)
				cmd.RootCmd.SetErr(io.Discard)
				cmd.RootCmd.SetArgs(testArgs)
				require.NoError(t, cmd.RootCmd.ExecuteContext(context.TODO()))
			})
		}
	}
}

func TestInstanceCustomWorkflowCommands(t *testing.T) {
	testutils.SmokeTest(t)

	instanceID := os.Getenv("CUSTOM_WORKFLOW_TEST_INSTANCE_ID")
	selector := os.Getenv("CUSTOM_WORKFLOW_TEST_SELECTOR")
	if instanceID == "" || selector == "" {
		t.Skip("set CUSTOM_WORKFLOW_TEST_INSTANCE_ID and CUSTOM_WORKFLOW_TEST_SELECTOR to run custom workflow smoke test")
	}

	ctx := context.TODO()
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "custom-workflow", "list", instanceID, "--output", "json"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	cmd.RootCmd.SetArgs([]string{"instance", "custom-workflow", "describe", instanceID, selector, "--output", "json"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	args := []string{"instance", "custom-workflow", "trigger", instanceID, selector, "--output", "json"}
	if params := os.Getenv("CUSTOM_WORKFLOW_TEST_PARAMS"); params != "" {
		args = append(args, "--param", params)
	}
	cmd.RootCmd.SetArgs(args)
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))
}
