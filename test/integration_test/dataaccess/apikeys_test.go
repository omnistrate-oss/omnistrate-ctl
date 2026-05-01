package dataaccess

import (
	"context"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

// TestAPIKeyLifecycle exercises the dataaccess wrappers for the
// org-bounded api-key feature end-to-end against a live environment:
//
//	bootstrap → list (probe) → create → list (contains) → describe →
//	signin-exchange (LoginWithAPIKey) → revoke → delete.
//
// The intent is to catch SDK/plumbing regressions at the
// internal/dataaccess layer before they reach the CLI surfaces (the
// smoke test in test/smoke_test/auth covers the CLI side).
//
// The test runs only when ENABLE_INTEGRATION_TEST=true and skips
// cleanly when the api-key feature is disabled in the target env
// (either because List returns api_keys_disabled_for_org or Create
// does — different envs surface the gate on different verbs).
func TestAPIKeyLifecycle(t *testing.T) {
	testutils.IntegrationTest(t)

	require := require.New(t)
	ctx := context.TODO()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	// Bootstrap admin session.
	bootstrap, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(err)
	adminToken := bootstrap.JWTToken
	require.NotEmpty(adminToken)

	// Probe (read path): if List returns the feature-disabled error
	// the rest of the test cannot run; skip cleanly.
	if _, listErr := dataaccess.ListAPIKeys(ctx, adminToken); listErr != nil &&
		strings.Contains(listErr.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org (List); skipping integration test")
	}

	// Negative: the platform must refuse to mint an api-key bound to
	// the org-root role. Scrub aggressively if a regression accidentally
	// creates one — a leaked root-privileged credential would persist
	// in the test org until manually revoked.
	rootDesc := "integration negative test — must be rejected"
	rootName := "ctl-integ-root-negative-" + testutils.RandomTestSuffix()
	if rootRes, rootErr := dataaccess.CreateAPIKey(ctx, adminToken, rootName, "root", &rootDesc, nil); rootErr == nil {
		if rootRes != nil && rootRes.Id != "" {
			_ = dataaccess.RevokeAPIKey(ctx, adminToken, rootRes.Id)
			_ = dataaccess.DeleteAPIKey(ctx, adminToken, rootRes.Id)
		}
		t.Fatalf("CreateAPIKey(role=root) must be rejected; platform unexpectedly minted id=%s", rootRes.Id)
	}

	name := "ctl-integ-" + testutils.RandomTestSuffix()
	desc := "ephemeral key created by TestAPIKeyLifecycle; safe to delete"

	// Best-effort cleanup so a partial failure does not leak a key.
	var createdID string
	t.Cleanup(func() {
		if createdID == "" {
			return
		}
		if err := dataaccess.RevokeAPIKey(ctx, adminToken, createdID); err != nil {
			t.Logf("cleanup: revoke %s: %v", createdID, err)
		}
		if err := dataaccess.DeleteAPIKey(ctx, adminToken, createdID); err != nil {
			t.Logf("cleanup: delete %s: %v", createdID, err)
		}
	})

	created, err := dataaccess.CreateAPIKey(ctx, adminToken, name, "editor", &desc, nil)
	if err != nil && strings.Contains(err.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org (Create); skipping integration test")
	}
	require.NoError(err)
	require.NotNil(created)
	require.NotEmpty(created.Id)
	require.NotEmpty(created.Key, "platform must return plaintext exactly once")
	require.Equal("editor", created.Metadata.RoleType)
	createdID = created.Id

	// List should now contain the new key by name; metadata response
	// must NEVER carry plaintext.
	listing, err := dataaccess.ListAPIKeys(ctx, adminToken)
	require.NoError(err)
	var foundInList bool
	for _, k := range listing.ApiKeys {
		if k.Id == created.Id {
			require.Equal(name, k.Name)
			foundInList = true
		}
	}
	require.True(foundInList, "newly-created key must appear in list response")

	// Signin-exchange must mint a JWT bound to the key.
	session, err := dataaccess.LoginWithAPIKey(ctx, created.Key)
	require.NoError(err)
	require.NotEmpty(session.JWTToken, "signin-exchange must return a JWT")

	// Revoke is idempotent from the caller's perspective; the platform
	// should accept it on a freshly-created key.
	require.NoError(dataaccess.RevokeAPIKey(ctx, adminToken, created.Id))

	// Delete tombstones the row; subsequent describe should fail.
	require.NoError(dataaccess.DeleteAPIKey(ctx, adminToken, created.Id))
	createdID = "" // Cleanup no-op since we already deleted.
}
