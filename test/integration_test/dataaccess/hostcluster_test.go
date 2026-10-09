package dataaccess

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestRestartHostClusterDeploymentNonexistentHostCluster(t *testing.T) {
	testutils.IntegrationTest(t)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	ctx := context.Background()
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, login.JWTToken)

	err = dataaccess.RestartHostClusterDeployment(ctx, login.JWTToken, "hc-123456789")
	require.Error(t, err)
	// The backend may check access before looking up the nonexistent cluster.
	require.Regexp(t, `^(not_found|bad_request|invalid_format|invalid_length|forbidden)\nDetail: .+`, err.Error())
	t.Logf("Restarting nonexistent host cluster returned: %v", err)
}
