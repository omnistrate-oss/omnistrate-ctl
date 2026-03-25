package serviceplan

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

func Test_service_plan_basic(t *testing.T) {
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

	// PASS: create postgresql service
	serviceName := "postgresql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/postgresql.yaml", "--name", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: release postgresql service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "release", serviceName, "postgresql"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: set it as preferred
	cmd.RootCmd.SetArgs([]string{"service-plan", "set-default", serviceName, "postgresql", "--version=latest"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list service plans
	cmd.RootCmd.SetArgs([]string{"service-plan", "list", "--filter", fmt.Sprintf("service_name:%s", serviceName), "--filter", "plan_name:postgresql"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: describe service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "describe", serviceName, "postgresql"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list service plan versions
	cmd.RootCmd.SetArgs([]string{"service-plan", "list-versions", serviceName, "postgresql", "--filter", "service_name:postgresql", "--filter", "plan_name:postgresql", "--latest-n=1"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: describe service plan version
	cmd.RootCmd.SetArgs([]string{"service-plan", "describe-version", serviceName, "postgresql", "--version=latest"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete postgresql service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "delete", serviceName, "postgresql"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete postgresql service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: create postgresql service
	serviceName = "postgresql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/postgresql.yaml", "--name", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
	serviceID := build.ServiceID
	productTierID := build.ProductTierID

	// PASS: release postgresql service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "release", "--service-id", serviceID, "--plan-id", productTierID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: set it as preferred
	cmd.RootCmd.SetArgs([]string{"service-plan", "set-default", "--service-id", serviceID, "--plan-id", productTierID, "--version=latest"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list service plans
	cmd.RootCmd.SetArgs([]string{"service-plan", "list", "--filter", fmt.Sprintf("service_id:%s", serviceID), "--filter", fmt.Sprintf("plan_id:%s", productTierID)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: describe service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "describe", "--service-id", serviceID, "--plan-id", productTierID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list service plan versions
	cmd.RootCmd.SetArgs([]string{"service-plan", "list-versions", "--service-id", serviceID, "--plan-id", productTierID, "--limit=1"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: describe service plan version
	cmd.RootCmd.SetArgs([]string{"service-plan", "describe-version", "--service-id", serviceID, "--plan-id", productTierID, "--version=preferred"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete postgresql service plan
	cmd.RootCmd.SetArgs([]string{"service-plan", "delete", "--service-id", serviceID, "--plan-id", productTierID})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete postgresql service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_service_plan_features_modification(t *testing.T) {
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

	// PASS: create postgresql service
	serviceName := "postgresql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/byoa_postgresql.yaml", "--name", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: enable CUSTOM_TERRAFORM_POLICY feature
	cmd.RootCmd.SetArgs([]string{"service-plan", "enable-feature", serviceName, "BYOA Postgres", "--feature", "CUSTOM_TERRAFORM_POLICY", "--feature-configuration-file", "../build/configfiles/customTfPolicyFeatureConfig.json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: disable CUSTOM_TERRAFORM_POLICY feature
	cmd.RootCmd.SetArgs([]string{"service-plan", "disable-feature", serviceName, "BYOA Postgres", "--feature", "CUSTOM_TERRAFORM_POLICY"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: delete the service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}
