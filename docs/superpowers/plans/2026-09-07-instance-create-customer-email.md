# `instance create --customer-email` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let `omnistrate-ctl instance create` accept `--customer-email` and resolve (or create) the customer's subscription for the target service plan, instead of requiring a hand-copied `--subscription-id`.

**Architecture:** The resolution lives in the shared `internal/dataaccess` lookup that `instance adopt` and `account customer create` already call, so all three commands gain the same behaviour. That lookup is rewritten to paginate, to fall back to the production-only org-scoped email form (`<local>+<orgID>@<domain>`), and to reject non-ACTIVE subscriptions with an actionable error. `cmd/instance/create.go` mirrors the flag handling of `account customer create`.

**Tech Stack:** Go 1.26, cobra, `omnistrate-sdk-go` (fleet + v1 clients), testify, `net/http/httptest` for dataaccess tests.

**Spec:** `docs/superpowers/specs/2026-09-07-instance-create-customer-email-design.md`

## Global Constraints

- **No identifying data anywhere.** Code, comments, test fixtures, docs, and commit/PR text use placeholders only: `customer@example.com`, `provider@example.com`, `org-abc123`, `s-test`, `se-test`, `pt-test`, `sub-test`. Never a real customer, organization, person, or domain.
- **Org scoping is production-only.** The `<local>+<orgID>@<domain>` fallback must never run outside a production environment type.
- **Subscription status vocabulary:** `ACTIVE`, `SUSPENDED`, `CANCELLED`, `TERMINATED`. Only `ACTIVE` may back a new instance.
- **Do not enable `IncludeInactive` on the subscription listing.** It switches the server to an unscoped query that returns soft-deleted (terminated) rows, which would block legitimate re-subscription.
- `account customer create` (`cmd/account/customer_create.go`) is the reference for flag naming, validation, and call ordering. Do not replicate its PROD "default to the calling user's own email" behaviour (`customer_create.go:610`) in `instance create`.
- Run `make unit-test` before every commit. Commit only with tests passing.

## Before you start

The working tree may contain an untracked `internal/dataaccess/k8sclient_test.go` that does not
compile (it references `restConfigForHostClusterKubeConfig` and `k8sRequestTimeout`, which do not
exist). It is unrelated to this work, but it fails the build of the whole `internal/dataaccess`
test binary, so every `go test ./internal/dataaccess/...` in this plan will fail until it is dealt
with. Check first:

```bash
git status --short internal/dataaccess/k8sclient_test.go
go vet ./internal/dataaccess/
```

If it is present and broken, move it out of the package for the duration of this work and put it
back afterwards. Do not delete it and do not commit it — it is someone's work in progress.

All fixture payloads used in this plan were verified to decode against the SDK models
(`FleetListSubscriptionsResult`, `FleetListAllUsersResult`, `FleetDescribeSubscriptionResult`,
`FleetCreateSubscriptionOnBehalfOfCustomerResult`, v1 `DescribeUserResult`), so a decode failure in
these tests means the fixture was altered, not that a required field is missing.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `internal/utils/scopedemail.go` (create) | Pure org-scoped email helpers + production environment-type predicate. No API calls. |
| `internal/utils/scopedemail_test.go` (create) | Unit tests for the above. |
| `internal/dataaccess/subscription.go` (modify) | Candidate matching, status gate, the rewritten lookup, and the extracted user-ID resolver. |
| `internal/dataaccess/subscription_test.go` (create) | Pure-function tests plus `httptest`-backed lookup tests. |
| `cmd/account/customer_create.go` (modify) | Call the lookup with the new options struct; delegate the PROD predicate to `utils`. |
| `cmd/account/customer_create_test.go` (modify) | Update the injected stub signature. |
| `cmd/instance/create.go` (modify) | `--customer-email` flag, validation, and subscription resolution. |
| `cmd/instance/create_test.go` (modify) | Flag registration, validation, and resolution wiring tests. |
| `mkdocs/docs/` (regenerate) | Generated CLI reference. |

---

### Task 1: Org-scoped email helpers

Pure string helpers mirroring the server's `commons/pkg/utils/scopedorgutils.go`, plus the production predicate that gates when scoping applies. No dependencies on other tasks.

**Files:**
- Create: `internal/utils/scopedemail.go`
- Test: `internal/utils/scopedemail_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `utils.FormatEmailWithScopedOrg(email, orgID string) (string, error)`
  - `utils.EmailHasScopedOrg(email string) bool`
  - `utils.IsProductionEnvironmentType(environmentType string) bool`

- [ ] **Step 1: Write the failing tests**

Create `internal/utils/scopedemail_test.go`:

```go
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatEmailWithScopedOrg(t *testing.T) {
	scoped, err := FormatEmailWithScopedOrg("customer@example.com", "org-abc123")
	require.NoError(t, err)
	assert.Equal(t, "customer+org-abc123@example.com", scoped)
}

func TestFormatEmailWithScopedOrgPreservesExistingPlusTag(t *testing.T) {
	scoped, err := FormatEmailWithScopedOrg("customer+team@example.com", "org-abc123")
	require.NoError(t, err)
	assert.Equal(t, "customer+team+org-abc123@example.com", scoped)
}

func TestFormatEmailWithScopedOrgRejectsAlreadyScopedEmail(t *testing.T) {
	_, err := FormatEmailWithScopedOrg("customer+org-abc123@example.com", "org-abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already scoped")
}

func TestFormatEmailWithScopedOrgRejectsBadInput(t *testing.T) {
	for name, tc := range map[string]struct{ email, orgID string }{
		"no at sign":    {"customer.example.com", "org-abc123"},
		"two at signs":  {"customer@example@com", "org-abc123"},
		"empty org":     {"customer@example.com", ""},
		"org bad shape": {"customer@example.com", "abc123"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := FormatEmailWithScopedOrg(tc.email, tc.orgID)
			require.Error(t, err)
		})
	}
}

func TestEmailHasScopedOrg(t *testing.T) {
	assert.True(t, EmailHasScopedOrg("customer+org-abc123@example.com"))
	assert.True(t, EmailHasScopedOrg("customer+team+org-abc123@example.com"))
	assert.False(t, EmailHasScopedOrg("customer@example.com"))
	assert.False(t, EmailHasScopedOrg("customer+team@example.com"))
	assert.False(t, EmailHasScopedOrg("not-an-email"))
}

func TestIsProductionEnvironmentType(t *testing.T) {
	assert.True(t, IsProductionEnvironmentType("PROD"))
	assert.True(t, IsProductionEnvironmentType("production"))
	assert.True(t, IsProductionEnvironmentType("  Prod  "))
	assert.False(t, IsProductionEnvironmentType("DEV"))
	assert.False(t, IsProductionEnvironmentType(""))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/utils/ -run 'ScopedOrg|ProductionEnvironmentType' -v`
Expected: FAIL — `undefined: FormatEmailWithScopedOrg`.

- [ ] **Step 3: Write the implementation**

Create `internal/utils/scopedemail.go`:

```go
package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// scopedOrgIDRegex matches an organization ID. It mirrors the server-side definition so the
// CLI and the platform agree on what the trailing segment of a scoped address looks like.
var scopedOrgIDRegex = regexp.MustCompile(`^org-[a-zA-Z0-9][a-zA-Z0-9._\-]*$`)

// FormatEmailWithScopedOrg returns the org-scoped form of an address, which is how the
// platform stores an end customer's identity inside a service provider's organization:
//
//	("customer@example.com", "org-abc123") -> "customer+org-abc123@example.com"
//
// Any plus tag already present is preserved, with the organization ID appended after it.
func FormatEmailWithScopedOrg(email, orgID string) (string, error) {
	localPart, domain, err := splitEmail(email)
	if err != nil {
		return "", err
	}
	if !scopedOrgIDRegex.MatchString(orgID) {
		return "", fmt.Errorf("invalid organization ID %q", orgID)
	}
	if localPartHasScopedOrg(localPart) {
		return "", fmt.Errorf("email %q is already scoped to an organization", email)
	}

	return fmt.Sprintf("%s+%s@%s", localPart, orgID, domain), nil
}

// EmailHasScopedOrg reports whether an address already carries an organization ID as the
// last segment of its local part.
func EmailHasScopedOrg(email string) bool {
	localPart, _, err := splitEmail(email)
	if err != nil {
		return false
	}
	return localPartHasScopedOrg(localPart)
}

// IsProductionEnvironmentType reports whether an environment type is a production one.
// Org-scoped customer identities only exist in production environments.
func IsProductionEnvironmentType(environmentType string) bool {
	switch strings.ToUpper(strings.TrimSpace(environmentType)) {
	case "PROD", "PRODUCTION":
		return true
	default:
		return false
	}
}

func splitEmail(email string) (localPart, domain string, err error) {
	parts := strings.Split(strings.TrimSpace(email), "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid email address %q", email)
	}
	return parts[0], parts[1], nil
}

func localPartHasScopedOrg(localPart string) bool {
	segments := strings.Split(localPart, "+")
	if len(segments) < 2 {
		return false
	}
	return scopedOrgIDRegex.MatchString(segments[len(segments)-1])
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/utils/ -run 'ScopedOrg|ProductionEnvironmentType' -v`
Expected: PASS, all subtests included.

- [ ] **Step 5: Run the package suite and commit**

Run: `go test ./internal/utils/`
Expected: PASS.

```bash
git add internal/utils/scopedemail.go internal/utils/scopedemail_test.go
git commit -m "feat: add org-scoped email helpers

Adds the formatting and detection helpers for the <local>+<orgID>@<domain>
identity form, plus the production environment-type predicate that gates
where that form applies."
```

---

### Task 2: Candidate matching and the subscription status gate

Two pure functions in `internal/dataaccess`, with no network access, that decide which subscription a customer email resolves to and what error a non-usable one produces.

**Files:**
- Modify: `internal/dataaccess/subscription.go` (add functions; leave existing ones untouched in this task)
- Test: `internal/dataaccess/subscription_test.go` (create)

**Interfaces:**
- Consumes: nothing from Task 1.
- Produces:
  - `matchSubscriptionsByEmail(subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult, planID, email string) []openapiclientfleet.FleetDescribeSubscriptionResult`
  - `selectCustomerSubscription(candidates []openapiclientfleet.FleetDescribeSubscriptionResult, customerEmail string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error)` — returns `(nil, nil)` when there are no candidates, which tells the caller to create one.
  - `const subscriptionStatusActive = "ACTIVE"`

- [ ] **Step 1: Write the failing tests**

Create `internal/dataaccess/subscription_test.go`:

```go
package dataaccess

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSubscription(id, email, status string) openapiclientfleet.FleetDescribeSubscriptionResult {
	return openapiclientfleet.FleetDescribeSubscriptionResult{
		Id:              id,
		ProductTierId:   "pt-test",
		ProductTierName: "test-plan",
		RootUserEmail:   email,
		Status:          status,
	}
}

func TestMatchSubscriptionsByEmailIsCaseInsensitive(t *testing.T) {
	subscriptions := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "Customer@Example.com", "ACTIVE"),
	}

	matches := matchSubscriptionsByEmail(subscriptions, "pt-test", "customer@example.com")

	require.Len(t, matches, 1)
	assert.Equal(t, "sub-1", matches[0].Id)
}

func TestMatchSubscriptionsByEmailFiltersOtherPlans(t *testing.T) {
	other := testSubscription("sub-2", "customer@example.com", "ACTIVE")
	other.ProductTierId = "pt-other"
	subscriptions := []openapiclientfleet.FleetDescribeSubscriptionResult{other}

	matches := matchSubscriptionsByEmail(subscriptions, "pt-test", "customer@example.com")

	assert.Empty(t, matches)
}

func TestMatchSubscriptionsByEmailDoesNotMatchScopedFormImplicitly(t *testing.T) {
	subscriptions := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer+org-abc123@example.com", "ACTIVE"),
	}

	matches := matchSubscriptionsByEmail(subscriptions, "pt-test", "customer@example.com")

	assert.Empty(t, matches, "the scoped form must only match when the caller asks for it explicitly")
}

func TestSelectCustomerSubscriptionNoCandidatesSignalsCreate(t *testing.T) {
	subscription, err := selectCustomerSubscription(nil, "customer@example.com")

	require.NoError(t, err)
	assert.Nil(t, subscription)
}

func TestSelectCustomerSubscriptionReturnsSingleActive(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "ACTIVE"),
	}

	subscription, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-1", subscription.Id)
}

func TestSelectCustomerSubscriptionIgnoresNonActiveWhenAnActiveExists(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "SUSPENDED"),
		testSubscription("sub-2", "customer@example.com", "ACTIVE"),
	}

	subscription, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-2", subscription.Id)
}

func TestSelectCustomerSubscriptionRejectsMultipleActive(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "ACTIVE"),
		testSubscription("sub-2", "customer@example.com", "ACTIVE"),
	}

	_, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub-1")
	assert.Contains(t, err.Error(), "sub-2")
	assert.Contains(t, err.Error(), "--subscription-id")
}

func TestSelectCustomerSubscriptionReportsSuspended(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "SUSPENDED"),
	}

	_, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub-1")
	assert.Contains(t, err.Error(), "SUSPENDED")
	assert.Contains(t, err.Error(), "omnistrate-ctl subscription resume sub-1")
}

func TestSelectCustomerSubscriptionReportsOtherNonActiveStatus(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "CANCELLED"),
	}

	_, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CANCELLED")
	assert.Contains(t, err.Error(), "expected ACTIVE")
	assert.NotContains(t, err.Error(), "subscription resume")
}

func TestSelectCustomerSubscriptionReportsEveryNonActiveCandidate(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "SUSPENDED"),
		testSubscription("sub-2", "customer@example.com", "CANCELLED"),
	}

	_, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub-1 (SUSPENDED)")
	assert.Contains(t, err.Error(), "sub-2 (CANCELLED)")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/dataaccess/ -run 'MatchSubscriptionsByEmail|SelectCustomerSubscription' -v`
Expected: FAIL — `undefined: matchSubscriptionsByEmail`.

- [ ] **Step 3: Write the implementation**

Append to `internal/dataaccess/subscription.go`:

```go
// subscriptionStatusActive is the only subscription status that may back a new instance.
// The platform's full vocabulary is ACTIVE, SUSPENDED, CANCELLED and TERMINATED.
const subscriptionStatusActive = "ACTIVE"

// matchSubscriptionsByEmail returns the subscriptions on a plan whose root user is the
// given address. The plan filter is redundant with the server-side query parameter and is
// kept so the matching is correct on its own terms.
func matchSubscriptionsByEmail(
	subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult,
	planID string,
	email string,
) []openapiclientfleet.FleetDescribeSubscriptionResult {
	matches := make([]openapiclientfleet.FleetDescribeSubscriptionResult, 0, 1)
	for _, subscription := range subscriptions {
		if planID != "" && !strings.EqualFold(subscription.ProductTierId, planID) {
			continue
		}
		if !strings.EqualFold(subscription.RootUserEmail, email) {
			continue
		}
		matches = append(matches, subscription)
	}
	return matches
}

// selectCustomerSubscription picks the one subscription that may back a new instance.
// It returns (nil, nil) when there is nothing to choose from, which tells the caller to
// create a subscription instead.
func selectCustomerSubscription(
	candidates []openapiclientfleet.FleetDescribeSubscriptionResult,
	customerEmail string,
) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	active := make([]openapiclientfleet.FleetDescribeSubscriptionResult, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Status, subscriptionStatusActive) {
			active = append(active, candidate)
		}
	}

	switch len(active) {
	case 1:
		selected := active[0]
		return &selected, nil
	case 0:
		return nil, unusableSubscriptionError(candidates, customerEmail)
	default:
		ids := make([]string, 0, len(active))
		for _, candidate := range active {
			ids = append(ids, candidate.Id)
		}
		return nil, errors.Errorf(
			"found %d active subscriptions for customer %s on plan %s (%s). Pass --subscription-id to choose one",
			len(active),
			customerEmail,
			active[0].ProductTierName,
			strings.Join(ids, ", "),
		)
	}
}

// unusableSubscriptionError explains why the subscriptions a customer does own cannot back
// a new instance, and names the remedy when there is one.
func unusableSubscriptionError(
	candidates []openapiclientfleet.FleetDescribeSubscriptionResult,
	customerEmail string,
) error {
	if len(candidates) == 1 {
		candidate := candidates[0]
		if strings.EqualFold(candidate.Status, "SUSPENDED") {
			return errors.Errorf(
				"subscription %s for customer %s on plan %s is SUSPENDED and cannot be used to create an instance. "+
					"Resume it with 'omnistrate-ctl subscription resume %s', or pass --subscription-id to use a different subscription",
				candidate.Id,
				customerEmail,
				candidate.ProductTierName,
				candidate.Id,
			)
		}
		return errors.Errorf(
			"subscription %s for customer %s on plan %s is in status %s, expected ACTIVE, so it cannot be used to create an instance. "+
				"Pass --subscription-id to use a different subscription",
			candidate.Id,
			customerEmail,
			candidate.ProductTierName,
			candidate.Status,
		)
	}

	described := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		described = append(described, fmt.Sprintf("%s (%s)", candidate.Id, candidate.Status))
	}
	return errors.Errorf(
		"customer %s has no active subscription on plan %s; found %s. "+
			"Resume a suspended subscription with 'omnistrate-ctl subscription resume', or pass --subscription-id",
		customerEmail,
		candidates[0].ProductTierName,
		strings.Join(described, ", "),
	)
}
```

Add `"fmt"` to the import block of `internal/dataaccess/subscription.go`; `strings` and `github.com/pkg/errors` are already imported.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/dataaccess/ -run 'MatchSubscriptionsByEmail|SelectCustomerSubscription' -v`
Expected: PASS (10 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/dataaccess/subscription.go internal/dataaccess/subscription_test.go
git commit -m "feat: add subscription candidate matching and status gate

Adds the pure matching and selection helpers that decide which
subscription a customer email resolves to, and produce an actionable
error when the only matches are suspended or otherwise not active."
```

---

### Task 3: Rewrite the customer-email lookup

Replace the positional lookup with an options struct carrying the environment type, add pagination and the production-only scoped fallback, and apply the Task 2 status gate. The signature change forces the two call sites and their stubs to move in the same commit.

**Files:**
- Modify: `internal/dataaccess/subscription.go:37-119`
- Modify: `cmd/account/customer_create.go:626-646` (the `resolveCustomerSubscriptionByEmail` call) and `cmd/account/customer_create.go:651-658` (`isProductionEnvironmentType`)
- Modify: `cmd/account/customer_create_test.go:362`, `:406`, `:439` (stub signatures)
- Test: `internal/dataaccess/subscription_test.go` (append)

**Interfaces:**
- Consumes: `utils.FormatEmailWithScopedOrg`, `utils.EmailHasScopedOrg`, `utils.IsProductionEnvironmentType` (Task 1); `matchSubscriptionsByEmail`, `selectCustomerSubscription` (Task 2).
- Produces:
  - `dataaccess.CustomerSubscriptionLookup` struct with fields `ServiceID`, `EnvironmentID`, `EnvironmentType`, `PlanID`, `CustomerEmail` (all `string`).
  - `dataaccess.GetSubscriptionByCustomerEmailInEnvironment(ctx context.Context, token string, lookup CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error)`
  - `dataaccess.GetSubscriptionByCustomerEmail(ctx context.Context, token, serviceID, planID, customerEmail string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error)` — signature unchanged.

- [ ] **Step 1: Write the failing tests**

Append to `internal/dataaccess/subscription_test.go`. Add these imports to the file's import block: `"context"`, `"encoding/json"`, `"net/http"`, `"net/http/httptest"`, `"net/url"`, `"strings"`.

```go
// subscriptionJSON builds a fleet subscription payload with every field the SDK requires.
func subscriptionJSON(id, email, status string) map[string]any {
	return map[string]any{
		"createdAt":         "2026-09-07T00:00:00Z",
		"id":                id,
		"instanceCount":     0,
		"productTierId":     "pt-test",
		"productTierName":   "test-plan",
		"rootUserEmail":     email,
		"rootUserId":        "user-test",
		"rootUserName":      "test-customer",
		"serviceId":         "s-test",
		"serviceName":       "test-service",
		"status":            status,
		"updatedAt":         "2026-09-07T00:00:00Z",
		"updatedByUserId":   "user-test",
		"updatedByUserName": "test-customer",
	}
}

// startSubscriptionTestServer points the SDK clients at a stub server for the duration of
// the test and returns it.
func startSubscriptionTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	return server
}

func TestGetSubscriptionByCustomerEmailInEnvironmentPaginates(t *testing.T) {
	var listCalls int
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription", r.URL.Path)
		require.Equal(t, "pt-test", r.URL.Query().Get("productTierId"))
		listCalls++

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("nextPageToken") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ids":           []string{"sub-1"},
				"subscriptions": []map[string]any{subscriptionJSON("sub-1", "other@example.com", "ACTIVE")},
				"nextPageToken": "page-2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ids":           []string{"sub-2"},
			"subscriptions": []map[string]any{subscriptionJSON("sub-2", "customer@example.com", "ACTIVE")},
		})
	})

	subscription, err := GetSubscriptionByCustomerEmailInEnvironment(context.Background(), "test-token", CustomerSubscriptionLookup{
		ServiceID:       "s-test",
		EnvironmentID:   "se-test",
		EnvironmentType: "DEV",
		PlanID:          "pt-test",
		CustomerEmail:   "customer@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-2", subscription.Id)
	assert.Equal(t, 2, listCalls, "the lookup must follow nextPageToken")
}

func TestGetSubscriptionByCustomerEmailInEnvironmentFallsBackToScopedEmailInProd(t *testing.T) {
	var describeUserCalls int
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/2022-09-01-00/user":
			describeUserCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "user-provider", "orgId": "org-abc123"})
		case "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ids": []string{"sub-1"},
				"subscriptions": []map[string]any{
					subscriptionJSON("sub-1", "customer+org-abc123@example.com", "ACTIVE"),
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	subscription, err := GetSubscriptionByCustomerEmailInEnvironment(context.Background(), "test-token", CustomerSubscriptionLookup{
		ServiceID:       "s-test",
		EnvironmentID:   "se-test",
		EnvironmentType: "PROD",
		PlanID:          "pt-test",
		CustomerEmail:   "customer@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-1", subscription.Id)
	assert.Equal(t, 1, describeUserCalls)
}

func TestGetSubscriptionByCustomerEmailInEnvironmentSkipsScopedFallbackOutsideProd(t *testing.T) {
	var createCalls int
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/2022-09-01-00/user":
			t.Fatal("describe user must not be called outside production")
		case r.URL.Path == "/2022-09-01-00/fleet/users":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"users": []map[string]any{{"userId": "user-test", "email": "customer@example.com"}},
			})
		case r.URL.Path == "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription" && r.Method == http.MethodPost:
			createCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sub-new"})
		case r.URL.Path == "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ids":           []string{"sub-1"},
				"subscriptions": []map[string]any{subscriptionJSON("sub-1", "customer+org-abc123@example.com", "ACTIVE")},
			})
		case r.URL.Path == "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription/sub-new":
			_ = json.NewEncoder(w).Encode(subscriptionJSON("sub-new", "customer@example.com", "ACTIVE"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	subscription, err := GetSubscriptionByCustomerEmailInEnvironment(context.Background(), "test-token", CustomerSubscriptionLookup{
		ServiceID:       "s-test",
		EnvironmentID:   "se-test",
		EnvironmentType: "DEV",
		PlanID:          "pt-test",
		CustomerEmail:   "customer@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-new", subscription.Id)
	assert.Equal(t, 1, createCalls)
}

func TestGetSubscriptionByCustomerEmailInEnvironmentReportsSuspended(t *testing.T) {
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			t.Fatal("a suspended subscription must not trigger a create")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ids":           []string{"sub-1"},
			"subscriptions": []map[string]any{subscriptionJSON("sub-1", "customer@example.com", "SUSPENDED")},
		})
	})

	_, err := GetSubscriptionByCustomerEmailInEnvironment(context.Background(), "test-token", CustomerSubscriptionLookup{
		ServiceID:       "s-test",
		EnvironmentID:   "se-test",
		EnvironmentType: "DEV",
		PlanID:          "pt-test",
		CustomerEmail:   "customer@example.com",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SUSPENDED")
	assert.Contains(t, err.Error(), "omnistrate-ctl subscription resume sub-1")
}

func TestGetSubscriptionByCustomerEmailInEnvironmentRequiresEmail(t *testing.T) {
	_, err := GetSubscriptionByCustomerEmailInEnvironment(context.Background(), "test-token", CustomerSubscriptionLookup{
		ServiceID:     "s-test",
		EnvironmentID: "se-test",
		PlanID:        "pt-test",
		CustomerEmail: "   ",
	})

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "customer email is required")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/dataaccess/ -run GetSubscriptionByCustomerEmailInEnvironment -v`
Expected: FAIL — `undefined: CustomerSubscriptionLookup`.

- [ ] **Step 3: Replace the lookup implementation**

In `internal/dataaccess/subscription.go`, replace the whole of `GetSubscriptionByCustomerEmail` and `GetSubscriptionByCustomerEmailInEnvironment` (currently lines 37-119) with:

```go
// CustomerSubscriptionLookup identifies the customer and the plan a subscription is being
// resolved for. EnvironmentType decides whether the org-scoped email fallback applies,
// because org-scoped customer identities only exist in production environments.
type CustomerSubscriptionLookup struct {
	ServiceID       string
	EnvironmentID   string
	EnvironmentType string
	PlanID          string
	CustomerEmail   string
}

func GetSubscriptionByCustomerEmail(ctx context.Context, token string, serviceID string, planID string, customerEmail string) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	// Describe the service offering for this service and plan (product tier) ID to get the environment ID
	serviceOfferingResult, err := DescribeServiceOffering(ctx, token, serviceID, planID, "")
	if err != nil {
		return nil, err
	}

	for _, offering := range serviceOfferingResult.ConsumptionDescribeServiceOfferingResult.Offerings {
		if offering.ProductTierID == planID {
			return GetSubscriptionByCustomerEmailInEnvironment(ctx, token, CustomerSubscriptionLookup{
				ServiceID:       serviceID,
				EnvironmentID:   offering.ServiceEnvironmentID,
				EnvironmentType: offering.ServiceEnvironmentType,
				PlanID:          planID,
				CustomerEmail:   customerEmail,
			})
		}
	}

	err = errors.New("no subscription found for the given customer email or the plan does not exist")
	return
}

// GetSubscriptionByCustomerEmailInEnvironment resolves the subscription a customer owns for
// a plan, creating one when the customer has none.
//
// The listing deliberately leaves IncludeInactive off. Terminating a subscription
// soft-deletes its row, and only the inactive listing returns soft-deleted rows, so a
// terminated subscription is correctly invisible here and the customer can be resubscribed.
// A suspended subscription stays live and is reported rather than silently duplicated.
func GetSubscriptionByCustomerEmailInEnvironment(
	ctx context.Context,
	token string,
	lookup CustomerSubscriptionLookup,
) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	customerEmail := strings.TrimSpace(lookup.CustomerEmail)
	if customerEmail == "" {
		return nil, errors.New("customer email is required to look up a subscription")
	}

	subscriptions, err := ListAllSubscriptions(ctx, token, lookup.ServiceID, lookup.EnvironmentID, &ListSubscriptionsOptions{
		ProductTierId: &lookup.PlanID,
	})
	if err != nil {
		return nil, err
	}

	candidates := matchSubscriptionsByEmail(subscriptions, lookup.PlanID, customerEmail)

	// In production the platform stores an end customer's identity scoped to the service
	// provider's organization, so a customer with no match under the plain address may still
	// own a subscription under <local>+<orgID>@<domain>.
	if len(candidates) == 0 &&
		utils.IsProductionEnvironmentType(lookup.EnvironmentType) &&
		!utils.EmailHasScopedOrg(customerEmail) {
		var scopedEmail string
		if scopedEmail, err = scopedCustomerEmail(ctx, token, customerEmail); err != nil {
			return nil, err
		}
		candidates = matchSubscriptionsByEmail(subscriptions, lookup.PlanID, scopedEmail)
	}

	selected, err := selectCustomerSubscription(candidates, customerEmail)
	if err != nil {
		return nil, err
	}
	if selected != nil {
		return selected, nil
	}

	createResp, err := CreateSubscriptionOnBehalf(ctx, token, lookup.ServiceID, lookup.EnvironmentID, &CreateSubscriptionOnBehalfOptions{
		ProductTierID:           lookup.PlanID,
		OnBehalfOfCustomerEmail: customerEmail,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create subscription for user %s", customerEmail)
	}

	subscriptionID := strings.TrimSpace(createResp.GetId())
	if subscriptionID == "" {
		return nil, errors.Errorf("subscription creation for user %s returned an empty subscription ID", customerEmail)
	}

	resp, err = DescribeSubscription(ctx, token, lookup.ServiceID, lookup.EnvironmentID, subscriptionID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to describe newly created subscription for user %s", customerEmail)
	}

	// A newly created subscription goes through the same status gate, so every "unusable
	// subscription" message in this command comes from one place.
	if _, err = selectCustomerSubscription([]openapiclientfleet.FleetDescribeSubscriptionResult{*resp}, customerEmail); err != nil {
		return nil, err
	}

	return resp, nil
}

// scopedCustomerEmail builds the org-scoped form of a customer address using the calling
// service provider's organization ID.
func scopedCustomerEmail(ctx context.Context, token, customerEmail string) (string, error) {
	user, err := DescribeUser(ctx, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to describe the current user to resolve the provider organization ID")
	}
	if user == nil || user.OrgId == nil || strings.TrimSpace(*user.OrgId) == "" {
		return "", errors.New("describe user returned an empty organization ID; cannot check for an org-scoped customer subscription")
	}

	scopedEmail, err := utils.FormatEmailWithScopedOrg(customerEmail, strings.TrimSpace(*user.OrgId))
	if err != nil {
		return "", errors.Wrapf(err, "failed to build the org-scoped form of %s", customerEmail)
	}
	return scopedEmail, nil
}
```

Add `"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"` to the import block.

- [ ] **Step 4: Update the `account customer create` call site**

In `cmd/account/customer_create.go`, replace the `resolveCustomerSubscriptionByEmail` call (currently lines 626-633):

```go
	subscription, err := resolveCustomerSubscriptionByEmail(
		ctx,
		token,
		dataaccess.CustomerSubscriptionLookup{
			ServiceID:       target.ServiceID,
			EnvironmentID:   target.EnvironmentID,
			EnvironmentType: target.EnvironmentType,
			PlanID:          target.ProductTierID,
			CustomerEmail:   customerEmail,
		},
	)
```

Then replace the body of `isProductionEnvironmentType` (currently lines 651-658) so the production rule has one definition:

```go
func isProductionEnvironmentType(environmentType string) bool {
	return utils.IsProductionEnvironmentType(environmentType)
}
```

- [ ] **Step 5: Update the `account customer create` test stubs**

In `cmd/account/customer_create_test.go`, change all three `resolveCustomerSubscriptionByEmail` stubs. The two "must not be called" stubs (lines 362 and 439) become:

```go
	resolveCustomerSubscriptionByEmail = func(context.Context, string, dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		t.Fatal("subscription lookup should not be called")
		return nil, nil
	}
```

The asserting stub (line 406) becomes:

```go
	resolveCustomerSubscriptionByEmail = func(ctx context.Context, token string, lookup dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		assert.Equal(t, "token", token)
		assert.Equal(t, "svc-1", lookup.ServiceID)
		assert.Equal(t, "env-1", lookup.EnvironmentID)
		assert.Equal(t, "pt-1", lookup.PlanID)
		assert.Equal(t, "caller@example.com", lookup.CustomerEmail)
		return &openapiclientfleet.FleetDescribeSubscriptionResult{Id: "sub-456"}, nil
	}
```

Search the file for any remaining `resolveCustomerSubscriptionByEmail = func(` occurrences and convert them the same way:

Run: `grep -n "resolveCustomerSubscriptionByEmail = func" cmd/account/customer_create_test.go`

- [ ] **Step 6: Build, then run the tests to verify they pass**

Run: `go build ./... && go test ./internal/dataaccess/ ./cmd/account/ -v -run 'Subscription|Customer'`
Expected: PASS. If the build fails, the compiler names every remaining caller of the old signature — fix each to the struct form.

- [ ] **Step 7: Run the full unit suite and commit**

Run: `make unit-test`
Expected: PASS.

```bash
git add internal/dataaccess/subscription.go internal/dataaccess/subscription_test.go cmd/account/customer_create.go cmd/account/customer_create_test.go
git commit -m "fix: paginate and org-scope the customer subscription lookup

The lookup read only the first page of subscriptions and matched the
plain customer address, so a customer whose production identity is
stored in the org-scoped form looked absent and a second subscription
was created for them. It now paginates, falls back to the scoped form in
production only, and reports a suspended subscription instead of
returning it as usable."
```

---

### Task 4: Extract the customer user-ID matcher

Pull the email-to-user-ID search out of `CreateSubscriptionOnBehalf` into a named, testable
function. Behaviour is unchanged: the fleet users API reports unscoped addresses, so a plain
case-insensitive match is the correct and only lookup.

**Files:**
- Modify: `internal/dataaccess/subscription.go` (the `customerUserID` block inside `CreateSubscriptionOnBehalf`)
- Test: `internal/dataaccess/subscription_test.go` (append)

**Interfaces:**
- Consumes: nothing from Tasks 1-3.
- Produces: `matchUserIDByEmail(users []openapiclientfleet.AccessSideUser, email string) string` — returns `""` when no user matches.

- [ ] **Step 1: Write the failing tests**

Append to `internal/dataaccess/subscription_test.go`. Add
`"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"` to the test file's import block if it
is not already there.

```go
func TestMatchUserIDByEmail(t *testing.T) {
	users := []openapiclientfleet.AccessSideUser{
		{UserId: utils.ToPtr("user-1"), Email: utils.ToPtr("other@example.com")},
		{UserId: utils.ToPtr("user-2"), Email: utils.ToPtr("Customer@Example.com")},
		{UserId: utils.ToPtr("user-3")},
		{Email: utils.ToPtr("no-id@example.com")},
	}

	assert.Equal(t, "user-2", matchUserIDByEmail(users, "customer@example.com"))
	assert.Equal(t, "", matchUserIDByEmail(users, "absent@example.com"))
	assert.Equal(t, "", matchUserIDByEmail(users, "no-id@example.com"))
	assert.Equal(t, "", matchUserIDByEmail(nil, "customer@example.com"))
}

func TestCreateSubscriptionOnBehalfResolvesUserByEmail(t *testing.T) {
	var capturedUserID string
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/2022-09-01-00/fleet/users":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"users": []map[string]any{
					{"userId": "user-other", "email": "other@example.com"},
					{"userId": "user-test", "email": "customer@example.com"},
				},
			})
		case "/2022-09-01-00/fleet/service/s-test/environment/se-test/subscription":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			capturedUserID, _ = body["onBehalfOfCustomerUserId"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sub-new"})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	})

	result, err := CreateSubscriptionOnBehalf(context.Background(), "test-token", "s-test", "se-test", &CreateSubscriptionOnBehalfOptions{
		ProductTierID:           "pt-test",
		OnBehalfOfCustomerEmail: "customer@example.com",
	})

	require.NoError(t, err)
	assert.Equal(t, "sub-new", result.GetId())
	assert.Equal(t, "user-test", capturedUserID)
}

func TestCreateSubscriptionOnBehalfReportsUnknownEmail(t *testing.T) {
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/2022-09-01-00/fleet/users" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"users": []map[string]any{}})
	})

	_, err := CreateSubscriptionOnBehalf(context.Background(), "test-token", "s-test", "se-test", &CreateSubscriptionOnBehalfOptions{
		ProductTierID:           "pt-test",
		OnBehalfOfCustomerEmail: "customer@example.com",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer@example.com")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/dataaccess/ -run 'MatchUserIDByEmail|CreateSubscriptionOnBehalf' -v`
Expected: FAIL — `undefined: matchUserIDByEmail`.

- [ ] **Step 3: Write the implementation**

In `internal/dataaccess/subscription.go`, replace the `if customerUserID == "" && opts.OnBehalfOfCustomerEmail != ""` block inside `CreateSubscriptionOnBehalf` with:

```go
	if customerUserID == "" && opts.OnBehalfOfCustomerEmail != "" {
		listUsersRes, r, listErr := apiClient.InventoryApiAPI.InventoryApiListAllUsers(ctxWithToken).Execute()
		if r != nil {
			_ = r.Body.Close()
		}
		if listErr != nil {
			return nil, handleFleetError(errors.Wrap(listErr, "failed to list users"))
		}

		customerUserID = matchUserIDByEmail(listUsersRes.Users, opts.OnBehalfOfCustomerEmail)
		if customerUserID == "" {
			return nil, errors.Errorf("no user found with email %s", opts.OnBehalfOfCustomerEmail)
		}
	}
```

Note the response body is now closed on both the success and error paths; the original closed it
only after the error check.

Then add, next to the other helpers:

```go
// matchUserIDByEmail returns the ID of the user with the given address, or "" when there
// is none. The fleet users API reports unscoped addresses, so a plain case-insensitive
// comparison is the correct match.
func matchUserIDByEmail(users []openapiclientfleet.AccessSideUser, email string) string {
	for _, user := range users {
		if user.Email == nil || user.UserId == nil {
			continue
		}
		if strings.EqualFold(*user.Email, email) {
			return *user.UserId
		}
	}
	return ""
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/dataaccess/ -run 'MatchUserIDByEmail|CreateSubscriptionOnBehalf' -v`
Expected: PASS.

- [ ] **Step 5: Run the full unit suite and commit**

Run: `make unit-test`
Expected: PASS.

```bash
git add internal/dataaccess/subscription.go internal/dataaccess/subscription_test.go
git commit -m "refactor: extract the customer user-ID matcher

Pulls the email-to-user-ID search out of CreateSubscriptionOnBehalf into
a named function so it can be tested directly, and closes the response
body on the error path as well as the success path."
```

---


### Task 5: The `--customer-email` flag on `instance create`

Wire the flag into `instance create`, mirroring `account customer create`.

**Files:**
- Modify: `cmd/instance/create.go:20-49` (examples), `:90-127` (flags), `:129-238` (flag reads and validation), `:300-360` (resolution and request build)
- Test: `cmd/instance/create_test.go` (append)
- Regenerate: `mkdocs/docs/`

**Interfaces:**
- Consumes: `dataaccess.CustomerSubscriptionLookup`, `dataaccess.GetSubscriptionByCustomerEmailInEnvironment` (Task 3).
- Produces:
  - `validateCustomerEmailFlags(customerEmail, subscriptionID string) error`
  - `resolveInstanceSubscriptionID(ctx context.Context, token string, lookup dataaccess.CustomerSubscriptionLookup, requestedSubscriptionID string) (string, error)`
  - `var resolveInstanceSubscriptionByEmail = dataaccess.GetSubscriptionByCustomerEmailInEnvironment`

- [ ] **Step 1: Write the failing tests**

Append to `cmd/instance/create_test.go`. Add `"context"`, `"strings"`, and `"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"` to the import block.

```go
func TestCreateCommandFlags_CustomerEmail(t *testing.T) {
	flag := createCmd.Flags().Lookup("customer-email")
	require.NotNil(t, flag, "Expected flag 'customer-email' to be registered")
	assert.Contains(t, flag.Usage, "subscription")
	assert.Equal(t, "", flag.DefValue)
}

func TestValidateCustomerEmailFlags(t *testing.T) {
	require.NoError(t, validateCustomerEmailFlags("", ""))
	require.NoError(t, validateCustomerEmailFlags("", "sub-test"))
	require.NoError(t, validateCustomerEmailFlags("customer@example.com", ""))

	err := validateCustomerEmailFlags("customer@example.com", "sub-test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-email")
	assert.Contains(t, err.Error(), "--subscription-id")

	err = validateCustomerEmailFlags("not-an-email", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-email")
}

func TestResolveInstanceSubscriptionIDWithoutEmailPassesThrough(t *testing.T) {
	original := resolveInstanceSubscriptionByEmail
	t.Cleanup(func() { resolveInstanceSubscriptionByEmail = original })

	resolveInstanceSubscriptionByEmail = func(context.Context, string, dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		t.Fatal("subscription lookup should not be called without --customer-email")
		return nil, nil
	}

	subscriptionID, err := resolveInstanceSubscriptionID(
		context.Background(),
		"token",
		dataaccess.CustomerSubscriptionLookup{ServiceID: "s-test"},
		"sub-test",
	)

	require.NoError(t, err)
	assert.Equal(t, "sub-test", subscriptionID)
}

func TestResolveInstanceSubscriptionIDResolvesEmail(t *testing.T) {
	original := resolveInstanceSubscriptionByEmail
	t.Cleanup(func() { resolveInstanceSubscriptionByEmail = original })

	resolveInstanceSubscriptionByEmail = func(ctx context.Context, token string, lookup dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		assert.Equal(t, "token", token)
		assert.Equal(t, "s-test", lookup.ServiceID)
		assert.Equal(t, "se-test", lookup.EnvironmentID)
		assert.Equal(t, "PROD", lookup.EnvironmentType)
		assert.Equal(t, "pt-test", lookup.PlanID)
		assert.Equal(t, "customer@example.com", lookup.CustomerEmail)
		return &openapiclientfleet.FleetDescribeSubscriptionResult{Id: "sub-test"}, nil
	}

	subscriptionID, err := resolveInstanceSubscriptionID(
		context.Background(),
		"token",
		dataaccess.CustomerSubscriptionLookup{
			ServiceID:       "s-test",
			EnvironmentID:   "se-test",
			EnvironmentType: "PROD",
			PlanID:          "pt-test",
			CustomerEmail:   "customer@example.com",
		},
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, "sub-test", subscriptionID)
}

func TestResolveInstanceSubscriptionIDRejectsEmptyResult(t *testing.T) {
	original := resolveInstanceSubscriptionByEmail
	t.Cleanup(func() { resolveInstanceSubscriptionByEmail = original })

	resolveInstanceSubscriptionByEmail = func(context.Context, string, dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		return &openapiclientfleet.FleetDescribeSubscriptionResult{}, nil
	}

	_, err := resolveInstanceSubscriptionID(
		context.Background(),
		"token",
		dataaccess.CustomerSubscriptionLookup{CustomerEmail: "customer@example.com"},
		"",
	)

	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "empty subscription id")
}

func TestCreateExampleDocumentsCustomerEmail(t *testing.T) {
	assert.Contains(t, createExample, "--customer-email")
}
```

Also extend the expected-flag list in `TestCreateCommandFlags_AllExpectedFlags` by adding `"customer-email"` to the `expectedFlags` slice.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/instance/ -run 'CustomerEmail|ResolveInstanceSubscriptionID' -v`
Expected: FAIL — `undefined: validateCustomerEmailFlags`.

- [ ] **Step 3: Register the flag and the example**

In `cmd/instance/create.go`, append to the `createExample` const (after the on-prem example, inside the same backtick string):

```
# Create an instance deployment on behalf of an end customer, resolved from their email
omnistrate-ctl instance create --service=mysql --environment=prod --plan=mysql --resource=mySQL --cloud-provider=aws --region=us-east-2 --param-file /path/to/params.json --customer-email customer@example.com
```

In `init()`, next to the `subscription-id` flag registration (line 106):

```go
	createCmd.Flags().String("customer-email", "", "Customer email to create the instance deployment on behalf of. Resolves the customer's subscription for this service plan, creating one if they have none. Cannot be combined with --subscription-id.")
```

Add `--customer-email` to the `Use:` string on line 82, after `[--subscription-id=id]` if present, otherwise after `[--instance-id=id]`:

```go
	Use:          "create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] [--cloud-provider=aws|gcp|azure|nebius] [--region=region] [--param=param] [--param-file=file-path] [--instance-id=id] [--customer-email=email] [--customer-account-id=account-instance-id] [--cloud-provider-native-network-id=network-id] [--network-type=PUBLIC|INTERNAL] [--onprem-platform=platform] [--tags key=value,key2=value2] [--breakpoints id-or-key[:event[|event...]],...]",
```

- [ ] **Step 4: Add the helpers**

Add near the other helper functions in `cmd/instance/create.go` (for example above `validateCreateCloudTarget`):

```go
// resolveInstanceSubscriptionByEmail is a package variable so tests can exercise the
// resolution wiring without a live API.
var resolveInstanceSubscriptionByEmail = dataaccess.GetSubscriptionByCustomerEmailInEnvironment

// validateCustomerEmailFlags checks the flag combination before any API call is made.
func validateCustomerEmailFlags(customerEmail, subscriptionID string) error {
	customerEmail = strings.TrimSpace(customerEmail)
	if customerEmail == "" {
		return nil
	}
	if strings.TrimSpace(subscriptionID) != "" {
		return fmt.Errorf("cannot specify both --customer-email and --subscription-id")
	}
	if err := utils.ValidateEmail(customerEmail); err != nil {
		return fmt.Errorf("invalid --customer-email value: %w", err)
	}
	return nil
}

// resolveInstanceSubscriptionID returns the subscription the instance should be created
// under. Without --customer-email the requested subscription ID passes through unchanged,
// which keeps the existing behaviour of letting the platform choose.
func resolveInstanceSubscriptionID(
	ctx context.Context,
	token string,
	lookup dataaccess.CustomerSubscriptionLookup,
	requestedSubscriptionID string,
) (string, error) {
	if strings.TrimSpace(lookup.CustomerEmail) == "" {
		return requestedSubscriptionID, nil
	}

	subscription, err := resolveInstanceSubscriptionByEmail(ctx, token, lookup)
	if err != nil {
		return "", fmt.Errorf("failed to resolve subscription for customer %s: %w", lookup.CustomerEmail, err)
	}
	if subscription == nil || strings.TrimSpace(subscription.Id) == "" {
		return "", fmt.Errorf("subscription lookup for customer %s returned an empty subscription ID", lookup.CustomerEmail)
	}

	return strings.TrimSpace(subscription.Id), nil
}
```

- [ ] **Step 5: Wire the flag into `runCreate`**

In `runCreate`, read the flag next to the `subscription-id` read (after line 203):

```go
	customerEmail, err := cmd.Flags().GetString("customer-email")
	if err != nil {
		utils.PrintError(err)
		return err
	}
```

Validate early, immediately after the existing `validateCreateCloudTarget` call (line 234-237), so a bad flag combination fails before the spinner and any API call:

```go
	if err = validateCustomerEmailFlags(customerEmail, subscriptionID); err != nil {
		utils.PrintError(err)
		return err
	}
```

Resolve after `selectOfferingForEnvironment` and the `offering := *selectedOffering` assignment (line 311), where the environment type is known:

```go
	subscriptionID, err = resolveInstanceSubscriptionID(cmd.Context(), token, dataaccess.CustomerSubscriptionLookup{
		ServiceID:       serviceID,
		EnvironmentID:   environmentID,
		EnvironmentType: offering.ServiceEnvironmentType,
		PlanID:          productTierID,
		CustomerEmail:   customerEmail,
	}, subscriptionID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
```

The existing `if subscriptionID != "" { request.SubscriptionId = ... }` block at line 358 needs no change.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./cmd/instance/ -run 'CustomerEmail|ResolveInstanceSubscriptionID|CreateCommandFlags|CreateExample' -v`
Expected: PASS.

- [ ] **Step 7: Regenerate the docs**

Run: `make docs`
Then: `git status --short mkdocs/` — expect a modified `omnistrate-ctl_instance_create.md` containing `--customer-email`.

- [ ] **Step 8: Verify no identifying data leaked**

List every address and organization ID the change introduces and confirm each one is a
placeholder. An allowlist review is used rather than a search for specific names, so that this
plan does not itself have to spell out the data it is guarding against.

```bash
git add -A
git diff --cached -U0 | grep '^+' | grep -oE "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}|org-[A-Za-z0-9]+" | sort -u
```

Expected: nothing outside `@example.com` addresses and `org-abc123`. Any other real-looking
domain, personal name, or organization ID must be replaced with a placeholder before committing.

- [ ] **Step 9: Run the full unit suite and commit**

Run: `make unit-test`
Expected: PASS.

```bash
git add cmd/instance/create.go cmd/instance/create_test.go mkdocs/
git commit -m "feat: add --customer-email to instance create

Resolves the customer's subscription for the target service plan from
their email, creating one when they have none, so operators no longer
need to look up a subscription ID by hand."
```

---

## Manual verification

After Task 5, confirm the behaviour against a real tenant. Substitute your own service, plan, and customer values; do not paste real customer data back into the repository.

```bash
go build -o /tmp/omctl-dev .

# Rejects the flag combination before doing any work
/tmp/omctl-dev instance create --service=<service> --environment=<env> --plan=<plan> \
  --resource=<resource> --cloud-provider=aws --region=us-east-2 \
  --customer-email customer@example.com --subscription-id sub-test
# Expect: "cannot specify both --customer-email and --subscription-id"

# Resolves an existing production customer stored under the org-scoped identity
/tmp/omctl-dev instance create --service=<service> --environment=<prod-env> --plan=<plan> \
  --resource=<resource> --cloud-provider=aws --region=us-east-2 \
  --param-file ./params.json --customer-email <known-customer-email> --output json
# Expect: subscriptionId in the output matches the customer's existing subscription,
# and `omctl subscription list` shows no newly created duplicate.
```
