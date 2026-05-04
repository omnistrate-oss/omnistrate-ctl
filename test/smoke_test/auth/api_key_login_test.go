package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/require"
)

// Test_login_with_api_key exercises the full org-bounded api-key
// lifecycle through both the dataaccess layer (create/list/describe/
// revoke/delete) and the CLI using all supported login mechanisms:
//
//  1. Bootstrap an admin session via password login (uses TEST_EMAIL/
//     TEST_PASSWORD) so we have a JWT that can mint api-keys.
//  2. For each non-root role (admin, editor, reader, service_editor,
//     service_operator) create an api-key with a unique, test-scoped
//     name. The cleanup defer revokes and deletes every key created in
//     this run.
//  3. For each created key, exercise all three login mechanisms:
//       a. `omnistrate-ctl login --api-key=<key>` — plaintext flag.
//       b. `OMNISTRATE_API_KEY=<key> omnistrate-ctl login` — env var
//          auto-detection (zero-flag CI/CD path).
//       c. `echo <key> | omnistrate-ctl login --api-key-stdin` — stdin
//          pipe (secure, no process-visible plaintext).
//     After each mechanism, exchange the key via the dataaccess helper
//     and confirm the resulting JWT is accepted by ListServices.
//  4. After the three mechanisms, validate the token lifecycle:
//       a. Refresh the token and confirm the new JWT works.
//       b. Revoke the refreshed token and confirm it can no longer be
//          used for refresh.
//  5. Re-establish the admin session and confirm the test-scoped names
//     are present in the list response.
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

	// Step 3: for each key, exercise all supported login mechanisms.
	// Each mechanism is validated with a ListServices call to confirm
	// the resulting session is genuinely usable.
	for _, k := range created {
		// --- Mechanism 1: --api-key flag ---
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))

		cmd.RootCmd.SetArgs([]string{"login", "--api-key=" + k.Plaintext})
		require.NoErrorf(cmd.RootCmd.ExecuteContext(ctx),
			"login --api-key for role %s (key %s)", k.Role, k.Name)

		// Validate the session works via dataaccess exchange.
		session, err := dataaccess.LoginWithAPIKey(ctx, k.Plaintext)
		require.NoErrorf(err, "signin-exchange for role %s (key %s)", k.Role, k.Name)
		require.NotEmpty(session.JWTToken, "signin-exchange must return a JWT for role %s", k.Role)

		_, err = dataaccess.ListServices(ctx, session.JWTToken)
		require.NoErrorf(err, "ListServices with --api-key JWT for role %s (key %s)", k.Role, k.Name)

		// --- Mechanism 2: OMNISTRATE_API_KEY env var ---
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))

		os.Setenv(config.OmnistrateAPIKeyEnv, k.Plaintext)
		cmd.RootCmd.SetArgs([]string{"login"})
		err = cmd.RootCmd.ExecuteContext(ctx)
		os.Unsetenv(config.OmnistrateAPIKeyEnv)
		require.NoErrorf(err,
			"login via OMNISTRATE_API_KEY env var for role %s (key %s)", k.Role, k.Name)

		envSession, err := dataaccess.LoginWithAPIKey(ctx, k.Plaintext)
		require.NoErrorf(err, "signin-exchange (env var path) for role %s", k.Role)
		_, err = dataaccess.ListServices(ctx, envSession.JWTToken)
		require.NoErrorf(err, "ListServices with env-var JWT for role %s (key %s)", k.Role, k.Name)

		// --- Mechanism 3: --api-key-stdin ---
		cmd.RootCmd.SetArgs([]string{"logout"})
		require.NoError(cmd.RootCmd.ExecuteContext(ctx))

		origStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(err)
		os.Stdin = r
		_, err = w.WriteString(k.Plaintext + "\n")
		require.NoError(err)
		w.Close()

		cmd.RootCmd.SetArgs([]string{"login", "--api-key-stdin"})
		stdinErr := cmd.RootCmd.ExecuteContext(ctx)
		os.Stdin = origStdin
		require.NoErrorf(stdinErr,
			"login --api-key-stdin for role %s (key %s)", k.Role, k.Name)

		stdinSession, err := dataaccess.LoginWithAPIKey(ctx, k.Plaintext)
		require.NoErrorf(err, "signin-exchange (stdin path) for role %s", k.Role)
		_, err = dataaccess.ListServices(ctx, stdinSession.JWTToken)
		require.NoErrorf(err, "ListServices with stdin JWT for role %s (key %s)", k.Role, k.Name)

		// --- Token lifecycle (refresh + revoke) ---
		// Use the last session for lifecycle validation.
		require.NotEmptyf(stdinSession.RefreshToken,
			"signin-exchange must return a refresh token for role %s (key %s)", k.Role, k.Name)
		refreshed, err := dataaccess.RefreshToken(ctx, stdinSession.RefreshToken)
		require.NoErrorf(err, "RefreshToken for role %s (key %s)", k.Role, k.Name)
		require.NotEmpty(refreshed.JWTToken, "refreshed JWT must be non-empty")
		require.NotEmpty(refreshed.RefreshToken, "refreshed refresh token must be non-empty")
		_, err = dataaccess.ListServices(ctx, refreshed.JWTToken)
		require.NoErrorf(err, "ListServices with refreshed api-key JWT for role %s (key %s)", k.Role, k.Name)

		err = dataaccess.RevokeToken(ctx, refreshed.RefreshToken)
		require.NoErrorf(err, "RevokeToken for role %s (key %s)", k.Role, k.Name)
		_, err = dataaccess.RefreshToken(ctx, refreshed.RefreshToken)
		require.Errorf(err, "RefreshToken after revocation must fail for role %s (key %s)", k.Role, k.Name)

		// Log out before iterating to the next key.
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
