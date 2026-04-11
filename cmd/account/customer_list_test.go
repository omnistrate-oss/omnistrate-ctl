package account

import (
	"context"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerAccountListFiltersFromFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("cloud-provider", "", "")
	cmd.Flags().String("service", "", "")
	cmd.Flags().String("plan", "", "")
	cmd.Flags().String("subscription-id", "", "")
	cmd.Flags().String(customerEmailFlag, "", "")

	require.NoError(t, cmd.Flags().Set("cloud-provider", "nebius"))
	require.NoError(t, cmd.Flags().Set("service", "Nebius"))
	require.NoError(t, cmd.Flags().Set("plan", "Nebius BYOA"))
	require.NoError(t, cmd.Flags().Set("subscription-id", "sub-123"))

	filters, err := customerAccountListFiltersFromFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, customerAccountListFilters{
		CloudProvider:  "nebius",
		Service:        "Nebius",
		Plan:           "Nebius BYOA",
		SubscriptionID: "sub-123",
		CustomerEmail:  "",
	}, filters)
}

func TestValidateCustomerAccountListFilters_RequiresServiceAndPlanForCustomerEmail(t *testing.T) {
	err := validateCustomerAccountListFilters(customerAccountListFilters{
		CustomerEmail: "customer@example.com",
		Service:       "Nebius",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-email requires both --service and --plan")
}

func TestResolveCustomerAccountListSubscriptionIDs_ByCustomerEmail(t *testing.T) {
	originalResolveService := resolveCustomerListServiceFn
	originalResolvePlan := resolveCustomerListPlanIDFn
	originalDescribeOffering := describeServiceOfferingForListFn
	originalListSubscriptions := listSubscriptionsForEnvironmentFn
	t.Cleanup(func() {
		resolveCustomerListServiceFn = originalResolveService
		resolveCustomerListPlanIDFn = originalResolvePlan
		describeServiceOfferingForListFn = originalDescribeOffering
		listSubscriptionsForEnvironmentFn = originalListSubscriptions
	})

	resolveCustomerListServiceFn = func(ctx context.Context, token string, serviceArg string) (string, string, error) {
		assert.Equal(t, "token", token)
		assert.Equal(t, "Nebius", serviceArg)
		return "svc-123", "Nebius", nil
	}

	resolveCustomerListPlanIDFn = func(ctx context.Context, token string, serviceID string, planArg string) (string, error) {
		assert.Equal(t, "svc-123", serviceID)
		assert.Equal(t, "Nebius BYOA", planArg)
		return "pt-123", nil
	}

	describeServiceOfferingForListFn = func(ctx context.Context, token, serviceID, productTierID, productTierVersion string) (*openapiclientfleet.InventoryDescribeServiceOfferingResult, error) {
		assert.Equal(t, "svc-123", serviceID)
		assert.Equal(t, "pt-123", productTierID)
		return &openapiclientfleet.InventoryDescribeServiceOfferingResult{
			ConsumptionDescribeServiceOfferingResult: &openapiclientfleet.DescribeServiceOfferingResult{
				Offerings: []openapiclientfleet.ServiceOffering{
					{
						ProductTierID:        "pt-123",
						ServiceEnvironmentID: "env-dev",
					},
					{
						ProductTierID:        "pt-123",
						ServiceEnvironmentID: "env-prod",
					},
					{
						ProductTierID:        "pt-123",
						ServiceEnvironmentID: "env-prod",
					},
				},
			},
		}, nil
	}

	listSubscriptionsForEnvironmentFn = func(ctx context.Context, token, serviceID, environmentID string) (*openapiclientfleet.FleetListSubscriptionsResult, error) {
		switch environmentID {
		case "env-dev":
			return &openapiclientfleet.FleetListSubscriptionsResult{
				Subscriptions: []openapiclientfleet.FleetDescribeSubscriptionResult{
					{Id: "sub-dev", ProductTierId: "pt-123", RootUserEmail: "customer@example.com"},
					{Id: "sub-other", ProductTierId: "pt-123", RootUserEmail: "other@example.com"},
				},
			}, nil
		case "env-prod":
			return &openapiclientfleet.FleetListSubscriptionsResult{
				Subscriptions: []openapiclientfleet.FleetDescribeSubscriptionResult{
					{Id: "sub-prod", ProductTierId: "pt-123", RootUserEmail: "customer@example.com"},
				},
			}, nil
		default:
			t.Fatalf("unexpected environment lookup %s", environmentID)
			return nil, nil
		}
	}

	subscriptionIDs, err := resolveCustomerAccountListSubscriptionIDs(&cobra.Command{}, "token", customerAccountListFilters{
		Service:       "Nebius",
		Plan:          "Nebius BYOA",
		CustomerEmail: "customer@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{
		"sub-dev":  {},
		"sub-prod": {},
	}, subscriptionIDs)
}

func TestMatchesCustomerAccountListFilters(t *testing.T) {
	productTierName := "Nebius BYOA"
	subscriptionID := "sub-123"

	record := openapiclientfleet.ResourceInstanceSearchRecord{
		CloudProvider:   "nebius",
		ServiceId:       "svc-123",
		ServiceName:     "Nebius",
		ProductTierId:   "pt-123",
		ProductTierName: &productTierName,
		SubscriptionId:  &subscriptionID,
	}

	assert.True(t, matchesCustomerAccountListFilters(record, customerAccountListFilters{
		CloudProvider: "nebius",
		Service:       "Nebius",
		Plan:          "Nebius BYOA",
	}, nil))

	assert.False(t, matchesCustomerAccountListFilters(record, customerAccountListFilters{
		CloudProvider: "aws",
	}, nil))

	assert.False(t, matchesCustomerAccountListFilters(record, customerAccountListFilters{
		Service: "Other",
	}, nil))

	assert.True(t, matchesCustomerAccountListFilters(record, customerAccountListFilters{}, map[string]struct{}{
		"sub-123": {},
	}))
}
