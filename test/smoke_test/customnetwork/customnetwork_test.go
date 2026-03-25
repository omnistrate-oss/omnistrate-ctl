package customnetwork

import (
	"context"
	"fmt"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/customnetwork"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_custom_network_lifecycle(t *testing.T) {
	testutils.SmokeTest(t)

	require := require.New(t)
	defer testutils.Cleanup()
	ctx := context.Background()

	var err error

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.Execute()
	require.NoError(err)
	token, err := config.GetToken()

	// Pre-test cleanup
	deleteCustomNetworkIfExists(t, token, "aws", "ap-south-1", "ctl-test-network")
	require.NoError(err)

	// PASS: create custom network
	cmd.RootCmd.SetArgs([]string{"custom-network", "create", "--cloud-provider", "aws", "--region", "ap-south-1", "--cidr", "10.99.0.0/16", "--name", "ctl-test-network"})
	err = cmd.RootCmd.Execute()
	require.NoError(err)
	customNetworkID := customnetwork.CustomNetworkID
	require.NotEmpty(customNetworkID)

	// PASS: update custom network name
	cmd.RootCmd.SetArgs([]string{"custom-network", "update", "--custom-network-id", customNetworkID, "--name", "ctl-test-network updated"})
	err = cmd.RootCmd.Execute()
	require.NoError(err)

	// PASS: describe custom network
	cmd.RootCmd.SetArgs([]string{"custom-network", "describe", "--custom-network-id", customNetworkID})
	err = cmd.RootCmd.Execute()
	require.NoError(err)

	// PASS: describe custom network
	cmd.RootCmd.SetArgs([]string{"custom-network", "describe", "--custom-network-id", customNetworkID})
	err = cmd.RootCmd.Execute()
	require.NoError(err)

	// PASS: describe manually
	customNetwork, err := dataaccess.FleetDescribeCustomNetwork(ctx, token, customNetworkID)

	require.NoError(err)
	require.NotNil(customNetwork)
	require.Equal(customNetworkID, customNetwork.Id)
	require.NotNil(customNetwork.Name)
	require.Equal("ctl-test-network updated", *customNetwork.Name)
	require.Equal("10.99.0.0/16", customNetwork.Cidr)
	require.Equal("aws", customNetwork.CloudProviderName)
	require.Equal("ap-south-1", customNetwork.CloudProviderRegion)

	// PASS: list custom networks
	cmd.RootCmd.SetArgs([]string{"custom-network", "list", "--filter", "cloud_provider:aws,region:ap-south-1"})
	err = cmd.RootCmd.Execute()
	require.NoError(err)

	// PASS: delete custom network
	cmd.RootCmd.SetArgs([]string{"custom-network", "delete", "--custom-network-id", customNetworkID})
	err = cmd.RootCmd.Execute()
	require.NoError(err)

	// FAIL: describe again
	customNetwork, err = dataaccess.FleetDescribeCustomNetwork(ctx, token, customNetworkID)
	require.Error(err)
	require.Nil(customNetwork)
}

func deleteCustomNetworkIfExists(t *testing.T, token, cloudProvider, region, customNetworkName string) {
	ctx := context.Background()
	customNetworks, err := dataaccess.FleetListCustomNetworks(ctx, token, utils.ToPtr(cloudProvider), utils.ToPtr(region))
	require.NoError(t, err)

	for _, network := range customNetworks.CustomNetworks {
		if network.Name != nil && *network.Name == customNetworkName {
			err = dataaccess.FleetDeleteCustomNetwork(ctx, token, network.Id)
			require.NoError(t, err)
		}
	}
}
