package cost

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCostCommands(t *testing.T) {
	require := require.New(t)

	// Test that the cost command has all expected subcommands
	expectedCommands := []string{"cloud-provider", "deployment-cell", "region", "user"}
	
	require.Equal("cost [operation] [flags]", Cmd.Use)
	require.Contains(Cmd.Short, "Manage cost analytics")
	
	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}
	
	for _, expected := range expectedCommands {
		require.Contains(actualCommands, expected, "Expected command %s not found", expected)
	}
}

func TestCloudProviderCommandFlags(t *testing.T) {
	require := require.New(t)
	
	// Test that cloud-provider command has expected flags
	require.NotNil(cloudProviderCmd.Flag("start-date"))
	require.NotNil(cloudProviderCmd.Flag("end-date"))
	require.NotNil(cloudProviderCmd.Flag("environment-type"))
	require.NotNil(cloudProviderCmd.Flag("frequency"))
	require.NotNil(cloudProviderCmd.Flag("include-providers"))
	require.NotNil(cloudProviderCmd.Flag("exclude-providers"))
}

func TestDeploymentCellCommandFlags(t *testing.T) {
	require := require.New(t)
	
	// Test that deployment-cell command has expected flags
	require.NotNil(deploymentCellCmd.Flag("start-date"))
	require.NotNil(deploymentCellCmd.Flag("end-date"))
	require.NotNil(deploymentCellCmd.Flag("environment-type"))
	require.NotNil(deploymentCellCmd.Flag("frequency"))
	require.NotNil(deploymentCellCmd.Flag("include-providers"))
	require.NotNil(deploymentCellCmd.Flag("exclude-providers"))
	require.NotNil(deploymentCellCmd.Flag("include-cells"))
	require.NotNil(deploymentCellCmd.Flag("exclude-cells"))
	require.NotNil(deploymentCellCmd.Flag("include-instances"))
	require.NotNil(deploymentCellCmd.Flag("exclude-instances"))
	require.NotNil(deploymentCellCmd.Flag("top-n-instances"))
}

func TestRegionCommandFlags(t *testing.T) {
	require := require.New(t)
	
	// Test that region command has expected flags
	require.NotNil(regionCmd.Flag("start-date"))
	require.NotNil(regionCmd.Flag("end-date"))
	require.NotNil(regionCmd.Flag("environment-type"))
	require.NotNil(regionCmd.Flag("frequency"))
	require.NotNil(regionCmd.Flag("include-providers"))
	require.NotNil(regionCmd.Flag("exclude-providers"))
	require.NotNil(regionCmd.Flag("include-regions"))
	require.NotNil(regionCmd.Flag("exclude-regions"))
	// Region command should not have instance flags since they're not supported by the API
	require.Nil(regionCmd.Flag("include-instances"))
	require.Nil(regionCmd.Flag("exclude-instances"))
}

func TestUserCommandFlags(t *testing.T) {
	require := require.New(t)
	
	// Test that user command has expected flags
	require.NotNil(userCmd.Flag("start-date"))
	require.NotNil(userCmd.Flag("end-date"))
	require.NotNil(userCmd.Flag("environment-type"))
	require.NotNil(userCmd.Flag("include-users"))
	require.NotNil(userCmd.Flag("exclude-users"))
	require.NotNil(userCmd.Flag("top-n-users"))
	require.NotNil(userCmd.Flag("top-n-instances"))
	// User command should not have frequency flag since it's not supported by the API
	require.Nil(userCmd.Flag("frequency"))
}