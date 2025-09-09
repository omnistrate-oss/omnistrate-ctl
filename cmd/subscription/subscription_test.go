package subscription

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionCommands(t *testing.T) {
	require.Equal(t, "subscription [operation] [flags]", Cmd.Use)
	require.Equal(t, "Manage Customer Subscriptions for your service", Cmd.Short)

	// Check that all subcommands are added
	expectedCommands := []string{
		"list", "list-for-service", "describe", "list-requests",
		"approve-request", "deny-request", "create-on-behalf",
		"suspend", "resume", "terminate",
	}
	
	for _, expectedCmd := range expectedCommands {
		found := false
		for _, cmd := range Cmd.Commands() {
			cmdName := cmd.Use
			if space := strings.Index(cmdName, " "); space != -1 {
				cmdName = cmdName[:space]
			}
			if cmdName == expectedCmd {
				found = true
				break
			}
		}
		require.True(t, found, "Expected subcommand %s not found", expectedCmd)
	}
}

func TestListRequestsCommandFlags(t *testing.T) {
	cmd := listRequestsCmd

	require.Equal(t, "list-requests", cmd.Use)
	require.Equal(t, "List subscription requests", cmd.Short)

	// Check required flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentIDFlag := cmd.Flags().Lookup("environment-id")
	require.NotNil(t, environmentIDFlag)
	require.Equal(t, "e", environmentIDFlag.Shorthand)
}

func TestApproveRequestCommandFlags(t *testing.T) {
	cmd := approveRequestCmd

	require.Equal(t, "approve-request <request-id>", cmd.Use)
	require.Equal(t, "Approve a subscription request", cmd.Short)

	// Check required flags
	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentIDFlag := cmd.Flags().Lookup("environment-id")
	require.NotNil(t, environmentIDFlag)
	require.Equal(t, "e", environmentIDFlag.Shorthand)
}

func TestCreateOnBehalfCommandFlags(t *testing.T) {
	cmd := createOnBehalfCmd

	require.Equal(t, "create-on-behalf", cmd.Use)
	require.Equal(t, "Create subscription on behalf of customer", cmd.Short)

	// Check required flags
	requiredFlags := []string{"service-id", "environment-id", "product-tier-id", "customer-user-id"}
	for _, flagName := range requiredFlags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected required flag %s not found", flagName)
	}

	// Check optional flags
	optionalFlags := []string{
		"allow-creates-without-payment", "billing-provider", "custom-price",
		"custom-price-per-unit", "external-payer-id", "max-instances", "price-effective-date",
	}
	for _, flagName := range optionalFlags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected optional flag %s not found", flagName)
	}
}