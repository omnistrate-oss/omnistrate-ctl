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

	// Also exercise the user-facing remove command via the data access layer
	// (list and import CLI commands were removed; they are now internal).

	available := firstAvailableNetwork(listResult.CloudNativeNetworks)
	if available == "" {
		t.Logf("no AVAILABLE networks discovered for account-config %s; skipping import/unimport phase", accountConfigID)
		return
	}

	_, err = dataaccess.ImportAccountConfigCloudNativeNetwork(ctx, token, accountConfigID, available)
	require.NoError(err)

	postImport, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountConfigID)
	require.NoError(err)
	require.True(networkImported(postImport.CloudNativeNetworks, available), "network %s should be imported after import call", available)

	cmd.RootCmd.SetArgs([]string{"account", "cloud-native-network", "remove", accountConfigID, "--network-id", available, "--output", "json"})
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
	require.NotEmpty(t, id, "failed to resolve account-config ID for %s", awsAccountName)
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

// Test_account_create_cloud_native_networks verifies that passing
// --cloud-native-networks to `account create` automatically syncs and imports
// the specified VPCs after the account becomes READY.
//
// Required environment variables:
//   - TEST_BYOA_ACCOUNT_CONFIG_ID: an existing READY BYOA account-config with
//     at least one discoverable VPC, OR the test will create a throwaway.
//   - TEST_CNN_REGION: region containing the VPC (e.g. "us-east-1")
//   - TEST_CNN_NETWORK_ID: the VPC ID to sync+import (e.g. "vpc-abc123")
//
// When TEST_CNN_REGION / TEST_CNN_NETWORK_ID are not set, the test discovers
// one from the existing account via sync and uses it.
func Test_account_create_cloud_native_networks(t *testing.T) {
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

	region := config.GetEnv("TEST_CNN_REGION", "")
	networkID := config.GetEnv("TEST_CNN_NETWORK_ID", "")

	// If region/networkID aren't explicitly set, discover one from an existing
	// account so the test is self-contained.
	if region == "" || networkID == "" {
		existingAccountID := config.GetEnv("TEST_BYOA_ACCOUNT_CONFIG_ID", "")
		if existingAccountID == "" {
			t.Skip("TEST_CNN_REGION and TEST_CNN_NETWORK_ID not set, and no TEST_BYOA_ACCOUNT_CONFIG_ID to discover from; skipping")
		}

		syncResult, syncErr := dataaccess.SyncAccountConfigCloudNativeNetworks(ctx, token, existingAccountID, nil)
		if syncErr != nil {
			t.Skipf("cannot sync existing account to discover networks: %v", syncErr)
		}
		for _, vpc := range syncResult.CloudNativeNetworks {
			if strings.EqualFold(vpc.Status, "AVAILABLE") {
				region = vpc.Region
				networkID = vpc.CloudNativeNetworkId
				break
			}
		}
		if region == "" || networkID == "" {
			t.Skip("no AVAILABLE networks discovered from existing account; cannot test --cloud-native-networks")
		}

		// Clean up: unimport if it was previously imported
		_ = ensureNetworkUnimported(ctx, token, existingAccountID, networkID)
	}

	// Create a new account with --cloud-native-networks flag.
	awsAccountName := "cnn-create-" + uuid.NewString()[:8]
	// nolint:gosec
	rand12DigitsNum := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(900000000000) + 100000000000
	awsAccountID := fmt.Sprintf("%d", rand12DigitsNum)
	cnnFlag := fmt.Sprintf("%s:%s", region, networkID)

	cmd.RootCmd.SetArgs([]string{
		"account", "create", awsAccountName,
		"--aws-account-id", awsAccountID,
		"--cloud-native-networks", cnnFlag,
		"--output", "json",
	})

	createErr := cmd.RootCmd.ExecuteContext(ctx)

	// Resolve account ID for cleanup regardless of create outcome.
	accountID := lookupAccountConfigID(t, ctx, token, awsAccountName)
	if accountID != "" {
		t.Cleanup(func() {
			// Unimport first so the account can be deleted cleanly.
			_, _ = dataaccess.UnimportAccountConfigCloudNativeNetwork(ctx, token, accountID, networkID)
			if delErr := dataaccess.DeleteAccount(ctx, token, accountID); delErr != nil {
				t.Logf("best-effort throwaway account delete (%s) failed: %v", accountID, delErr)
			}
		})
	}

	require.NoError(createErr, "account create with --cloud-native-networks should succeed")
	require.NotEmpty(accountID)

	// Verify the network was imported.
	listResult, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountID)
	require.NoError(err)
	require.True(networkImported(listResult.CloudNativeNetworks, networkID),
		"network %s should be imported after account create with --cloud-native-networks", networkID)
}

func ensureNetworkUnimported(ctx context.Context, token, accountID, networkID string) error {
	list, err := dataaccess.ListAccountConfigCloudNativeNetworks(ctx, token, accountID)
	if err != nil {
		return err
	}
	if networkImported(list.CloudNativeNetworks, networkID) {
		_, err = dataaccess.UnimportAccountConfigCloudNativeNetwork(ctx, token, accountID, networkID)
	}
	return err
}
