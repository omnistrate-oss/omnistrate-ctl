package spec

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_build_basic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	require := require.New(t)
	defer testutils.Cleanup()

	var err error

	// Step 1: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Step 2: get spec files
	specFiles, err := os.ReadDir("../../specfiles/kustomize")
	require.NoError(err)

	if len(specFiles) == 0 {
		require.Fail("no spec files found")
	}

	// Step 3: test build service on all compose files
	for _, f := range specFiles {
		if f.IsDir() {
			continue
		}

		cmd.RootCmd.SetArgs([]string{"build",
			"--spec-type", "ServicePlanSpec",
			"-f", "../../specfiles/kustomize/" + f.Name(),
			"--name", f.Name() + uuid.NewString(),
			"--description", "My Service Description",
			"--service-logo-url", "https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png",
			"--environment", "dev",
			"--environment-type", "dev",
			"--release-as-preferred",
			"--release-name", "v1.0.0-alpha",
		})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())

		cmd.RootCmd.SetArgs([]string{"service", "describe", "--id", build.ServiceID})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())

		cmd.RootCmd.SetArgs([]string{"service-plan", "describe", "--service-id", build.ServiceID, "--plan-id", build.ProductTierID})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())

		cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", build.ServiceID})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, f.Name())
	}
}
