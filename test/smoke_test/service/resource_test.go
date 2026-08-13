package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_service_resource_delete_removed_resource(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	token, err := common.GetTokenWithLogin()
	require.NoError(err)

	serviceName := "mysql cluster resource delete " + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{
		"build",
		"-f",
		"../composefiles/mysqlcluster.yaml",
		"--name",
		serviceName,
		"--description",
		"My Service Description",
		"--service-logo-url",
		"https://freepnglogos.com/uploads/server-png/server-computer-database-network-vector-graphic-pixabay-31.png",
		"--release",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	serviceID := build.ServiceID
	planID := build.ProductTierID
	require.NotEmpty(serviceID)
	require.NotEmpty(planID)
	defer func() {
		cmd.RootCmd.SetArgs([]string{"service", "delete", "--id", serviceID})
		_ = cmd.RootCmd.ExecuteContext(ctx)
	}()

	removedResourceID := resourceIDByName(ctx, t, token, serviceID, planID, "writer")
	require.NotEmpty(removedResourceID)

	cmd.RootCmd.SetArgs([]string{
		"build",
		"-f",
		"../composefiles/variations/mysqlcluster_variation_account_integration_resource.yaml",
		"--name",
		serviceName,
		"--release-as-preferred",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	cmd.RootCmd.SetArgs([]string{
		"service",
		"resource",
		"delete",
		"--service-id",
		serviceID,
		"--resource-id",
		removedResourceID,
		"--output",
		"json",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func resourceIDByName(ctx context.Context, t *testing.T, token, serviceID, planID, resourceName string) string {
	t.Helper()

	resources, err := dataaccess.ListResources(ctx, token, serviceID, planID, nil)
	require.NoError(t, err)

	for _, resource := range resources.GetResources() {
		if strings.EqualFold(resource.GetName(), resourceName) {
			return resource.GetId()
		}
	}

	require.Failf(t, "resource not found", "resource %q was not found in service %q plan %q", resourceName, serviceID, planID)
	return ""
}
