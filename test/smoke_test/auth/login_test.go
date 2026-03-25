package auth

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_login(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	tests := []struct {
		Args           []string
		WantErr        bool
		ExpectedErrMsg string
	}{
		{[]string{"login", "--email=" + testEmail, "--password=" + testPassword}, false, ""},
		{[]string{"login", "--email=xzhang+cli@omnistrate.com"}, true, "must provide a non-empty password via --password or --password-stdin"},
		{[]string{"login", "--email=xzhang+cli@omnistrate.com", "--password=wrong_password"}, true, "wrong user email or password"},
	}

	for _, tt := range tests {
		cmd.RootCmd.SetArgs(tt.Args)
		err = cmd.RootCmd.ExecuteContext(ctx)
		if tt.WantErr {
			require.Error(err, tt.ExpectedErrMsg)
			require.Contains(err.Error(), tt.ExpectedErrMsg)
		} else {
			require.NoError(err)
		}
	}
}
