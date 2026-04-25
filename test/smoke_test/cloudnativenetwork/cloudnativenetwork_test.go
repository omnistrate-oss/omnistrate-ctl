package cloudnativenetwork

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
)

// Test_cloud_native_network_discovery exercises the BYOA cloud-native-network
// discovery / import / unimport surface that backs the omni-test BYOC VPC
// onboarding suite. The test:
//  1. logs in with the smoke-test account,
//  2. resolves a BYOA AWS account-config (env TEST_BYOA_ACCOUNT_CONFIG_ID,
//     or a freshly created throwaway account),
//  3. runs `account cloud-native-network sync` to trigger discovery,
//  4. lists the discovered networks and asserts the fleet response carries the
//     new IMPORTED / IN USE fields,
//  5. imports one network and verifies Imported flips to true,
//  6. unimports it and verifies the flag clears,
//  7. cleans up the throwaway account-config (when it created one).
//
// The discovery + import/unimport phases are skipped when:
//   - the account has no regions enabled (sync returns a 400 about "regions")
//     because that requires service-plan registration, OR
//   - discovery returns no AVAILABLE networks (sandbox account may have no
//     default VPC).
func Test_cloud_native_network_discovery(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.Background()
	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	token, err := config.GetToken()
	require.NoError(err)

	accountConfigID := config.GetEnv("TEST_BYOA_ACCOUNT_CONFIG_ID", "")
	if accountConfigID == "" {
		accountConfigID = createThrowawayAccount(t, ctx)
	}
	require.NotEmpty(accountConfigID)

	syncRegions := []string{}
	if envRegions := config.GetEnv("TEST_BYOA_REGIONS", ""); envRegions != "" {
		syncRegions = strings.Split(envRegions, ",")
	}
	// Call the dataaccess layer directly so we can recognize the "no regions
	// resolved" 400 and skip cleanly. The cobra sync command spins up a TUI
	// spinner and calls os.Exit(1) on error, which would deadlock or kill the
	// test process.
	if _, err := dataaccess.SyncAccountConfigCloudNativeNetworks(ctx, token, accountConfigID, syncRegions); err != nil {
		if strings.Contains(err.Error(), "regions") {
			t.Skipf("sync requires a service-plan-registered account; set TEST_BYOA_ACCOUNT_CONFIG_ID to a registered account: %v", err)
		}
		require.NoError(err)
	}

	listResult, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountConfigID)
	require.NoError(err)
	require.NotNil(listResult)
	for _, vpc := range listResult.CloudNativeNetworks {
		require.NotNil(vpc.Imported, "fleet response missing 'imported' for %s", vpc.CloudNativeNetworkId)
		require.NotNil(vpc.InUse, "fleet response missing 'inUse' for %s", vpc.CloudNativeNetworkId)
	}

	// Also exercise the user-facing list command (json output disables the spinner).
	cmd.RootCmd.SetArgs([]string{"account", "cloud-native-network", "list", accountConfigID, "--output", "json"})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	available := firstAvailableNetwork(listResult.CloudNativeNetworks)
	if available == "" {
		t.Logf("no AVAILABLE networks discovered for account-config %s; skipping import/unimport phase", accountConfigID)
		return
	}

	cmd.RootCmd.SetArgs([]string{"account", "cloud-native-network", "import", accountConfigID, available, "--output", "json"})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	postImport, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountConfigID)
	require.NoError(err)
	require.True(networkImported(postImport.CloudNativeNetworks, available), "network %s should be imported after import call", available)

	cmd.RootCmd.SetArgs([]string{"account", "cloud-native-network", "unimport", accountConfigID, available, "--output", "json"})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	postUnimport, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountConfigID)
	require.NoError(err)
	require.False(networkImported(postUnimport.CloudNativeNetworks, available), "network %s should not be imported after unimport call", available)
}

// createThrowawayAccount creates a BYOA AWS account-config and registers a
// t.Cleanup hook to delete it via the dataaccess layer (going through the
// cobra command would block on the deferred testutils.Cleanup wiping the
// token before t.Cleanup fires). Returns the resolved account-config ID.
func createThrowawayAccount(t *testing.T, ctx context.Context) string {
	t.Helper()
	awsAccountName := "cnn" + uuid.NewString()
	// nolint:gosec
	rand12DigitsNum := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(900000000000) + 100000000000
	awsAccountID := fmt.Sprintf("%d", rand12DigitsNum)

	cmd.RootCmd.SetArgs([]string{"account", "create", awsAccountName, "--aws-account-id", awsAccountID, "--skip-wait"})
	require.NoError(t, cmd.RootCmd.ExecuteContext(ctx))

	token, err := config.GetToken()
	require.NoError(t, err)
	id := lookupAccountConfigID(t, ctx, token, awsAccountName)
	t.Cleanup(func() {
		// Snapshot the token before testutils.Cleanup wipes the config dir.
		if delErr := dataaccess.DeleteAccount(ctx, token, id); delErr != nil {
			t.Logf("best-effort throwaway account delete (%s) failed: %v", id, delErr)
		}
	})
	return id
}

func lookupAccountConfigID(t *testing.T, ctx context.Context, token, name string) string {
	t.Helper()
	accounts, err := dataaccess.ListAccounts(ctx, token, "all")
	require.NoError(t, err)
	for _, account := range accounts.AccountConfigs {
		if strings.EqualFold(account.Name, name) {
			return account.Id
		}
	}
	return ""
}

func firstAvailableNetwork(vpcs []openapiclientfleet.FleetAccountConfigCloudNativeNetworkResult) string {
	for _, vpc := range vpcs {
		if strings.EqualFold(vpc.Status, "AVAILABLE") {
			return vpc.CloudNativeNetworkId
		}
	}
	return ""
}

func networkImported(vpcs []openapiclientfleet.FleetAccountConfigCloudNativeNetworkResult, id string) bool {
	for _, vpc := range vpcs {
		if vpc.CloudNativeNetworkId == id && vpc.Imported != nil && *vpc.Imported {
			return true
		}
	}
	return false
}
