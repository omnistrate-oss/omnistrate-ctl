package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpgradeCommands(t *testing.T) {
	require.Equal(t, "upgrade --version=[version]", Cmd.Use)
	require.Equal(t, "Upgrade Instance Deployments to a newer or older version", Cmd.Short)

	// Check that new subcommands are added
	subCommands := []string{"list", "describe"}
	for _, subCmd := range subCommands {
		found := false
		for _, cmd := range Cmd.Commands() {
			if cmd.Use == subCmd || cmd.Use == subCmd+" <upgrade-path-id>" {
				found = true
				break
			}
		}
		require.True(t, found, "Expected subcommand %s not found", subCmd)
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd := listCmd

	require.Equal(t, "list", cmd.Use)
	require.Equal(t, "List upgrade paths", cmd.Short)

	// Check required flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	productTierIDFlag := cmd.Flags().Lookup("product-tier-id")
	require.NotNil(t, productTierIDFlag)
	require.Equal(t, "p", productTierIDFlag.Shorthand)

	// Check optional flags
	flags := []string{"source-version", "target-version", "status", "type", "next-page-token", "page-size"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found", flagName)
	}
}

func TestDescribeCommandFlags(t *testing.T) {
	cmd := describeCmd

	require.Equal(t, "describe <upgrade-path-id>", cmd.Use)
	require.Equal(t, "Describe an upgrade path", cmd.Short)

	// Check required flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	productTierIDFlag := cmd.Flags().Lookup("product-tier-id")
	require.NotNil(t, productTierIDFlag)
	require.Equal(t, "p", productTierIDFlag.Shorthand)
}
