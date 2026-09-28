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
		"update", "suspend", "resume", "terminate",
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
		"allowed-deployment-locations",
	}
	for _, flagName := range optionalFlags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Expected optional flag %s not found", flagName)
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	cmd := updateCmd

	require.Equal(t, "update <subscription-id>", cmd.Use)
	require.Equal(t, "Update a subscription", cmd.Short)

	serviceIDFlag := cmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "s", serviceIDFlag.Shorthand)

	environmentIDFlag := cmd.Flags().Lookup("environment-id")
	require.NotNil(t, environmentIDFlag)
	require.Equal(t, "e", environmentIDFlag.Shorthand)

	locationsFlag := cmd.Flags().Lookup("allowed-deployment-locations")
	require.NotNil(t, locationsFlag)
	require.Equal(t, "string", locationsFlag.Value.Type())
}

func TestParseAllowedDeploymentLocations(t *testing.T) {
	locations, err := parseAllowedDeploymentLocations(`[{"cloudProvider":"aws","regions":["us-east-1","us-west-2"]},{"cloudProvider":"gcp"}]`)

	require.NoError(t, err)
	require.Len(t, locations, 2)
	require.Equal(t, "aws", locations[0].CloudProvider)
	require.Equal(t, []string{"us-east-1", "us-west-2"}, locations[0].Regions)
	require.Equal(t, "gcp", locations[1].CloudProvider)
	require.Nil(t, locations[1].Regions)
}

func TestParseAllowedDeploymentLocationsPreservesEmptyList(t *testing.T) {
	locations, err := parseAllowedDeploymentLocations(`[]`)

	require.NoError(t, err)
	require.NotNil(t, locations)
	require.Empty(t, locations)
}

func TestParseAllowedDeploymentLocationsRejectsNull(t *testing.T) {
	_, err := parseAllowedDeploymentLocations(`null`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--allowed-deployment-locations")
	require.Contains(t, err.Error(), "JSON array")
}

func TestParseAllowedDeploymentLocationsRejectsInvalidInput(t *testing.T) {
	_, err := parseAllowedDeploymentLocations(`{"cloudProvider":"aws"}`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--allowed-deployment-locations")
	require.Contains(t, err.Error(), "JSON array")
}
