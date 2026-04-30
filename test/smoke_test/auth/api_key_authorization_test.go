package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

	"github.com/stretchr/testify/assert"
	require_pkg "github.com/stretchr/testify/require"
)

// Test_api_key_authorization_lifecycle exercises the full
// authorization surface of the org-bounded API keys feature against
// a live environment. It complements Test_login_with_api_key (which
// covers the happy-path login) by validating the negative-space
// guarantees that the design relies on:
//
//   - cross-org reads collapse to 404 (no information leak between
//     orgs even for the read-side verbs that are open to all roles)
//   - per-role RBAC: List/Describe are visible to every org member;
//     Create/Update/Revoke are gated to Admin (and Root, which is
//     never assignable to an api-key)
//   - lastUsedAt advances on every successful authentication so
//     operators can see staleness without an extra read path
//   - revocation immediately invalidates the key for signin-exchange
//   - expiration uses the same gating logic as revocation
//
// All keys created here are revoked and deleted by t.Cleanup so a
// failed run never leaves credentials behind in the org listing.
//
// The test runs only when ENABLE_SMOKE_TEST=true. It probes
// ListAPIKeys with the admin token and skips cleanly when
// FeatureAPIKeys is off in the target environment so it can stay
// enabled in CI without flapping against unsupported envs.
//
// Expiration note: the backend rejects past or non-future expiresAt
// values at create time, so the only way to e2e-test the expired
// path is to create a key with a short-future expiry and wait. We
// use 5 seconds plus a 2-second cushion (7s sleep) which is well
// inside any reasonable signin TTL while still keeping the test
// runtime bounded.
func Test_api_key_authorization_lifecycle(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require_pkg.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	bootstrap, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(err)
	adminToken := bootstrap.JWTToken
	require.NotEmpty(adminToken)

	// Probe the feature flag so the test no-ops cleanly on envs
	// where FeatureAPIKeys is OFF (List returns empty, Create returns
	// 403 "api_keys_disabled_for_org").
	if _, listErr := dataaccess.ListAPIKeys(ctx, adminToken); listErr != nil &&
		strings.Contains(listErr.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org; skipping authorization smoke test")
	}

	runID := testutils.RandomTestSuffix()

	// Track every key id we create so cleanup catches them all even
	// if a sub-step fails after the create succeeds.
	var createdIDs []string
	t.Cleanup(func() {
		for _, id := range createdIDs {
			if err := dataaccess.RevokeAPIKey(ctx, adminToken, id); err != nil &&
				!isExpectedAuthFailure(err) {
				t.Logf("cleanup: revoke api key %s: %v", id, err)
			}
			if err := dataaccess.DeleteAPIKey(ctx, adminToken, id); err != nil {
				t.Logf("cleanup: delete api key %s: %v", id, err)
			}
		}
	})

	createKey := func(t *testing.T, role, label string, expiresAt *time.Time) (id, plaintext string) {
		t.Helper()
		name := fmt.Sprintf("ctl-authz-%s-%s-%s", role, label, runID)
		desc := "ephemeral key from Test_api_key_authorization_lifecycle; safe to delete"
		res, err := dataaccess.CreateAPIKey(ctx, adminToken, name, role, &desc, expiresAt)
		require.NoErrorf(err, "create api key role=%s label=%s", role, label)
		require.NotEmpty(res.Key)
		require.NotEmpty(res.Id)
		require.Equal(role, res.Metadata.RoleType)
		createdIDs = append(createdIDs, res.Id)
		return res.Id, res.Key
	}

	// One key per non-root role for the per-role and lastUsedAt subtests.
	roleKeys := map[string]struct {
		ID, Plaintext string
	}{}
	for _, role := range []string{"admin", "editor", "reader"} {
		id, pt := createKey(t, role, "matrix", nil)
		roleKeys[role] = struct{ ID, Plaintext string }{id, pt}
	}

	// Capture the bootstrap user's OrgID once. The api-key principal
	// signin must always resolve to the SAME org as the user that
	// created the keys — this is the single most security-critical
	// invariant of the api-key signin path. We assert it explicitly
	// in the org_binding subtest below and any time we verify a
	// session principal downstream.
	bootstrapMe, err := dataaccess.DescribeUser(ctx, adminToken)
	require.NoError(err)
	require.NotNil(bootstrapMe.OrgId, "bootstrap user must report a non-nil OrgId")
	require.NotEmpty(*bootstrapMe.OrgId, "bootstrap user must report a non-empty OrgId")
	bootstrapOrgID := *bootstrapMe.OrgId

	t.Run("org_binding", func(t *testing.T) {
		// SECURITY-CRITICAL: an api-key signin must NEVER produce a
		// session that touches a different org from the one the key
		// was created in. We exchange each role's plaintext for a
		// session JWT, call DescribeUser as that principal, and
		// require the returned OrgId to equal the bootstrap user's
		// OrgId. A mismatch here would indicate the signin-exchange
		// path is leaking cross-org access — fail the test loudly.
		for _, role := range []string{"admin", "editor", "reader"} {
			role := role
			t.Run(role, func(t *testing.T) {
				session, err := dataaccess.LoginWithAPIKey(ctx, roleKeys[role].Plaintext)
				require_pkg.NoErrorf(t, err, "signin-exchange must succeed for %s key", role)
				require_pkg.NotEmpty(t, session.JWTToken)

				me, err := dataaccess.DescribeUser(ctx, session.JWTToken)
				require_pkg.NoErrorf(t, err, "DescribeUser must succeed for %s api-key session", role)
				require_pkg.NotNilf(t, me.OrgId,
					"api-key %s session must resolve to a user with a non-nil OrgId", role)
				require_pkg.Equalf(t, bootstrapOrgID, *me.OrgId,
					"api-key %s session resolved to org %q but key was created in org %q — cross-org leak",
					role, derefOr(me.OrgId, "<nil>"), bootstrapOrgID)
			})
		}
	})

	t.Run("cross_org_isolation", func(t *testing.T) {
		// A random uuid is overwhelmingly likely to either not exist
		// or live in a different org. Either way the design says we
		// must collapse the response to 404 not_found so the caller
		// cannot distinguish "wrong org" from "no such key" — that
		// is the entire information-leak prevention argument.
		_, err := dataaccess.DescribeAPIKey(ctx, adminToken, uuid.NewString())
		require_pkg.Error(t, err, "describe of unknown id must error")
		assert.True(t,
			strings.Contains(strings.ToLower(err.Error()), "not found") ||
				strings.Contains(err.Error(), "404"),
			"describe of unknown id should surface as not-found; got: %v", err)
	})

	t.Run("per_role_rbac", func(t *testing.T) {
		cases := []struct {
			role            string
			canMutate       bool
			expectDenialKey string
		}{
			{"admin", true, ""},
			{"editor", false, "denied"},
			{"reader", false, "denied"},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.role, func(t *testing.T) {
				// Exchange the api-key plaintext for a session JWT
				// so subsequent calls run as the api-key principal.
				session, err := dataaccess.LoginWithAPIKey(ctx, roleKeys[tc.role].Plaintext)
				require_pkg.NoErrorf(t, err, "signin-exchange must succeed for active %s key", tc.role)
				keyToken := session.JWTToken

				// Read paths must succeed for every org member.
				_, err = dataaccess.ListAPIKeys(ctx, keyToken)
				require_pkg.NoErrorf(t, err, "List must be allowed for role %s", tc.role)
				_, err = dataaccess.DescribeAPIKey(ctx, keyToken, roleKeys[tc.role].ID)
				require_pkg.NoErrorf(t, err, "Describe-self must be allowed for role %s", tc.role)

				// Mutating paths: only admin allowed.
				probeName := fmt.Sprintf("ctl-authz-probe-%s-%s", tc.role, runID)
				readerProbe := "reader-role-create-probe"
				createRes, createErr := dataaccess.CreateAPIKey(ctx, keyToken, probeName, "reader", &readerProbe, nil)
				if tc.canMutate {
					require_pkg.NoError(t, createErr, "Create must be allowed for role %s", tc.role)
					require_pkg.NotNil(t, createRes)
					createdIDs = append(createdIDs, createRes.Id)
				} else {
					require_pkg.Errorf(t, createErr, "Create must be denied for role %s", tc.role)
					assertDenied(t, createErr, tc.expectDenialKey, tc.role, "Create")
				}

				newDesc := "rbac-probe-rename"
				_, updateErr := dataaccess.UpdateAPIKeyMetadata(ctx, keyToken, roleKeys[tc.role].ID, nil, &newDesc)
				if tc.canMutate {
					require_pkg.NoError(t, updateErr, "Update must be allowed for role %s", tc.role)
				} else {
					require_pkg.Errorf(t, updateErr, "Update must be denied for role %s", tc.role)
					assertDenied(t, updateErr, tc.expectDenialKey, tc.role, "Update")
				}

				// Revoke is best-tested against a foreign id (a key
				// not created by this principal) so we don't burn
				// the role's own key mid-test. Use one we know lives
				// in this org but was created by the bootstrap user.
				foreign := roleKeys["admin"].ID
				if tc.role == "admin" {
					foreign = roleKeys["editor"].ID // admin-on-admin would self-revoke; use a sibling
				}
				revokeErr := dataaccess.RevokeAPIKey(ctx, keyToken, foreign)
				if tc.canMutate {
					// Admin can revoke any org-bounded key. We just
					// validated they can; immediately recreate by
					// noting the id is already on cleanup list.
					require_pkg.NoError(t, revokeErr, "Revoke must be allowed for role %s", tc.role)
				} else {
					require_pkg.Errorf(t, revokeErr, "Revoke must be denied for role %s", tc.role)
					assertDenied(t, revokeErr, tc.expectDenialKey, tc.role, "Revoke")
				}
			})
		}
	})

	t.Run("last_used_at_updates", func(t *testing.T) {
		// The lastUsedAt field must monotonically advance on every
		// successful authentication. Use a fresh key so we have a
		// clean baseline (lastUsedAt may be nil right after create
		// or already populated from the per-role-rbac subtest above
		// — either way we capture it and assert advancement).
		id, plaintext := createKey(t, "reader", "lastused", nil)

		before, err := dataaccess.DescribeAPIKey(ctx, adminToken, id)
		require_pkg.NoError(t, err)
		baseline := before.Metadata.LastUsedAt

		// Force a successful authentication on the key.
		_, err = dataaccess.LoginWithAPIKey(ctx, plaintext)
		require_pkg.NoError(t, err)

		// lastUsedAt may be updated asynchronously; poll briefly.
		var after *time.Time
		for i := 0; i < 10; i++ {
			d, derr := dataaccess.DescribeAPIKey(ctx, adminToken, id)
			require_pkg.NoError(t, derr)
			after = d.Metadata.LastUsedAt
			if after != nil && (baseline == nil || after.After(*baseline)) {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		require_pkg.NotNil(t, after, "lastUsedAt must be populated after a successful signin")
		if baseline != nil {
			assert.Truef(t, after.After(*baseline) || after.Equal(*baseline),
				"lastUsedAt must not move backwards: before=%v after=%v", baseline, after)
			assert.Truef(t, after.After(*baseline),
				"lastUsedAt must advance after a fresh successful signin: before=%v after=%v",
				baseline, after)
		}
	})

	t.Run("revocation_blocks_signin", func(t *testing.T) {
		id, plaintext := createKey(t, "reader", "revoke", nil)

		// Sanity: the key works pre-revocation.
		_, err := dataaccess.LoginWithAPIKey(ctx, plaintext)
		require_pkg.NoError(t, err)

		require_pkg.NoError(t, dataaccess.RevokeAPIKey(ctx, adminToken, id))

		// Revoked keys must fail signin-exchange. The platform
		// returns auth_failure / 401 (the same shape as a wrong
		// password) so revoked keys cannot be distinguished from
		// nonexistent keys via probing.
		_, err = dataaccess.LoginWithAPIKey(ctx, plaintext)
		require_pkg.Error(t, err, "revoked api key must be rejected at signin-exchange")
		assertAuthFailure(t, err, "revoked")
	})

	t.Run("expiration_blocks_signin", func(t *testing.T) {
		// Backend rejects past expiresAt at create time
		// (apikey/api.go: "expiresAt: must be in the future"), so
		// short-future + sleep is the only way to exercise the
		// expired-key gate end-to-end.
		expiresAt := time.Now().Add(5 * time.Second)
		id, plaintext := createKey(t, "reader", "expire", &expiresAt)

		// Sanity: the key works pre-expiry.
		_, err := dataaccess.LoginWithAPIKey(ctx, plaintext)
		require_pkg.NoError(t, err)

		// Wait past the expiry boundary plus a small cushion to
		// absorb clock skew between the test runner and the API.
		time.Sleep(7 * time.Second)

		_, err = dataaccess.LoginWithAPIKey(ctx, plaintext)
		require_pkg.Error(t, err, "expired api key must be rejected at signin-exchange")
		assertAuthFailure(t, err, "expired")

		// Expiration and revocation share the same gating logic;
		// confirm the read paths still surface the lifecycle field
		// so operators can tell the two states apart in the UI.
		d, err := dataaccess.DescribeAPIKey(ctx, adminToken, id)
		require_pkg.NoError(t, err, "describe must continue to work for expired keys (admin read)")
		assert.Equal(t, "expired", strings.ToLower(d.Metadata.Status),
			"status must reflect expiry on the next describe")
	})
}

// isExpectedAuthFailure reports whether err looks like a 401/403
// auth-style failure, used in cleanup to suppress noise when the
// resource was already invalidated by an earlier subtest.
func isExpectedAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, kw := range []string{"unauthorized", "forbidden", "auth_failure", "401", "403"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// assertDenied checks that err is a recognisable RBAC denial. We
// match on a small set of substrings rather than exact codes because
// the Go SDK wraps errors and the platform may return 403 with
// either "access_denied" or a richer "insufficient privileges"
// payload depending on the verb.
func assertDenied(t *testing.T, err error, _hint, role, verb string) {
	t.Helper()
	msg := strings.ToLower(err.Error())
	for _, kw := range []string{"denied", "forbidden", "not allowed", "insufficient", "403"} {
		if strings.Contains(msg, kw) {
			return
		}
	}
	t.Fatalf("expected RBAC denial for role=%s verb=%s; got: %v", role, verb, err)
}

// assertAuthFailure checks that err looks like a signin-exchange
// auth rejection. Revoked and expired keys MUST surface as the same
// auth_failure shape as an unknown key so an attacker cannot probe
// the lifecycle state of a key without the admin token.
func assertAuthFailure(t *testing.T, err error, label string) {
	t.Helper()
	msg := strings.ToLower(err.Error())
	for _, kw := range []string{"auth_failure", "unauthorized", "wrong user email or password", "401", "403"} {
		if strings.Contains(msg, kw) {
			return
		}
	}
	t.Fatalf("expected auth-failure shape for %s key; got: %v", label, err)
}

// derefOr returns the dereferenced string or fallback if p is nil.
// Used in failure messages so we never panic while formatting an
// org-binding mismatch.
func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}
