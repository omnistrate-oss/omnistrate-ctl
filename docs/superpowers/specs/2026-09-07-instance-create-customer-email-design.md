# `instance create --customer-email` design

Let a service provider create an instance on behalf of an end customer by naming the
customer's email instead of hunting down their subscription ID.

## Problem

`omnistrate-ctl instance create` accepts `--subscription-id`. Operators know customers by
email, not by `sub-XXXXXXXX`, so using it means a manual `subscription list` and a
copy-paste. `omnistrate-ctl account customer create` already solved this with a
`--customer-email` flag; `instance create` should behave the same way.

The lookup that flag relies on, `dataaccess.GetSubscriptionByCustomerEmailInEnvironment`,
has three defects that must be fixed for the feature to be correct:

1. **It misses org-scoped customers.** In production environments the platform stores an
   end customer's subscription under an *org-scoped* identity: the service provider's org
   ID is appended to the local part of the email, as `<local>+<orgID>@<domain>`. So
   `customer@example.com` owns a subscription whose `rootUserEmail` reads
   `customer+org-abc123@example.com`. The fleet users API (`/fleet/users`) reports the
   plain, unscoped email, but the fleet subscription API reports the scoped one. Matching
   `rootUserEmail` against the raw input therefore finds nothing, and the helper's
   not-found branch **creates a second subscription** for a customer who already has one.
2. **It reads only the first page.** It calls `InventoryApiListSubscription` once and
   ignores `nextPageToken`, so customers past the first page look like they have no
   subscription — again ending in a duplicate.
3. **It ignores subscription status.** A SUSPENDED subscription is returned as if usable,
   and the create then fails downstream with an opaque error.

## Format rules

The canonical implementation lives server-side in `commons/pkg/utils/scopedorgutils.go`.
This design mirrors it so the CLI and server agree:

- Scoped form is `<local>+<orgID>@<domain>`; the org ID is appended as the **last** `+`
  segment of the local part.
- An org ID matches `^org-[a-zA-Z0-9][a-zA-Z0-9._-]*$`.
- An address is already scoped when its last `+` segment matches that pattern.
- Pre-existing plus tags are preserved: `customer+tag@example.com` scoped with
  `org-abc123` becomes `customer+tag+org-abc123@example.com`.
- **Org scoping only exists in production environments.** The fallback must not be
  attempted outside PROD.

## Subscription statuses

The server's vocabulary is `ACTIVE`, `SUSPENDED`, `CANCELLED`, `TERMINATED`.

`TerminateSubscription` soft-deletes the row, and the default listing path excludes
soft-deleted rows — `includeInactive` is what switches the query to unscoped. TERMINATED
and CANCELLED subscriptions are therefore invisible to the default listing, which is the
behaviour we want: a terminated subscription should not block a customer from being
re-subscribed. `SuspendSubscription` only sets the status, leaving the row live, so
SUSPENDED is the non-ACTIVE status a lookup can realistically encounter.

Consequently the listing keeps `IncludeInactive` off. Enabling it would resurrect
terminated rows and turn legitimate re-subscription into an error.

## Design

### 1. `internal/utils/scopedemail.go` (new)

```go
// FormatEmailWithScopedOrg returns the org-scoped form of an address:
//   ("customer@example.com", "org-abc123") -> "customer+org-abc123@example.com"
func FormatEmailWithScopedOrg(email, orgID string) (string, error)

// EmailHasScopedOrg reports whether the local part already ends in an org ID segment.
func EmailHasScopedOrg(email string) bool

// IsProductionEnvironmentType gates where org scoping applies. It replaces the private
// copy in cmd/account so the production rule has one definition.
func IsProductionEnvironmentType(environmentType string) bool
```

`FormatEmailWithScopedOrg` errors on a malformed address, a malformed org ID, or an
address that is already scoped. `EmailHasScopedOrg` lets the lookup skip the scoped pass
when the operator already passed a scoped address, which would otherwise scope it twice.
Both functions are pure and make no API calls.

### 2. `internal/dataaccess/subscription.go`

The lookup takes an options struct rather than growing to six positional strings, and
gains the environment type it needs to decide whether org scoping applies:

```go
type CustomerSubscriptionLookup struct {
	ServiceID       string
	EnvironmentID   string
	EnvironmentType string // org-scoped fallback applies to PROD only
	PlanID          string
	CustomerEmail   string
}

func GetSubscriptionByCustomerEmailInEnvironment(
	ctx context.Context,
	token string,
	lookup CustomerSubscriptionLookup,
) (*openapiclientfleet.FleetDescribeSubscriptionResult, error)
```

Resolution order:

1. `ListAllSubscriptions` filtered by `ProductTierId` — the paginating variant, replacing
   the current single-page call.
2. Collect candidates whose `rootUserEmail` equals the requested email, case-insensitively.
3. If there are none, the environment type is production, and the requested address is not
   already scoped, call `DescribeUser` for the caller's org ID, build the scoped address,
   and collect candidates again. The `DescribeUser` call is lazy: an exact match never
   pays for it.
4. Apply the status gate below. It resolves to a subscription, an error, or "no candidates".
5. On "no candidates", `CreateSubscriptionOnBehalf`, then `DescribeSubscription`, then run
   the result through the same status gate.

The status gate is a pure function over the candidate slice, which is what makes it
testable without a network:

| Candidates | Result |
| --- | --- |
| none | fall through to create |
| exactly one ACTIVE | use it |
| more than one ACTIVE | error listing the IDs, directing the user to `--subscription-id` |
| one or more, none ACTIVE | status-specific error |

Exact-email candidates take precedence over scoped-email candidates; the scoped pass only
runs when the exact pass found nothing, so the two sets never compete.

The SUSPENDED error names its remedy:

> subscription sub-XXXX for customer@example.com on plan `<plan>` is SUSPENDED and cannot
> be used to create an instance. Resume it with
> `omnistrate-ctl subscription resume sub-XXXX`, or pass `--subscription-id` to use a
> different subscription.

Any other non-ACTIVE status produces the generic form (`is in status <STATUS>, expected
ACTIVE`), so a status added later cannot be silently treated as usable.

`CreateSubscriptionOnBehalf`'s inline email-to-user-ID search moves into a helper and gains
the same exact-then-scoped ordering, gated on environment type the same way. The
scoped pass costs nothing there — the user list is already in memory — and in practice the
exact pass is what matches, because `/fleet/users` reports unscoped addresses.

`GetSubscriptionByCustomerEmail`, the plan-level wrapper, passes the offering's
`ServiceEnvironmentType` through to the lookup.

### 3. `cmd/instance/create.go`

Add `--customer-email`, mirroring `account customer create`:

- Rejected together with `--subscription-id`.
- Validated with `utils.ValidateEmail` before any API call.
- Resolved after `selectOfferingForEnvironment`, so target errors surface first and
  `offering.ServiceEnvironmentType` is available. The resolved ID flows into the existing
  `request.SubscriptionId` assignment; request construction is otherwise untouched.
- Accepted in any environment. Only the org-scoped fallback is gated on PROD.
- Creates the subscription when the customer has none, and says so in the flag help. The
  command already prints `SubscriptionID`, so the operator sees which subscription was used.

Explicitly **not** copied from `account customer create`: its behaviour of defaulting to
the calling user's own email in PROD when neither flag is given. `instance create` with no
flags keeps omitting `subscriptionId` and letting the server resolve it, exactly as today.

`createExample` gains an entry for the new flag.

### 4. Inherited fixes

Because the lookup is shared, `instance adopt` and `account customer create` also stop
creating duplicate subscriptions for org-scoped customers, start paginating, and start
reporting suspended subscriptions instead of failing downstream.

## Testing

- `internal/utils/scopedemail_test.go` — scoping, rejection of an already-scoped address,
  preservation of an existing plus tag, malformed org ID, malformed address.
- `internal/dataaccess/subscription_test.go` — the status gate and candidate selection:
  exact beats scoped, single ACTIVE, multiple ACTIVE, suspended-only, plan filtering.
- `cmd/instance/create_test.go` — flag validation (both flags together, malformed email)
  and resolution wiring through an injectable package-level func var.
- Update the `account customer create` test stubs for the new signature.
- `make docs` to regenerate the generated CLI reference under `mkdocs/docs/`.

All fixtures use placeholder identities (`customer@example.com`, `org-abc123`). No real
customer, organization, or address appears in code, tests, or documentation.
