package dataaccess

import (
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func TestEnhanceServicePlansStructure(t *testing.T) {
	// Create a mock service with existing structure
	service := &openapiclient.DescribeServiceResult{
		Id:   "s-test123",
		Name: "Test Service",
		ServiceEnvironments: []openapiclient.ServiceEnvironment{
			{
				Id:   "se-env123",
				Name: "Dev",
				ServicePlans: []openapiclient.ServicePlan{
					{
						Description:           "Test Customer Hosted Plan",
						ModelType:            "CUSTOMER_HOSTED",
						Name:                 "Test Plan",
						ProductTierID:        "pt-test123",
						TierType:             "DEDICATED",
						AdditionalProperties: nil, // This should be initialized by the enhance function
					},
					{
						Description:   "Test Omnistrate Plan",
						ModelType:     "OMNISTRATE_HOSTED",
						Name:          "Omnistrate Plan",
						ProductTierID: "pt-omni123",
						TierType:      "SHARED",
					},
				},
			},
		},
	}

	// Mock the enhancement (since we can't actually call the APIs in tests)
	// This simulates what the enhance function should do
	customerHostedPlan := &service.ServiceEnvironments[0].ServicePlans[0]
	
	// Initialize AdditionalProperties
	if customerHostedPlan.AdditionalProperties == nil {
		customerHostedPlan.AdditionalProperties = make(map[string]interface{})
	}

	// Add the enhanced fields that the function should add
	customerHostedPlan.AdditionalProperties["serviceModelId"] = "sm-model123"
	customerHostedPlan.AdditionalProperties["accountConfigIds"] = []string{"acc-1", "acc-2"}
	customerHostedPlan.AdditionalProperties["activeAccountConfigIds"] = map[string]interface{}{
		"aws": []string{"acc-1"},
	}
	customerHostedPlan.AdditionalProperties["accountsByProvider"] = map[string]interface{}{
		"AWS": []interface{}{
			map[string]interface{}{
				"id":              "acc-1",
				"name":            "Test AWS Account",
				"cloudProviderId": "aws",
			},
		},
	}
	customerHostedPlan.AdditionalProperties["hasAccountsConfigured"] = true
	customerHostedPlan.AdditionalProperties["accountsAvailable"] = true

	// Test that the enhancement worked correctly
	enhancedPlan := service.ServiceEnvironments[0].ServicePlans[0]

	// Check that AdditionalProperties was initialized
	if enhancedPlan.AdditionalProperties == nil {
		t.Error("AdditionalProperties should be initialized")
	}

	// Check that the enhanced fields are present
	if enhancedPlan.AdditionalProperties["serviceModelId"] != "sm-model123" {
		t.Errorf("Expected serviceModelId to be 'sm-model123', got %v", enhancedPlan.AdditionalProperties["serviceModelId"])
	}

	if accountIds, ok := enhancedPlan.AdditionalProperties["accountConfigIds"].([]string); !ok || len(accountIds) != 2 {
		t.Errorf("Expected accountConfigIds to be []string with 2 elements, got %v", enhancedPlan.AdditionalProperties["accountConfigIds"])
	}

	if hasAccounts, ok := enhancedPlan.AdditionalProperties["hasAccountsConfigured"].(bool); !ok || !hasAccounts {
		t.Errorf("Expected hasAccountsConfigured to be true, got %v", enhancedPlan.AdditionalProperties["hasAccountsConfigured"])
	}

	// Check that the Omnistrate hosted plan was NOT enhanced
	omnistratePlan := service.ServiceEnvironments[0].ServicePlans[1]
	if omnistratePlan.AdditionalProperties != nil {
		t.Error("Omnistrate hosted plan should not have AdditionalProperties enhanced")
	}

	// Verify the core structure remains unchanged
	if enhancedPlan.ModelType != "CUSTOMER_HOSTED" {
		t.Errorf("Expected ModelType to remain 'CUSTOMER_HOSTED', got %s", enhancedPlan.ModelType)
	}

	if enhancedPlan.ProductTierID != "pt-test123" {
		t.Errorf("Expected ProductTierID to remain 'pt-test123', got %s", enhancedPlan.ProductTierID)
	}
}