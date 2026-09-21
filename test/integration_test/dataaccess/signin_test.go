package dataaccess

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignIn(t *testing.T) {
	testutils.IntegrationTest(t)

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)

	tests := []struct {
		name             string
		email            string
		password         string
		wantErr          bool
		expectedErrParts []string
	}{
		{
			"valid login",
			testEmail,
			testPassword,
			false,
			nil,
		},
		{
			"missing email",
			"",
			"",
			true,
			[]string{"invalid_format", "body.email must be formatted as a email", "length of body.email", "length of body.password"},
		},
		{
			"missing password",
			"xzhang+cli1@omnistrate.com",
			"",
			true,
			[]string{"invalid_length", "length of body.password"},
		},
		{
			"invalid password",
			"--email=xzhang+cli@omnistrate.com",
			"wrong_password",
			true,
			[]string{"bad_request", "Invalid request: wrong user email or password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			ctx := context.TODO()
			result, err := dataaccess.LoginWithPassword(ctx, tt.email, tt.password)

			if tt.wantErr {
				require.Error(err)
				for _, expectedErrPart := range tt.expectedErrParts {
					assert.Contains(err.Error(), expectedErrPart)
				}
				assert.Empty(result.JWTToken)
			} else {
				require.NoError(err)
				assert.NotEmpty(result.JWTToken)
			}
		})
	}
}
