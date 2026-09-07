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
