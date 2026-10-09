package deploymentcell

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestDeploymentCellRestartNonexistentHostCluster(t *testing.T) {
	testutils.SmokeTest(t)
	defer testutils.Cleanup()

	ctx := context.Background()
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	for _, output := range []string{"table", "json"} {
		t.Run(output, func(t *testing.T) {
			cmd.RootCmd.SetArgs([]string{"deployment-cell", "restart", "hc-123456789", "--output", output})
			err := cmd.RootCmd.ExecuteContext(ctx)
			require.Error(t, err)
			require.Regexp(t, `^failed to restart deployment cell hc-123456789: (not_found|bad_request|invalid_format|invalid_length|forbidden)\nDetail: .+`, err.Error())
		})
	}
}
