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
//  3. For each created key:
//       a. Log out so the CLI is forced to re-authenticate.
//       b. Run `omnistrate-ctl login --api-key=…` against the live
//          signin endpoint.
//       c. Exercise the resulting session with a benign read
//          (ListServices) — proves the JWT is actually accepted by an
//          unrelated v1 endpoint, not just by the signin endpoint
//          that minted it.
//       d. Refresh the token via the dataaccess RefreshToken helper
//          using the refresh token returned alongside the JWT — proves
//          that api-key signin sessions are renewable like any other
//          session and the new JWT also works.
//       e. Log out before iterating to the next key so each key is
//          tested in isolation.
//  4. Re-establish the admin session and confirm the test-scoped
//     names are present in the list response.
//
// The test runs only when both ENABLE_SMOKE_TEST=true and the api-key
// feature is enabled in the target environment (dev). On any
// environment where FeatureAPIKeys is OFF, ListAPIKeys returns an
// empty array and CreateAPIKey returns 403 with
// "api_keys_disabled_for_org"; the test skips on either signal so it
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

	// Probe (read path): if List returns the feature-disabled error
	// the rest of the test cannot run; skip cleanly.
	if _, listErr := dataaccess.ListAPIKeys(ctx, adminToken); listErr != nil &&
		strings.Contains(listErr.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org (List); skipping smoke test")
	}

	// Negative path: the platform must refuse to mint an api-key
	// bound to the org-root role. Run before the positive mints so a
	// regression here fails fast and we never end up with a root-
	// privileged machine credential in the test org.
	rootDesc := "smoke negative test — must be rejected"
	rootName := fmt.Sprintf("ctl-smoke-root-negative-%s", testutils.RandomTestSuffix())
	if rootRes, rootErr := dataaccess.CreateAPIKey(ctx, adminToken, rootName, "root", &rootDesc, nil); rootErr == nil {
		// SECURITY: scrub immediately so the leaked credential does not
		// outlive the failing assertion.
		if rootRes != nil && rootRes.Id != "" {
			_ = dataaccess.RevokeAPIKey(ctx, adminToken, rootRes.Id)
			_ = dataaccess.DeleteAPIKey(ctx, adminToken, rootRes.Id)
		}
		require.FailNowf("root role must not be assignable to api keys",
			"CreateAPIKey(role=root) unexpectedly succeeded (id=%s)", rootRes.Id)
	}

	// Step 2: create one api-key per non-root role. service_editor
	// and service_operator are included alongside admin/editor/reader
	// so the smoke covers every role assignable to a machine
	// credential. The org root role is intentionally excluded — the
	// platform rejects it (verified separately below).
	roles := []string{"admin", "editor", "reader", "service_editor", "service_operator"}
	type createdKey struct {
		ID        string
		Name      string
		Plaintext string
		Role      string
	}
	created := make([]createdKey, 0, len(roles))

	// Cleanup runs before testutils.Cleanup so the org doesn't keep
	// the test keys around even if the assertions below fail. Revoke
	// before delete so even an interrupted run leaves no usable key
	// behind, only a tombstone.
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
		// Probe (write path): some envs expose the feature flag only on
		// the create path (List returns empty, Create returns 403). Skip
		// cleanly on that signal too.
		if err != nil && strings.Contains(err.Error(), "api_keys_disabled_for_org") {
			t.Skip("api keys disabled for org (Create); skipping smoke test")
		}
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

	// Step 3: log in with each key via the CLI, exercise the resulting
	// session, refresh the token, then log out before the next key.
	for _, k := range created {
		// (a) Log out so the CLI is forced to authenticate via the api-key.
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))

		// (b) Login with the key plaintext.
		cmd.RootCmd.SetArgs([]string{"login", "--api-key=" + k.Plaintext})
		require.NoErrorf(cmd.RootCmd.ExecuteContext(ctx),
			"login --api-key for role %s (key %s)", k.Role, k.Name)

		// We also exchange the key via the dataaccess helper so we can
		// inspect the JWT/refresh-token pair directly — the CLI persists
		// these to disk but does not surface them to test code.
		session, err := dataaccess.LoginWithAPIKey(ctx, k.Plaintext)
		require.NoErrorf(err, "signin-exchange for role %s (key %s)", k.Role, k.Name)
		require.NotEmpty(session.JWTToken, "signin-exchange must return a JWT for role %s", k.Role)

		// (c) Smoke a benign read against an unrelated v1 endpoint to
		// prove the JWT is genuinely accepted by the platform (not just
		// minted). ListServices is universally available to every role.
		_, err = dataaccess.ListServices(ctx, session.JWTToken)
		require.NoErrorf(err, "ListServices with api-key JWT for role %s (key %s)", k.Role, k.Name)

		// (d) Validate the session is renewable. Refresh tokens are
		// returned alongside the JWT for every signin path, including
		// signin-exchange; missing refresh token is a regression.
		require.NotEmptyf(session.RefreshToken,
			"signin-exchange must return a refresh token for role %s (key %s)", k.Role, k.Name)
		refreshed, err := dataaccess.RefreshToken(ctx, session.RefreshToken)
		require.NoErrorf(err, "RefreshToken for role %s (key %s)", k.Role, k.Name)
		require.NotEmpty(refreshed.JWTToken, "refreshed JWT must be non-empty")
		// Confirm the refreshed JWT is itself usable.
		_, err = dataaccess.ListServices(ctx, refreshed.JWTToken)
		require.NoErrorf(err, "ListServices with refreshed api-key JWT for role %s (key %s)", k.Role, k.Name)

		// (e) Log out so the next iteration starts from a clean slate.
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))
	}

	// Step 4: re-establish admin session and confirm the keys are visible.
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
