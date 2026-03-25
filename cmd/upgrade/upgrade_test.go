package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpgradeCommands(t *testing.T) {
	require.Equal(t, "upgrade --version=[version]", Cmd.Use)
	require.Equal(t, "Upgrade Instance Deployments to a newer or older version", Cmd.Short)

	// Check that subcommands are registered
	subCommands := []string{"list", "describe", "create"}
	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}
	for _, expected := range subCommands {
		require.Contains(t, actualCommands, expected, "Expected subcommand %s not found", expected)
	}
}

func TestUpgradeCommandFlags(t *testing.T) {
	// Verify flags on the root upgrade command
	flags := []string{"version", "version-name", "scheduled-date", "notify-customer", "max-concurrent-upgrades"}
	for _, flagName := range flags {
		flag := Cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found on upgrade command", flagName)
	}
}

func TestCreateCommandFlags(t *testing.T) {
	require.Equal(t, "create [instance-id] [instance-id] ... --version=[version]", createCmd.Use)
	require.Equal(t, "Create an upgrade path for one or more instances", createCmd.Short)

	// Check flags
	flags := []string{"version", "version-name", "scheduled-date", "notify-customer", "max-concurrent-upgrades"}
	for _, flagName := range flags {
		flag := createCmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected flag %s not found on create command", flagName)
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
