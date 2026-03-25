package operations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperationsCommands(t *testing.T) {
	require.Equal(t, "operations", Cmd.Use)
	require.Equal(t, "Operations and health monitoring commands", Cmd.Short)
	require.Contains(t, Cmd.Long, "Manage and monitor operational health")

	// Check that all subcommands are added
	subCommands := []string{"health", "deployment-cell-health", "events"}
	for _, subCmd := range subCommands {
		found := false
		for _, cmd := range Cmd.Commands() {
			if cmd.Use == subCmd {
				found = true
				break
			}
		}
		require.True(t, found, "Expected subcommand %s not found", subCmd)
	}
}

func TestHealthCommandFlags(t *testing.T) {
	cmd := healthCmd

	require.Equal(t, "health", cmd.Use)
	require.Equal(t, "Get service health summary", cmd.Short)

	// Check required flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentIDFlag := cmd.Flags().Lookup("environment-id")
	require.NotNil(t, environmentIDFlag)
	require.Equal(t, "e", environmentIDFlag.Shorthand)
}

func TestDeploymentCellHealthCommandFlags(t *testing.T) {
	cmd := deploymentCellHealthCmd

	require.Equal(t, "deployment-cell-health", cmd.Use)
	require.Equal(t, "Get deployment cell health details", cmd.Short)

	// Check optional flags
	flags := []string{"host-cluster-id", "service-id", "environment-id"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found", flagName)
	}
}

func TestEventsCommandFlags(t *testing.T) {
	cmd := eventsCmd

	require.Equal(t, "events", cmd.Use)
	require.Equal(t, "List operational events", cmd.Short)

	// Check flags
	flags := []string{
		"next-page-token", "page-size", "environment-type", "event-types",
		"service-id", "service-environment-id", "instance-id",
		"start-date", "end-date", "product-tier-id",
	}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found", flagName)
	}

	// Check shorthand flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentTypeFlag := cmd.Flags().Lookup("environment-type")
	require.Equal(t, "e", environmentTypeFlag.Shorthand)

	instanceIDFlag := cmd.Flags().Lookup("instance-id")
	require.Equal(t, "i", instanceIDFlag.Shorthand)
}
