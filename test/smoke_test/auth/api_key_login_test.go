package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

// Test_login_with_api_key exercises the full org-bounded api-key
// lifecycle through both the dataaccess layer (create/list/describe/
// revoke/delete) and the CLI (`login --api-key`):
//
//  1. Bootstrap an admin session via password login (uses TEST_EMAIL/
//     TEST_PASSWORD) so we have a JWT that can mint api-keys.
//  2. For each non-root role (admin, editor, reader) create an api-key
//     with a unique, test-scoped name. The cleanup defer revokes and
//     deletes every key created in this run, even if any later step
//     fails — leaving keys behind would leak credentials and pollute
//     the org listing.
//  3. For each created key, log out and run `omnistrate-ctl login
//     --api-key=…` against the live signin endpoint; assert no error
//     (the signin-exchange path returns a JWT bound to the key's
//     backing user).
//  4. As a paranoia check, list api-keys after logout and re-login
//     and verify the test-scoped names are present.
//
// The test runs only when both ENABLE_SMOKE_TEST=true and the api-key
// feature is enabled in the target environment (dev). On any
// environment where FeatureAPIKeys is OFF, ListAPIKeys returns an
// empty array and CreateAPIKey returns 403 with
// "api_keys_disabled_for_org"; the test skips on that signal so it
// can be left enabled in CI without flapping against unsupported
// environments.
func Test_login_with_api_key(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	// Step 1: bootstrap admin session.
	cmd.RootCmd.SetArgs([]string{"login", "--email=" + testEmail, "--password=" + testPassword})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	bootstrap, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(err)
	adminToken := bootstrap.JWTToken
	require.NotEmpty(adminToken)

	// Probe: skip cleanly when the feature is disabled in this env.
	if _, listErr := dataaccess.ListAPIKeys(ctx, adminToken); listErr != nil &&
		strings.Contains(listErr.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org; skipping smoke test")
	}

	// Step 2: create one api-key per non-root role.
	roles := []string{"admin", "editor", "reader"}
	type createdKey struct {
		ID        string
		Name      string
		Plaintext string
		Role      string
	}
	created := make([]createdKey, 0, len(roles))

	// Cleanup runs before testutils.Cleanup so the org doesn't keep
	// the test keys around even if the assertions below fail.
	t.Cleanup(func() {
		for _, k := range created {
			if err := dataaccess.RevokeAPIKey(ctx, adminToken, k.ID); err != nil {
				t.Logf("cleanup: revoke api key %s (%s): %v", k.ID, k.Name, err)
			}
			if err := dataaccess.DeleteAPIKey(ctx, adminToken, k.ID); err != nil {
				t.Logf("cleanup: delete api key %s (%s): %v", k.ID, k.Name, err)
			}
		}
	})

	runID := testutils.RandomTestSuffix()
	for _, role := range roles {
		name := fmt.Sprintf("ctl-smoke-%s-%s", role, runID)
		desc := "ephemeral key created by Test_login_with_api_key; safe to delete"
		res, err := dataaccess.CreateAPIKey(ctx, adminToken, name, role, &desc, nil)
		require.NoErrorf(err, "create api key with role %s", role)
		require.NotNil(res)
		require.NotEmpty(res.Key, "platform must return plaintext exactly once")
		require.NotEmpty(res.Id)
		require.Equal(role, res.Metadata.RoleType, "metadata role must match request")
		created = append(created, createdKey{
			ID:        res.Id,
			Name:      name,
			Plaintext: res.Key,
			Role:      role,
		})
	}

	// Step 3: log in with each key via the CLI.
	for _, k := range created {
		// Log out so the CLI is forced to authenticate via the api-key.
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))

		cmd.RootCmd.SetArgs([]string{"login", "--api-key=" + k.Plaintext})
		err := cmd.RootCmd.ExecuteContext(ctx)
		require.NoErrorf(err, "login --api-key for role %s (key %s)", k.Role, k.Name)
	}

	// Step 4: re-establish admin session and confirm the keys are visible.
	cmd.RootCmd.SetArgs([]string{"logout"})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))
	cmd.RootCmd.SetArgs([]string{"login", "--email=" + testEmail, "--password=" + testPassword})
	require.NoError(cmd.RootCmd.ExecuteContext(ctx))

	listing, err := dataaccess.ListAPIKeys(ctx, adminToken)
	require.NoError(err)
	seen := map[string]bool{}
	for _, k := range listing.ApiKeys {
		seen[k.Name] = true
	}
	for _, k := range created {
		require.Truef(seen[k.Name], "api key %s missing from list response", k.Name)
	}
}
