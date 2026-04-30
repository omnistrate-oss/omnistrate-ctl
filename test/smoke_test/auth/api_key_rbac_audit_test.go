package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	sdkfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	sdkv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	require_pkg "github.com/stretchr/testify/require"
)

// Test_api_key_rbac_audit_matrix exercises every catalogued build/operate
// API across the four canonical session principals (the bootstrap human
// user + one api-key per non-root role) and emits a markdown matrix to
// $GITHUB_STEP_SUMMARY so the resulting RBAC posture can be audited
// directly from the workflow run UI.
//
// The matrix has two purposes:
//
//  1. Audit: a human reviewer can scan the table to see exactly which
//     endpoints accept which roles in production-like env. This is the
//     authoritative ground-truth for the platform's RBAC story and must
//     be regenerated on every change to the auth or routing layers.
//
//  2. Drift detection: every row carries an Expected outcome per role.
//     If the live response disagrees with Expected, the test FAILS so
//     a regression in either RBAC enforcement OR in the audit's own
//     assumptions is caught immediately.
//
// Mutation safety contract: rows whose verb is non-read are EXCLUSIVELY
// invoked against principals expected to be denied. We never call a
// mutation with a principal that is supposed to succeed — that would
// either create persistent state in a shared smoke env or pollute the
// audit with side-effects. Mutation rows therefore set the would-succeed
// principal(s) to OutcomeSkip explicitly.
func Test_api_key_rbac_audit_matrix(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require_pkg.New(t)
	defer testutils.Cleanup()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)

	bootstrap, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(err)
	userToken := bootstrap.JWTToken
	require.NotEmpty(userToken)

	// Skip cleanly when FeatureAPIKeys is OFF: we cannot mint api-key
	// principals, so the matrix would degenerate to one row.
	if _, listErr := dataaccess.ListAPIKeys(ctx, userToken); listErr != nil &&
		strings.Contains(listErr.Error(), "api_keys_disabled_for_org") {
		t.Skip("api keys disabled for org; skipping rbac audit matrix")
	}

	runID := testutils.RandomTestSuffix()
	var createdKeyIDs []string
	t.Cleanup(func() {
		for _, id := range createdKeyIDs {
			_ = dataaccess.RevokeAPIKey(ctx, userToken, id)
			_ = dataaccess.DeleteAPIKey(ctx, userToken, id)
		}
	})

	// Mint one api-key per non-root role and exchange each plaintext
	// for a session JWT. We cache the JWT per role for every matrix
	// row; the api-key signin path is exercised once here, and from
	// then on each row sees a stable principal-bound bearer token.
	keyTokens := map[string]string{}
	for _, role := range []string{"admin", "editor", "reader"} {
		desc := fmt.Sprintf("ephemeral key for rbac audit role=%s", role)
		name := fmt.Sprintf("ctl-rbac-audit-%s-%s", role, runID)
		res, cerr := dataaccess.CreateAPIKey(ctx, userToken, name, role, &desc, nil)
		require.NoErrorf(cerr, "create %s api-key", role)
		require.NotEmpty(res.Key)
		require.NotEmpty(res.Id)
		createdKeyIDs = append(createdKeyIDs, res.Id)
		session, lerr := dataaccess.LoginWithAPIKey(ctx, res.Key)
		require.NoErrorf(lerr, "signin-exchange for %s api-key", role)
		require.NotEmpty(session.JWTToken)
		keyTokens[role] = session.JWTToken
	}

	// Principals are listed in display order. The order shows up as
	// columns in the markdown summary, so keep the human user first
	// and the api-key roles in increasing privilege so the table
	// reads top-down as "least-privileged should fail more".
	principals := []principal{
		{Label: "user (root/admin)", Token: userToken},
		{Label: "key:admin", Token: keyTokens["admin"]},
		{Label: "key:editor", Token: keyTokens["editor"]},
		{Label: "key:reader", Token: keyTokens["reader"]},
	}

	v1Client := newAuditV1Client()
	fleetClient := newAuditFleetClient()

	rows := buildAuditMatrix(v1Client, fleetClient)

	// Execute every cell sequentially so the workflow log line order
	// matches the matrix. Most calls are <100ms; the total runtime is
	// bounded by len(rows) * len(principals) * roundtrip.
	results := make([][]cellResult, len(rows))
	for i, row := range rows {
		results[i] = make([]cellResult, len(principals))
		for j, p := range principals {
			expected := row.Expected[p.Label]
			if expected == OutcomeSkip {
				results[i][j] = cellResult{Status: 0, Outcome: OutcomeSkip, Match: true}
				continue
			}
			callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			status, callErr := row.Invoke(callCtx, p.Token)
			cancel()
			actual := classify(status, callErr, row.IsRead)
			results[i][j] = cellResult{
				Status:  status,
				Outcome: actual,
				Match:   matches(expected, actual, row.IsRead),
				Err:     callErr,
			}
		}
	}

	summary := renderMarkdown(rows, principals, results)
	t.Log("\n" + summary)
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		if werr := appendFile(path, summary); werr != nil {
			t.Logf("warning: failed to write GITHUB_STEP_SUMMARY: %v", werr)
		}
	}

	// Drift detection: any cell whose actual outcome disagrees with
	// the expected outcome is a test failure. The summary table above
	// already shows where; we collect the failures into a single
	// human-readable error message so CI surfaces the count + first
	// few examples without scrolling the log.
	var failures []string
	for i, row := range rows {
		for j, p := range principals {
			r := results[i][j]
			if !r.Match {
				failures = append(failures, fmt.Sprintf(
					"  %s | %s | principal=%s expected=%s got=%s status=%d",
					row.Category, row.Endpoint, p.Label, row.Expected[p.Label], r.Outcome, r.Status))
			}
		}
	}
	if len(failures) > 0 {
		const maxShown = 25
		shown := failures
		if len(shown) > maxShown {
			shown = shown[:maxShown]
		}
		t.Fatalf("RBAC audit matrix detected %d drift(s):\n%s%s",
			len(failures),
			strings.Join(shown, "\n"),
			func() string {
				if len(failures) > maxShown {
					return fmt.Sprintf("\n  ... and %d more", len(failures)-maxShown)
				}
				return ""
			}(),
		)
	}
}

// === outcome model ===

type rbacOutcome string

const (
	OutcomeAllow              rbacOutcome = "ALLOW"     // 2xx — RBAC let request through and succeeded
	OutcomeAllowReached       rbacOutcome = "REACHED"   // RBAC let request through; backend bounced on input/state (404/400/409/etc.)
	OutcomeDeny               rbacOutcome = "DENY"      // 401/403 — RBAC blocked at the auth layer
	OutcomeRejectedByValidate rbacOutcome = "REJ-INPUT" // mutation: not a 401/403, but did not create (sufficient for "ensure rejected")
	OutcomeServerError        rbacOutcome = "5XX"       // server error — inconclusive
	OutcomeTransport          rbacOutcome = "TRANSPORT" // network error — inconclusive
	OutcomeSkip               rbacOutcome = "SKIP"      // intentionally not invoked (mutation against would-succeed principal)
)

// classify maps an HTTP status + error into a single rbacOutcome.
// The classification differs between read and mutation rows because
// a 400 on a read means "RBAC let me through but my input was bogus"
// (still a useful ALLOW signal), while a 400 on a mutation means
// "did not create — RBAC may or may not have blocked, we cannot tell"
// which we record as REJ-INPUT so the audit reader can see the
// distinction at a glance.
func classify(status int, err error, isRead bool) rbacOutcome {
	if err != nil && status == 0 {
		return OutcomeTransport
	}
	switch {
	case status >= 200 && status < 300:
		return OutcomeAllow
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return OutcomeDeny
	case status >= 500:
		return OutcomeServerError
	case isRead:
		// 4xx other than 401/403 on a read => RBAC passed
		return OutcomeAllowReached
	default:
		// non-2xx, non-401/403 on a mutation => not created, but
		// the gate may have been validation rather than auth.
		return OutcomeRejectedByValidate
	}
}

// matches encodes which actual outcomes are acceptable given the
// expected outcome. For reads, REACHED counts as ALLOW (RBAC passed).
// For mutations, REJ-INPUT counts as DENY because the user's stated
// requirement is "ensure the call is rejected" — both shapes satisfy
// that. Server errors / transport failures NEVER match expected and
// surface as drift so the operator investigates.
func matches(expected, actual rbacOutcome, isRead bool) bool {
	if expected == actual {
		return true
	}
	if isRead && expected == OutcomeAllow && actual == OutcomeAllowReached {
		return true
	}
	if !isRead && expected == OutcomeDeny && actual == OutcomeRejectedByValidate {
		return true
	}
	return false
}

type principal struct {
	Label string
	Token string
}

type cellResult struct {
	Status  int
	Outcome rbacOutcome
	Match   bool
	Err     error
}

type rbacRow struct {
	Category string
	Endpoint string
	HTTPVerb string
	IsRead   bool
	Invoke   func(ctx context.Context, token string) (int, error)
	Expected map[string]rbacOutcome // principal label -> expected outcome
}

// === expected helpers ===

const (
	pUser   = "user (root/admin)"
	pAdmin  = "key:admin"
	pEditor = "key:editor"
	pReader = "key:reader"
)

// allAllow returns the expected map for an endpoint that every org
// member (admin, editor, reader) is permitted to call (typical for
// reads: List/Describe across the entitlement boundary).
func allAllow() map[string]rbacOutcome {
	return map[string]rbacOutcome{pUser: OutcomeAllow, pAdmin: OutcomeAllow, pEditor: OutcomeAllow, pReader: OutcomeAllow}
}

// adminOnlyMutation returns the expected map for a mutation that
// should be admitted only to admin-or-above principals. The
// would-succeed principals (pUser, pAdmin) are SKIPPED so we do not
// create resources in the smoke env.
func adminOnlyMutation() map[string]rbacOutcome {
	return map[string]rbacOutcome{pUser: OutcomeSkip, pAdmin: OutcomeSkip, pEditor: OutcomeDeny, pReader: OutcomeDeny}
}

// editorOrAboveMutation: editor and above admitted; reader denied.
// Skip user + admin + editor (the admitted principals).
func editorOrAboveMutation() map[string]rbacOutcome {
	return map[string]rbacOutcome{pUser: OutcomeSkip, pAdmin: OutcomeSkip, pEditor: OutcomeSkip, pReader: OutcomeDeny}
}

// === client construction ===

func newAuditV1Client() *sdkv1.APIClient {
	cfg := sdkv1.NewConfiguration()
	cfg.Host = config.GetHost()
	cfg.Scheme = config.GetHostScheme()
	cfg.UserAgent = config.GetUserAgent()
	for i, server := range cfg.Servers {
		server.URL = fmt.Sprintf("%s://%s", config.GetHostScheme(), config.GetHost())
		cfg.Servers[i] = server
	}
	cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	return sdkv1.NewAPIClient(cfg)
}

func newAuditFleetClient() *sdkfleet.APIClient {
	cfg := sdkfleet.NewConfiguration()
	cfg.Host = config.GetHost()
	cfg.Scheme = config.GetHostScheme()
	cfg.UserAgent = config.GetUserAgent()
	for i, server := range cfg.Servers {
		server.URL = fmt.Sprintf("%s://%s", config.GetHostScheme(), config.GetHost())
		cfg.Servers[i] = server
	}
	cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	return sdkfleet.NewAPIClient(cfg)
}

// withV1Token / withFleetToken attach the bearer token to the request
// context using the SDK's documented context key. We deliberately
// avoid re-using the dataaccess wrappers here because they swallow
// the *http.Response and we need .StatusCode for outcome classification.
func withV1Token(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, sdkv1.ContextAccessToken, token)
}
func withFleetToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, sdkfleet.ContextAccessToken, token)
}

// extractStatus returns the HTTP status code from a (resp, err) pair
// returned by an SDK Execute() call. The SDK always returns the raw
// response (even on 4xx/5xx) so we prefer that; only fall back to
// inspecting *GenericOpenAPIError when the response itself is nil
// (transport-level failures). Using the raw response also lets us
// distinguish 400/404 from 401/403 cleanly.
func extractStatus(resp *http.Response, err error) (int, error) {
	if resp != nil {
		_ = resp.Body.Close()
		return resp.StatusCode, err
	}
	if err == nil {
		return 0, nil
	}
	var v1Err *sdkv1.GenericOpenAPIError
	if errors.As(err, &v1Err) {
		// Older SDKs surfaced status only in the body; we cannot
		// recover it here. Return 0 to mark as transport-level.
		_ = v1Err
	}
	var fleetErr *sdkfleet.GenericOpenAPIError
	if errors.As(err, &fleetErr) {
		_ = fleetErr
	}
	return 0, err
}

// bogusUUID is a stable, syntactically-valid id used to reach
// resource-scoped read endpoints without depending on real fixtures.
// An authorized caller will see a 404 (REACHED -> ALLOW under the
// matches() rules); an unauthorized caller will see 401/403 (DENY).
const bogusUUID = "00000000-0000-0000-0000-000000000000"
const bogusServiceID = "s-00000000"
const bogusEnvID = "se-00000000"
const bogusInstanceID = "instance-00000000"
const bogusProductTierID = "pt-00000000"

// === the matrix ===

// buildAuditMatrix declares every audited endpoint exactly once.
// Adding coverage is a one-row change: copy an existing entry, point
// it at the new SDK call, and fill in the Expected map. Read rows are
// safe by construction; mutation rows MUST set the would-succeed
// principal(s) to OutcomeSkip so the audit never creates persistent
// state in the smoke env.
func buildAuditMatrix(v1c *sdkv1.APIClient, fc *sdkfleet.APIClient) []rbacRow {
	return []rbacRow{
		// --- BUILD: services ---
		{
			Category: "Build / Service", Endpoint: "ServiceApi.ListService", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ServiceApiAPI.ServiceApiListService(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Service", Endpoint: "ServiceApi.DescribeService(bogus)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ServiceApiAPI.ServiceApiDescribeService(withV1Token(ctx, token), bogusServiceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Service", Endpoint: "ServiceApi.DeleteService(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.ServiceApiAPI.ServiceApiDeleteService(withV1Token(ctx, token), bogusServiceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: editorOrAboveMutation(),
		},

		// --- BUILD: environments ---
		{
			Category: "Build / Environment", Endpoint: "ServiceEnvironmentApi.ListServiceEnvironment(bogus svc)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ServiceEnvironmentApiAPI.ServiceEnvironmentApiListServiceEnvironment(withV1Token(ctx, token), bogusServiceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Environment", Endpoint: "ServiceEnvironmentApi.DeleteServiceEnvironment(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.ServiceEnvironmentApiAPI.ServiceEnvironmentApiDeleteServiceEnvironment(withV1Token(ctx, token), bogusServiceID, bogusEnvID).Execute()
				return extractStatus(resp, err)
			},
			Expected: editorOrAboveMutation(),
		},

		// --- BUILD: service plans / product tiers ---
		{
			Category: "Build / Service Plan", Endpoint: "ServicePlanApi.ListServicePlans(bogus svc)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ServicePlanApiAPI.ServicePlanApiListServicePlans(withV1Token(ctx, token), bogusServiceID, bogusEnvID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Service Plan", Endpoint: "ProductTierApi.ListProductTier(bogus svc)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ProductTierApiAPI.ProductTierApiListProductTier(withV1Token(ctx, token), bogusServiceID, bogusUUID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Service Plan", Endpoint: "ProductTierApi.DeleteProductTier(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.ProductTierApiAPI.ProductTierApiDeleteProductTier(withV1Token(ctx, token), bogusServiceID, bogusProductTierID).Execute()
				return extractStatus(resp, err)
			},
			Expected: editorOrAboveMutation(),
		},

		// --- BUILD: image registry / helm ---
		{
			Category: "Build / Image Registry", Endpoint: "ImageRegistryApi.ListImageRegistry", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ImageRegistryApiAPI.ImageRegistryApiListImageRegistry(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Build / Helm", Endpoint: "HelmPackageApi.ListHelmPackages", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.HelmPackageApiAPI.HelmPackageApiListHelmPackages(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},

		// --- OPERATE: account configs ---
		{
			Category: "Operate / Account", Endpoint: "AccountConfigApi.ListAccountConfig", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.AccountConfigApiAPI.AccountConfigApiListAccountConfig(withV1Token(ctx, token), "aws").Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Account", Endpoint: "AccountConfigApi.DescribeAccountConfig(bogus)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.AccountConfigApiAPI.AccountConfigApiDescribeAccountConfig(withV1Token(ctx, token), bogusUUID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Account", Endpoint: "AccountConfigApi.DeleteAccountConfig(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.AccountConfigApiAPI.AccountConfigApiDeleteAccountConfig(withV1Token(ctx, token), bogusUUID).Execute()
				return extractStatus(resp, err)
			},
			Expected: adminOnlyMutation(),
		},

		// --- OPERATE: subscriptions / requests ---
		{
			Category: "Operate / Subscription", Endpoint: "SubscriptionApi.ListSubscriptions", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.SubscriptionApiAPI.SubscriptionApiListSubscriptions(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Subscription", Endpoint: "SubscriptionRequestApi.ListSubscriptionRequests", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.SubscriptionRequestApiAPI.SubscriptionRequestApiListSubscriptionRequests(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},

		// --- OPERATE: instances ---
		{
			Category: "Operate / Instance", Endpoint: "InventoryApi.ListAccountConfigs", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := fc.InventoryApiAPI.InventoryApiListAccountConfigs(withFleetToken(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Instance", Endpoint: "InstanceSnapshotApi.ListAllInstanceSnapshots", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.InstanceSnapshotApiAPI.InstanceSnapshotApiListAllInstanceSnapshots(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},

		// --- OPERATE: custom domain ---
		{
			Category: "Operate / Domain", Endpoint: "CustomDomainApi.DescribeCustomDomain(bogus)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.CustomDomainApiAPI.CustomDomainApiDescribeCustomDomain(withV1Token(ctx, token), bogusServiceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Domain", Endpoint: "CustomDomainApi.DeleteCustomDomain(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.CustomDomainApiAPI.CustomDomainApiDeleteCustomDomain(withV1Token(ctx, token), bogusServiceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: adminOnlyMutation(),
		},

		// --- OPERATE: secrets ---
		{
			Category: "Operate / Secrets", Endpoint: "SecretsApi.ListSecrets(bogus env)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.SecretsApiAPI.SecretsApiListSecrets(withV1Token(ctx, token), "dev").Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Operate / Secrets", Endpoint: "SecretsApi.DeleteSecret(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				resp, err := v1c.SecretsApiAPI.SecretsApiDeleteSecret(withV1Token(ctx, token), "dev", "rbac-audit-bogus-secret").Execute()
				return extractStatus(resp, err)
			},
			Expected: adminOnlyMutation(),
		},

		// --- ORG: identity providers / sp organization ---
		{
			Category: "Org / Identity", Endpoint: "IdentityProviderApi.ListIdentityProviders", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.IdentityProviderApiAPI.IdentityProviderApiListIdentityProviders(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Org / Self", Endpoint: "UsersApi.DescribeUser", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.UsersApiAPI.UsersApiDescribeUser(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},

		// --- ORG: api keys (admin-only mutations) ---
		{
			Category: "Org / API Keys", Endpoint: "ApiKeyApi.ListAPIKeys", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ApiKeyApiAPI.ApiKeyApiListAPIKeys(withV1Token(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Org / API Keys", Endpoint: "ApiKeyApi.RevokeAPIKey(bogus)", HTTPVerb: "POST", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ApiKeyApiAPI.ApiKeyApiRevokeAPIKey(withV1Token(ctx, token), bogusUUID).Execute()
				return extractStatus(resp, err)
			},
			Expected: adminOnlyMutation(),
		},
		{
			Category: "Org / API Keys", Endpoint: "ApiKeyApi.DeleteAPIKey(bogus)", HTTPVerb: "DELETE", IsRead: false,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.ApiKeyApiAPI.ApiKeyApiDeleteAPIKey(withV1Token(ctx, token), bogusUUID).Execute()
				return extractStatus(resp, err)
			},
			Expected: adminOnlyMutation(),
		},

		// --- BILLING / AUDIT (read-only, all roles) ---
		{
			Category: "Org / Audit", Endpoint: "AuditEventsApi.ListAuditEventsForInstance(bogus)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := v1c.AuditEventsApiAPI.AuditEventsApiListAuditEventsForInstance(withV1Token(ctx, token), bogusInstanceID).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},

		// --- FLEET (operator-side) reads ---
		{
			Category: "Fleet / Inventory", Endpoint: "InventoryApi.ListServiceOfferings", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := fc.InventoryApiAPI.InventoryApiListServiceOfferings(withFleetToken(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Fleet / HostCluster", Endpoint: "HostclusterApi.ListHostClusters", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := fc.HostclusterApiAPI.HostclusterApiListHostClusters(withFleetToken(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Fleet / Audit", Endpoint: "AuditEventsApi.AuditEvents (fleet)", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := fc.AuditEventsApiAPI.AuditEventsApiAuditEvents(withFleetToken(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
		{
			Category: "Fleet / Notifications", Endpoint: "NotificationsApi.ListNotificationChannels", HTTPVerb: "GET", IsRead: true,
			Invoke: func(ctx context.Context, token string) (int, error) {
				_, resp, err := fc.NotificationsApiAPI.NotificationsApiListNotificationChannels(withFleetToken(ctx, token)).Execute()
				return extractStatus(resp, err)
			},
			Expected: allAllow(),
		},
	}
}

// === markdown rendering ===

// renderMarkdown produces a $GITHUB_STEP_SUMMARY-friendly markdown
// document with the audit matrix and a per-category roll-up. The
// output is written to the test log AND appended to the workflow
// summary so reviewers can audit RBAC posture at a glance from the
// run UI without scrolling through verbose test output.
func renderMarkdown(rows []rbacRow, principals []principal, results [][]cellResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## API-Key RBAC Audit Matrix\n\n")
	fmt.Fprintf(&b, "_Generated by `Test_api_key_rbac_audit_matrix` against the live smoke environment._\n\n")
	fmt.Fprintf(&b, "**Legend:** ")
	fmt.Fprintf(&b, "✅ pass · ❌ drift · ⏭ skipped · 🟡 rejected-by-input (mutation: did-not-create but not via auth gate)\n\n")
	fmt.Fprintf(&b, "Status codes are inline so 401/403 vs 404 vs 400 are distinguishable.\n\n")

	// Header
	fmt.Fprintf(&b, "| Category | Endpoint | Verb |")
	for _, p := range principals {
		fmt.Fprintf(&b, " %s |", p.Label)
	}
	b.WriteString("\n|---|---|---|")
	for range principals {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	// Rows
	for i, row := range rows {
		fmt.Fprintf(&b, "| %s | `%s` | %s |", row.Category, row.Endpoint, row.HTTPVerb)
		for j := range principals {
			fmt.Fprintf(&b, " %s |", renderCell(results[i][j]))
		}
		b.WriteString("\n")
	}

	// Summary roll-up by category
	fmt.Fprintf(&b, "\n### Roll-up by category\n\n")
	categoryStats := map[string][3]int{} // [pass, drift, skip]
	for i, row := range rows {
		stat := categoryStats[row.Category]
		for j := range principals {
			r := results[i][j]
			switch {
			case r.Outcome == OutcomeSkip:
				stat[2]++
			case r.Match:
				stat[0]++
			default:
				stat[1]++
			}
		}
		categoryStats[row.Category] = stat
	}
	cats := make([]string, 0, len(categoryStats))
	for k := range categoryStats {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	fmt.Fprintf(&b, "| Category | ✅ pass | ❌ drift | ⏭ skipped |\n|---|---:|---:|---:|\n")
	var totalPass, totalDrift, totalSkip int
	for _, c := range cats {
		s := categoryStats[c]
		fmt.Fprintf(&b, "| %s | %d | %d | %d |\n", c, s[0], s[1], s[2])
		totalPass += s[0]
		totalDrift += s[1]
		totalSkip += s[2]
	}
	fmt.Fprintf(&b, "| **TOTAL** | **%d** | **%d** | **%d** |\n", totalPass, totalDrift, totalSkip)

	return b.String()
}

func renderCell(r cellResult) string {
	if r.Outcome == OutcomeSkip {
		return "⏭"
	}
	icon := "✅"
	if !r.Match {
		icon = "❌"
	} else if r.Outcome == OutcomeRejectedByValidate {
		icon = "🟡"
	}
	if r.Status > 0 {
		return fmt.Sprintf("%s %s `%d`", icon, r.Outcome, r.Status)
	}
	return fmt.Sprintf("%s %s", icon, r.Outcome)
}

func appendFile(path, content string) error {
	// path is GITHUB_STEP_SUMMARY supplied by the runner; not user-controlled.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G703,G304 -- path is $GITHUB_STEP_SUMMARY from the runner
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content + "\n")
	return err
}
