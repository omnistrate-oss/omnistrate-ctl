package dataaccess

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken(t *testing.T) {
	testutils.IntegrationTest(t)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	ctx := context.TODO()

	// Login to get a valid refresh token
	loginResult, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, loginResult.JWTToken)
	require.NotEmpty(t, loginResult.RefreshToken)

	// Refresh with a valid token should succeed
	refreshResult, err := dataaccess.RefreshToken(ctx, loginResult.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshResult.JWTToken, "should get a new JWT")
	assert.NotEqual(t, loginResult.JWTToken, refreshResult.JWTToken, "new JWT should differ from original")
	assert.NotEmpty(t, refreshResult.RefreshToken, "should get a rotated refresh token")

	// Using the same (now-consumed) refresh token again should fail (single-use)
	_, err = dataaccess.RefreshToken(ctx, loginResult.RefreshToken)
	assert.Error(t, err, "reusing a consumed refresh token should fail")

	// Using a completely invalid refresh token should fail
	_, err = dataaccess.RefreshToken(ctx, "invalid-token-value")
	assert.Error(t, err, "invalid refresh token should fail")
}
