package account

import (
	"context"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerCommandRegistersAllSubcommands(t *testing.T) {
	require.NotNil(t, customerCmd.Commands())

	commandNames := map[string]bool{}
	for _, command := range customerCmd.Commands() {
		commandNames[command.Name()] = true
	}

	assert.True(t, commandNames["create"])
	assert.True(t, commandNames["update"])
	assert.True(t, commandNames["delete"])
	assert.True(t, commandNames["list"])
	assert.True(t, commandNames["describe"])
}

func TestCustomerAccountListItemFromSearchRecord(t *testing.T) {
	resourceID := "r-injectedaccountconfigpt123"
	planName := "Nebius BYOA"
	version := "2026-04-01"
	subscriptionID := "sub-123"

	item, ok := customerAccountListItemFromSearchRecord(openapiclientfleet.ResourceInstanceSearchRecord{
		Id:                     "instance-123",
		ServiceName:            "Nebius",
		ServiceEnvironmentName: "dev",
		ProductTierId:          "pt-123",
		ProductTierName:        &planName,
		ProductTierVersion:     &version,
		ResourceId:             &resourceID,
		ResourceName:           "Cloud Provider Account",
		CloudProvider:          "nebius",
		RegionCode:             "eu-north1",
		Status:                 "READY",
		SubscriptionId:         &subscriptionID,
	})
	require.True(t, ok)
	assert.Equal(t, customerAccountListItem{
		InstanceID:     "instance-123",
		Service:        "Nebius",
		Environment:    "dev",
		Plan:           "Nebius BYOA",
		Version:        "2026-04-01",
		Resource:       "Cloud Provider Account",
		CloudProvider:  "nebius",
		Status:         "READY",
		SubscriptionID: "sub-123",
	}, *item)
}

func TestCustomerAccountListItemFromSearchRecord_IgnoresNonBYOAResource(t *testing.T) {
	resourceID := "r-somethingelse"
	item, ok := customerAccountListItemFromSearchRecord(openapiclientfleet.ResourceInstanceSearchRecord{
		Id:         "instance-123",
		ResourceId: &resourceID,
	})
	assert.False(t, ok)
	assert.Nil(t, item)
}

func TestResolveCustomerAccountInstanceByID(t *testing.T) {
	originalSearchInventory := searchInventoryFn
	t.Cleanup(func() {
		searchInventoryFn = originalSearchInventory
	})

	resourceID := "r-injectedaccountconfigpt123"
	planName := "Nebius BYOA"
	version := "2026-04-01"
	subscriptionID := "sub-123"

	searchInventoryFn = func(ctx context.Context, token string, query string) (*openapiclientfleet.SearchInventoryResult, error) {
		assert.Equal(t, "token", token)
		assert.Equal(t, "resourceinstance:instance-123", query)
		return &openapiclientfleet.SearchInventoryResult{
			ResourceInstanceResults: []openapiclientfleet.ResourceInstanceSearchRecord{
				{
					Id:                     "instance-123",
					ServiceId:              "svc-123",
					ServiceName:            "Nebius",
					ServiceEnvironmentId:   "env-123",
					ServiceEnvironmentName: "dev",
					ProductTierId:          "pt-123",
					ProductTierName:        &planName,
					ProductTierVersion:     &version,
					ResourceId:             &resourceID,
					ResourceName:           "Cloud Provider Account",
					CloudProvider:          "nebius",
					RegionCode:             "eu-north1",
					Status:                 "READY",
					SubscriptionId:         &subscriptionID,
				},
			},
		}, nil
	}

	ref, err := resolveCustomerAccountInstanceByID(context.Background(), "token", "instance-123")
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, &customerAccountInstanceRef{
		InstanceID:     "instance-123",
		ServiceID:      "svc-123",
		ServiceName:    "Nebius",
		EnvironmentID:  "env-123",
		Environment:    "dev",
		PlanID:         "pt-123",
		Plan:           "Nebius BYOA",
		Version:        "2026-04-01",
		ResourceID:     "r-injectedaccountconfigpt123",
		Resource:       "Cloud Provider Account",
		CloudProvider:  "nebius",
		Region:         "eu-north1",
		Status:         "READY",
		SubscriptionID: "sub-123",
	}, ref)
}

func TestBuildCustomerAccountSummary(t *testing.T) {
	ref := &customerAccountInstanceRef{
		InstanceID:     "instance-123",
		ServiceName:    "Nebius",
		Environment:    "dev",
		Plan:           "Nebius BYOA",
		Version:        "2026-04-01",
		Resource:       "Cloud Provider Account",
		CloudProvider:  "",
		Region:         "eu-north1",
		Status:         "READY",
		SubscriptionID: "sub-123",
	}

	account := &openapiclient.DescribeAccountConfigResult{
		Id:             "ac-123",
		Name:           "customer-nebius",
		Description:    "customer hosted account",
		Status:         "READY",
		StatusMessage:  "account verified",
		NebiusTenantID: strPtr("tenant-123"),
		NebiusBindings: []openapiclient.NebiusAccountBindingResult{
			{
				ProjectID: "project-123",
			},
		},
	}

	summary := buildCustomerAccountSummary(ref, "ac-123", account)
	assert.Equal(t, customerAccountSummary{
		InstanceID:         "instance-123",
		AccountConfigID:    "ac-123",
		AccountName:        "customer-nebius",
		AccountDescription: "customer hosted account",
		TargetAccountID:    "tenant-123 (1 bindings)",
		Service:            "Nebius",
		Environment:        "dev",
		Plan:               "Nebius BYOA",
		Version:            "2026-04-01",
		Resource:           "Cloud Provider Account",
		CloudProvider:      "Nebius",
		Region:             "eu-north1",
		InstanceStatus:     "READY",
		AccountStatus:      "READY",
		AccountStatusMsg:   "account verified",
		SubscriptionID:     "sub-123",
	}, summary)
}

func strPtr(value string) *string {
	return &value
}
