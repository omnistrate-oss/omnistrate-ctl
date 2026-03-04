package instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestInstanceBasic(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()

	defer testutils.Cleanup()

	// PASS: login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: build service
	log.Debug().Msg("Building service...")
	serviceName := "mysql" + uuid.NewString()
	cmd.RootCmd.SetArgs([]string{"build", "--file", "../composefiles/mysql.yaml", "--name", serviceName, "--environment=dev", "--environment-type=dev"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: create instance 1 with param
	log.Debug().Msg("Creating instance 1 with parameters from command line...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--tags", "environment=dev,owner=platform",
		"--param", `{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}`})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	instanceID1 := instance.InstanceID
	require.NotEmpty(t, instanceID1)

	// PASS: create instance 2 with param file
	log.Debug().Msg("Creating instance 2 with parameters from file...")
	cmd.RootCmd.SetArgs([]string{"instance", "create",
		fmt.Sprintf("--service=%s", serviceName),
		"--environment=dev",
		fmt.Sprintf("--plan=%s", serviceName),
		"--version=latest",
		"--resource=mySQL",
		"--cloud-provider=aws",
		"--region=ca-central-1",
		"--tags", "source=file",
		"--param-file", "paramfiles/instance_create_param.json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
	instanceID2 := instance.InstanceID
	require.NotEmpty(t, instanceID2)

	// PASS: describe instance 1
	log.Debug().Msg("Describing instance 1...")
	cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID1})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: describe instance 2
	log.Debug().Msg("Describing instance 2...")
	cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID2})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: modify instance 1 tags
	log.Debug().Msg("Modifying instance 1 tags...")
	cmd.RootCmd.SetArgs([]string{"instance", "modify", instanceID1, "--tags", "environment=prod,owner=platform"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	time.Sleep(60 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: stop instance 1
	cmd.RootCmd.SetArgs([]string{"instance", "stop", instanceID1})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusStopped)
	require.NoError(t, err)

	// PASS: start instance 1
	log.Debug().Msg("Starting instance 1...")
	cmd.RootCmd.SetArgs([]string{"instance", "start", instanceID1})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: restart instance 1
	log.Debug().Msg("Restarting instance 1...")
	cmd.RootCmd.SetArgs([]string{"instance", "restart", instanceID1})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	time.Sleep(60 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: update instance 1
	log.Debug().Msg("Updating instance 1 with new parameters...")
	cmd.RootCmd.SetArgs([]string{"instance", "update", instanceID1, "--param", `{"databaseName":"default","password":"updated_password","rootPassword":"updated_root_password","username":"user"}`})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	time.Sleep(60 * time.Second)
	log.Debug().Msg("Waiting for instance 1 to reach running status after update...")
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID1, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: update instance 2
	log.Debug().Msg("Updating instance 2 with new parameters from file...")
	cmd.RootCmd.SetArgs([]string{"instance", "update", instanceID2, "--param-file", "paramfiles/instance_update_param.json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	time.Sleep(60 * time.Second)
	err = testutils.WaitForInstanceToReachStatus(ctx, instanceID2, instance.InstanceStatusRunning)
	require.NoError(t, err)

	// PASS: instance list
	log.Debug().Msg("Listing all instances...")
	cmd.RootCmd.SetArgs([]string{"instance", "list"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: instance list with filters
	log.Debug().Msg("Listing instances with filters...")
	cmd.RootCmd.SetArgs([]string{"instance", "list", "-f", "environment:DEV,cloud_provider:gcp", "-f", "environment:Dev,cloud_provider:aws"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: instance list with single tag filter
	log.Debug().Msg("Listing instances with single tag filter...")
	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "environment=prod"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: instance list with multiple tag filters (both must match)
	log.Debug().Msg("Listing instances with multiple tag filters...")
	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "environment=prod", "--tag", "owner=platform"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: instance list with tag filter for second instance
	log.Debug().Msg("Listing instances with tag filter for second instance...")
	cmd.RootCmd.SetArgs([]string{"instance", "list", "--tag", "source=file"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: instance list combining regular filter and tag filter
	log.Debug().Msg("Listing instances with combined filters...")
	cmd.RootCmd.SetArgs([]string{"instance", "list", "-f", fmt.Sprintf("service:%s", serviceName), "--tag", "environment=prod"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: delete instance 1
	log.Debug().Msg("Deleting instance 1...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", instanceID1, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// PASS: delete instance 2
	log.Debug().Msg("Deleting instance 2...")
	cmd.RootCmd.SetArgs([]string{"instance", "delete", instanceID2, "--yes"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)

	// Wait for the instances to be deleted
	for {
		log.Debug().Msg("Waiting for instances to be deleted...")
		cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID1})
		err1 := cmd.RootCmd.ExecuteContext(ctx)

		cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID2})
		err2 := cmd.RootCmd.ExecuteContext(ctx)

		if err1 != nil && err2 != nil {
			break
		}

		time.Sleep(60 * time.Second)
	}

	// PASS: delete service
	cmd.RootCmd.SetArgs([]string{"service", "delete", serviceName})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(t, err)
}
