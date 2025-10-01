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
		name           string
		email          string
		password       string
		wantErr        bool
		expectedErrMsg string
	}{
		{
			"valid login",
			testEmail,
			testPassword,
			false,
			"",
		},
		{
			"missing email",
			"",
			"",
			true,
			"invalid_format\nDetail: body.email must be formatted as a email but got value \"\", mail: no address; length of body.email must be greater or equal than 1 but got value \"\" (len=0); length of body.password must be greater or equal than 1 but got value \"\" (len=0)",
		},
		{
			"missing password",
			"xzhang+cli1@omnistrate.com",
			"",
			true,
			"invalid_length\nDetail: length of body.password must be greater or equal than 1 but got value \"\" (len=0)",
		},
		{
			"invalid password",
			"--email=xzhang+cli@omnistrate.com",
			"wrong_password",
			true,
			"bad_request\nDetail: Invalid request: wrong user email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			ctx := context.TODO()
			token, err := dataaccess.LoginWithPassword(ctx, tt.email, tt.password)

			if tt.wantErr {
				assert.Equal(tt.expectedErrMsg, err.Error())
				assert.Empty(token)
			} else {
				require.NoError(err)
				assert.NotEmpty(token)
			}
		})
	}
}
