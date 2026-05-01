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
//  3. triggers discovery via dataaccess.SyncAccountConfigCloudNativeNetworks,
//  4. lists the discovered networks and asserts the fleet response carries the
//     new IMPORTED / IN USE fields,
//  5. imports one network and verifies Imported flips to true,
//  6. removes it via the CLI and verifies the flag clears,
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
	// Call the dataaccess layer directly so we can inspect the error and
	// skip cleanly without going through interactive CLI behavior.
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

	// Exercise the user-facing remove CLI command.
	// Sync, list, and import are internal-only and exercised through the dataaccess layer above.

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

// Test_account_customer_create_cloud_native_networks verifies that passing
// --cloud-native-networks to `account customer create` automatically syncs
// and imports the specified VPCs after the BYOA account becomes READY.
//
// Required environment variables:
//   - TEST_CNN_SERVICE: service name or ID for the BYOA plan
//   - TEST_CNN_ENVIRONMENT: environment name or ID
//   - TEST_CNN_PLAN: plan name or ID (must be a BYOA plan)
//   - TEST_CNN_REGION: region containing the VPC (e.g. "us-east-1")
//   - TEST_CNN_NETWORK_ID: the VPC ID to sync+import (e.g. "vpc-abc123")
//   - Optionally TEST_CNN_CUSTOMER_EMAIL: customer email for production envs
//
// When any of the required variables are missing, the test is skipped.
func Test_account_customer_create_cloud_native_networks(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.Background()
	require := require.New(t)
	defer testutils.Cleanup()

	service := config.GetEnv("TEST_CNN_SERVICE", "")
	environment := config.GetEnv("TEST_CNN_ENVIRONMENT", "")
	plan := config.GetEnv("TEST_CNN_PLAN", "")
	region := config.GetEnv("TEST_CNN_REGION", "")
	networkID := config.GetEnv("TEST_CNN_NETWORK_ID", "")

	if service == "" || environment == "" || plan == "" || region == "" || networkID == "" {
		t.Skip("TEST_CNN_SERVICE, TEST_CNN_ENVIRONMENT, TEST_CNN_PLAN, TEST_CNN_REGION, and TEST_CNN_NETWORK_ID must all be set; skipping")
	}

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	token, err := config.GetToken()
	require.NoError(err)

	// nolint:gosec
	rand12DigitsNum := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(900000000000) + 100000000000
	awsAccountID := fmt.Sprintf("%d", rand12DigitsNum)
	cnnFlag := fmt.Sprintf("%s:%s", region, networkID)

	cmdArgs := []string{
		"account", "customer", "create",
		"--service", service,
		"--environment", environment,
		"--plan", plan,
		"--aws-account-id", awsAccountID,
		"--cloud-native-networks", cnnFlag,
		"--output", "json",
	}

	if customerEmail := config.GetEnv("TEST_CNN_CUSTOMER_EMAIL", ""); customerEmail != "" {
		cmdArgs = append(cmdArgs, "--customer-email", customerEmail)
	}

	cmd.RootCmd.SetArgs(cmdArgs)
	createErr := cmd.RootCmd.ExecuteContext(ctx)

	// Best-effort cleanup: find the backing account-config and delete it.
	accounts, _ := dataaccess.ListAccounts(ctx, token, "all")
	if accounts != nil {
		for _, account := range accounts.AccountConfigs {
			if account.TargetAccountId == awsAccountID {
				t.Cleanup(func() {
					_, _ = dataaccess.UnimportAccountConfigCloudNativeNetwork(ctx, token, account.Id, networkID)
					if delErr := dataaccess.DeleteAccount(ctx, token, account.Id); delErr != nil {
						t.Logf("best-effort cleanup of account %s failed: %v", account.Id, delErr)
					}
				})
				break
			}
		}
	}

	require.NoError(createErr, "account customer create with --cloud-native-networks should succeed")
}
