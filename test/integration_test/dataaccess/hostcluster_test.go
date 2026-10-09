package dataaccess

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
)

func TestRestartHostClusterDeployment(t *testing.T) {
	testutils.IntegrationTest(t)
	hostClusterID := os.Getenv("TEST_RESTART_HOST_CLUSTER_ID")
	if hostClusterID == "" {
		t.Skip("set TEST_RESTART_HOST_CLUSTER_ID to a disposable host cluster; this test restarts its deployment")
	}
	email, password, err := testutils.GetTestAccount()
	require.NoError(t, err)
	loginResult, err := dataaccess.LoginWithPassword(t.Context(), email, password)
	require.NoError(t, err)
	token := loginResult.JWTToken
	require.NotEmpty(t, token)

	for _, tt := range []struct {
		name, token, id string
		wantErr         bool
		expectedErrMsg  string
	}{
		{name: "valid restart", token: token, id: hostClusterID},
		{name: "invalid token", token: "invalid-token", id: hostClusterID, wantErr: true},
		{name: "missing ID", token: token, wantErr: true, expectedErrMsg: "host cluster ID cannot be empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := dataaccess.RestartHostClusterDeployment(t.Context(), tt.token, tt.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					require.ErrorContains(t, err, tt.expectedErrMsg)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
