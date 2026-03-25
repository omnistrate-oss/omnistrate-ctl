package domain

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_domain_basic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	devDomainName := "dev" + uuid.NewString()
	devDomain := "domain" + uuid.NewString() + ".dev"
	prodDomainName := "prod" + uuid.NewString()
	prodDomain := "domain" + uuid.NewString() + ".prod"

	// create dev domain
	cmd.RootCmd.SetArgs([]string{"domain", "create", devDomainName, "--environment-type", "dev", "--domain", devDomain})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		require.Condition(func() bool {
			if strings.Contains(err.Error(), "saas portal does not exist for environment type") {
				return true
			}

			if strings.Contains(err.Error(), "domain with the same environment type already exists") {
				return true
			}

			return false
		})
	}

	// create prod domain
	cmd.RootCmd.SetArgs([]string{"domain", "create", prodDomainName, "--environment-type", "prod", "--domain", prodDomain})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		require.Condition(func() bool {
			if strings.Contains(err.Error(), "saas portal does not exist for environment type") {
				return true
			}

			if strings.Contains(err.Error(), "domain with the same environment type already exists") {
				return true
			}

			return false
		})
	}

	// PASS: list domains
	cmd.RootCmd.SetArgs([]string{"domain", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list domains by name
	cmd.RootCmd.SetArgs([]string{"domain", "list", "--filter", fmt.Sprintf("name:%s", devDomainName), "--filter", fmt.Sprintf("name:%s", prodDomainName)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// delete domains
	cmd.RootCmd.SetArgs([]string{"domain", "delete", devDomainName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		require.Contains(err.Error(), "domain not found")
	}

	// PASS: delete domain
	cmd.RootCmd.SetArgs([]string{"domain", "delete", prodDomainName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	if err != nil {
		require.Contains(err.Error(), "domain not found")
	}

}
