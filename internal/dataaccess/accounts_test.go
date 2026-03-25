package dataaccess

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func TestAccountStructures(t *testing.T) {
	// Test ServiceAccountInfo structure
	accountInfo := &ServiceAccountInfo{
		ServiceID:   "test-service-id",
		ServiceName: "test-service",
		AccountsByProvider: []AccountsByCloudProviderAndPlan{
			{
				CloudProvider: "AWS",
				Plans: []PlanWithAccounts{
					{
						PlanName:               "Basic Plan",
						PlanID:                 "basic-plan-id",
						ModelType:              "Customer",
						ServiceModelId:         "sm-12345",
						AccountConfigIds:       []string{"account-1"},
						ActiveAccountConfigIds: map[string]string{"aws": "account-1"},
						Accounts: []openapiclientfleet.FleetDescribeAccountConfigResult{
							{
								Id:              "account-1",
								Name:            "Test Account 1",
								CloudProviderId: "aws",
								Description:     "Test AWS account",
								Status:          "Active",
								StatusMessage:   "Ready",
							},
						},
					},
				},
			},
		},
	}

	if accountInfo.ServiceID != "test-service-id" {
		t.Errorf("Expected ServiceID to be 'test-service-id', got %s", accountInfo.ServiceID)
	}

	if len(accountInfo.AccountsByProvider) != 1 {
		t.Errorf("Expected 1 cloud provider, got %d", len(accountInfo.AccountsByProvider))
	}

	provider := accountInfo.AccountsByProvider[0]
	if provider.CloudProvider != "AWS" {
		t.Errorf("Expected CloudProvider to be 'AWS', got %s", provider.CloudProvider)
	}

	if len(provider.Plans) != 1 {
		t.Errorf("Expected 1 plan, got %d", len(provider.Plans))
	}

	plan := provider.Plans[0]
	if plan.PlanName != "Basic Plan" {
		t.Errorf("Expected PlanName to be 'Basic Plan', got %s", plan.PlanName)
	}

	if len(plan.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(plan.Accounts))
	}
}

func TestIsCustomerHosted(t *testing.T) {
	testCases := []struct {
		modelType string
		expected  bool
	}{
		{"Customer", true},
		{"customer", true},
		{"CUSTOMER", true},
		{"CUSTOMER_HOSTED", true},
		{"Customer-Hosted", true},
		{"customer_hosted", true},
		{"Omnistrate", false},
		{"BYOA", false},
		{"SaaS", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isCustomerHosted(tc.modelType)
		if result != tc.expected {
			t.Errorf("For modelType '%s', expected %v, got %v", tc.modelType, tc.expected, result)
		}
	}
}

func TestGetCloudProviderName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"aws", "AWS"},
		{"AWS", "AWS"},
		{"gcp", "GCP"},
		{"GCP", "GCP"},
		{"azure", "Azure"},
		{"AZURE", "Azure"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := getCloudProviderName(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}
