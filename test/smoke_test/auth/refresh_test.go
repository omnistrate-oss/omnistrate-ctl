package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_refresh(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	assert := assert.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	// Login first to get tokens
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Verify we have a token and refresh token stored
	token, err := config.GetToken()
	require.NoError(err)
	assert.NotEmpty(token, "should have access token after login")

	refreshToken, err := config.GetRefreshToken()
	require.NoError(err)
	assert.NotEmpty(refreshToken, "should have refresh token after login")

	// Refresh the token
	cmd.RootCmd.SetArgs([]string{"refresh"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Verify the new token is different and valid
	newToken, err := config.GetToken()
	require.NoError(err)
	assert.NotEmpty(newToken, "should have new access token after refresh")
	assert.NotEqual(token, newToken, "refreshed token should be different from original")

	// Verify the new refresh token was rotated
	newRefreshToken, err := config.GetRefreshToken()
	require.NoError(err)
	assert.NotEmpty(newRefreshToken, "should have new refresh token after refresh")
	assert.NotEqual(refreshToken, newRefreshToken, "refresh token should be rotated")

	// Verify the new token works by making an authenticated call
	cmd.RootCmd.SetArgs([]string{"logout"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Refresh without login should fail
	cmd.RootCmd.SetArgs([]string{"refresh"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	assert.Error(err, "refresh without login should fail")
}
