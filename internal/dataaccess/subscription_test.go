package dataaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
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

func TestMatchSubscriptionsByEmailWithoutPlanFilter(t *testing.T) {
	onOtherPlan := testSubscription("sub-2", "customer@example.com", "ACTIVE")
	onOtherPlan.ProductTierId = "pt-other"
	subscriptions := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "ACTIVE"),
		onOtherPlan,
		testSubscription("sub-3", "other@example.com", "ACTIVE"),
	}

	matches := matchSubscriptionsByEmail(subscriptions, "", "customer@example.com")

	require.Len(t, matches, 2)
	assert.Equal(t, "sub-1", matches[0].Id)
	assert.Equal(t, "sub-2", matches[1].Id)
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
	assert.Contains(t, err.Error(), "resume")
}

func TestSelectCustomerSubscriptionReportsNonSuspendedMultipleCandidates(t *testing.T) {
	candidates := []openapiclientfleet.FleetDescribeSubscriptionResult{
		testSubscription("sub-1", "customer@example.com", "CANCELLED"),
		testSubscription("sub-2", "customer@example.com", "TERMINATED"),
	}

	_, err := selectCustomerSubscription(candidates, "customer@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub-1 (CANCELLED)")
	assert.Contains(t, err.Error(), "sub-2 (TERMINATED)")
	assert.Contains(t, err.Error(), "--subscription-id")
	assert.NotContains(t, err.Error(), "resume")
}

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
// the test.
func startSubscriptionTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
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

// serviceOfferingJSON builds a fleet service-offering payload with every field the SDK's
// generated ServiceOffering.UnmarshalJSON requires present, for one offering on one plan.
func serviceOfferingJSON(planID, environmentID, environmentType string) map[string]any {
	return map[string]any{
		"ConsumptionDescribeServiceOfferingResult": map[string]any{
			"createdAt":           "2026-09-07T00:00:00Z",
			"isDeprecated":        false,
			"serviceDescription":  "test-service",
			"serviceId":           "s-test",
			"serviceName":         "test-service",
			"serviceOrgId":        "org-abc123",
			"serviceProviderId":   "sp-test",
			"serviceProviderName": "test-provider",
			"serviceURLKey":       "test-service",
			"offerings": []map[string]any{
				{
					"AutoApproveSubscription":      false,
					"productTierDocumentation":     "",
					"productTierID":                planID,
					"productTierName":              "test-plan",
					"productTierPricing":           nil,
					"productTierSupport":           "",
					"productTierType":              "",
					"productTierURLKey":            "",
					"productTierVersion":           "",
					"resourceParameters":           []any{},
					"serviceAPIID":                 "",
					"serviceAPIVersion":            "",
					"serviceEnvironmentID":         environmentID,
					"serviceEnvironmentName":       "",
					"serviceEnvironmentType":       environmentType,
					"serviceEnvironmentURLKey":     "",
					"serviceEnvironmentVisibility": "",
					"serviceLogoURL":               "",
					"serviceModelID":               "",
					"serviceModelName":             "",
					"serviceModelStatus":           "",
					"serviceModelType":             "",
					"serviceModelURLKey":           "",
				},
			},
		},
	}
}

// TestGetSubscriptionByCustomerEmailPlumbsEnvironmentTypeToScopedFallback proves that the
// plan-level wrapper GetSubscriptionByCustomerEmail actually threads the offering's
// ServiceEnvironmentType into the lookup: only a customer whose subscription exists under
// the org-scoped address is present, so the scoped fallback must fire, which only happens
// when EnvironmentType reaches the lookup as "PROD".
func TestGetSubscriptionByCustomerEmailPlumbsEnvironmentTypeToScopedFallback(t *testing.T) {
	var describeUserCalls int
	startSubscriptionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/2022-09-01-00/fleet/service-offering/s-test":
			_ = json.NewEncoder(w).Encode(serviceOfferingJSON("pt-test", "se-test", "PROD"))
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

	subscription, err := GetSubscriptionByCustomerEmail(context.Background(), "test-token", "s-test", "pt-test", "customer@example.com")

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, "sub-1", subscription.Id)
	assert.Equal(t, 1, describeUserCalls, "the offering's ServiceEnvironmentType must reach the lookup and trigger the scoped fallback")
}

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
