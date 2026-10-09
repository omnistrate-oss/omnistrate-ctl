package deploymentcell

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
)

func TestDeploymentCellRestart(t *testing.T) {
	testutils.SmokeTest(t)
	// Use separate disposable cells so the second restart cannot conflict with
	// the first in-progress restart. Never select a live cell automatically.
	hostClusterID := os.Getenv("TEST_RESTART_HOST_CLUSTER_ID")
	jsonHostClusterID := os.Getenv("TEST_RESTART_HOST_CLUSTER_ID_JSON")
	if hostClusterID == "" || jsonHostClusterID == "" {
		t.Skip("set TEST_RESTART_HOST_CLUSTER_ID and TEST_RESTART_HOST_CLUSTER_ID_JSON to separate disposable cells; this test restarts their deployments")
	}
	require.NotEqual(t, hostClusterID, jsonHostClusterID, "restart smoke tests require separate cells")
	defer testutils.Cleanup()
	email, password, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", email), fmt.Sprintf("--password=%s", password)})
	require.NoError(t, cmd.RootCmd.ExecuteContext(t.Context()))

	for _, tt := range []struct {
		name, id string
		flags    []string
	}{
		{name: "default output", id: hostClusterID},
		{name: "JSON output", id: jsonHostClusterID, flags: []string{"--output", "json"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"deployment-cell", "restart", tt.id}, tt.flags...)
			cmd.RootCmd.SetArgs(args)
			require.NoError(t, cmd.RootCmd.ExecuteContext(t.Context()))
			if len(tt.flags) > 0 {
				var result model.DeploymentCellRestartResult
				require.NoError(t, json.Unmarshal([]byte(utils.LastPrintedString), &result))
				require.Equal(t, tt.id, result.ID)
				require.Equal(t, "Deployment restart requested successfully", result.Message)
			}
		})
	}
}
