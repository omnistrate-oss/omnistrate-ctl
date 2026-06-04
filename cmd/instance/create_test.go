package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommandFlags(t *testing.T) {
	require.Equal(t, "Create an instance deployment", createCmd.Short)

	flag := createCmd.Flags().Lookup("customer-account-id")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "account customer list")
}

func TestCreateCommandFlags_InstanceID(t *testing.T) {
	flag := createCmd.Flags().Lookup("instance-id")
	require.NotNil(t, flag, "Expected flag 'instance-id' to be registered")
	assert.Contains(t, flag.Usage, "previously deleted instance")
	assert.Equal(t, "", flag.DefValue, "instance-id should default to empty string")
}

func TestCreateCommandFlags_AllExpectedFlags(t *testing.T) {
	expectedFlags := []string{
		"service", "environment", "plan", "version", "resource",
		"cloud-provider", "region", "param", "param-file",
		"customer-account-id", "tags", "breakpoints",
		"subscription-id", "instance-id", "wait",
	}
	for _, flagName := range expectedFlags {
		flag := createCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Expected flag '%s' not found", flagName)
	}
}

func TestCreateCommandUse_IncludesInstanceID(t *testing.T) {
	assert.Contains(t, createCmd.Use, "--instance-id")
}

func TestApplyCustomerAccountIDParam_NoCustomerAccountID(t *testing.T) {
	params := map[string]any{"existing": "value"}

	updated, err := applyCustomerAccountIDParam(params, &openapiclientfleet.ServiceOffering{}, "")
	require.NoError(t, err)
	assert.Equal(t, params, updated)
}

func TestApplyCustomerAccountIDParam_BYOARequiresCustomerAccount(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		nil,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-account-id")
	assert.Contains(t, err.Error(), "account customer list")
}

func TestApplyCustomerAccountIDParam_BYOAAllowsMagicParam(t *testing.T) {
	params := map[string]any{customerAccountConfigIDParamKey: "instance-existing"}

	updated, err := applyCustomerAccountIDParam(
		params,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, params, updated)
}

func TestApplyCustomerAccountIDParam_RequiresBYOAPlan(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		nil,
		&openapiclientfleet.ServiceOffering{ServiceModelType: "OMNISTRATE_HOSTED"},
		"instance-abcd1234",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supported for BYOA service plans")
}

func TestApplyCustomerAccountIDParam_RejectsDuplicateMagicParam(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		map[string]any{customerAccountConfigIDParamKey: "instance-existing"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"instance-abcd1234",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), customerAccountConfigIDParamKey)
}

func TestApplyCustomerAccountIDParam_AddsCustomerAccountID(t *testing.T) {
	updated, err := applyCustomerAccountIDParam(
		map[string]any{"existing": "value"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: "byoa"},
		"instance-abcd1234",
	)
	require.NoError(t, err)
	assert.Equal(t, "value", updated["existing"])
	assert.Equal(t, "instance-abcd1234", updated[customerAccountConfigIDParamKey])
}

func TestResolveServicePlanCandidates_ScopesToRequestedService(t *testing.T) {
	services := servicesFixture(
		serviceFixture("s-unrelated", "mysql", "se-prod", "Prod", "pt-prod", "mysql hosted tier"),
		serviceFixture("s-target", "mysql84786e9e-cb19-4681-b9f6-0317acecdadd", "se-dev", "dev", "pt-dev", "mysql84786e9e-cb19-4681-b9f6-0317acecdadd"),
	)

	candidates, state, err := resolveServicePlanCandidates(
		services,
		"mysql84786e9e-cb19-4681-b9f6-0317acecdadd",
		"dev",
		"mysql84786e9e-cb19-4681-b9f6-0317acecdadd",
	)

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.True(t, state.serviceFound)
	assert.Equal(t, "s-target", candidates[0].serviceID)
}

func TestResolveServicePlanCandidates_ScopesPlanToRequestedEnvironment(t *testing.T) {
	service := serviceFixture("s-target", "mysql84786e9e-cb19-4681-b9f6-0317acecdadd", "se-dev", "dev", "pt-dev", "mysql84786e9e-cb19-4681-b9f6-0317acecdadd")

	candidates, state, err := resolveServicePlanCandidates(
		servicesFixture(service),
		"mysql84786e9e-cb19-4681-b9f6-0317acecdadd",
		"dev",
		"mysql84786e9e-cb19-4681-b9f6-0317acecdadd",
	)

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.True(t, state.environmentFound)
	assert.Equal(t, "s-target", candidates[0].serviceID)
	assert.Equal(t, "se-dev", candidates[0].environmentID)
	assert.Equal(t, "pt-dev", candidates[0].productTierID)
}

func TestResolveServicePlanCandidates_MatchesIDs(t *testing.T) {
	service := serviceFixture("s-target", "mysql", "se-dev", "dev", "pt-dev", "mysql")

	candidates, state, err := resolveServicePlanCandidates(
		servicesFixture(service),
		"s-target",
		"se-dev",
		"pt-dev",
	)

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.True(t, state.planFound)
	assert.Equal(t, "s-target", candidates[0].serviceID)
	assert.Equal(t, "se-dev", candidates[0].environmentID)
	assert.Equal(t, "pt-dev", candidates[0].productTierID)
}

func TestResolveServicePlanCandidates_AllowsLaterDisambiguation(t *testing.T) {
	services := servicesFixture(
		serviceFixture("s-prod", "mysql", "se-prod", "Prod", "pt-prod", "mysql hosted tier"),
		serviceFixture("s-dev", "mysql", "se-dev", "dev", "pt-dev", "mysql hosted tier"),
	)

	candidates, _, err := resolveServicePlanCandidates(
		services,
		"mysql",
		"dev",
		"mysql hosted tier",
	)

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "s-dev", candidates[0].serviceID)
}

func TestResolveResourceFromServiceOffering_MatchesIDsAndNames(t *testing.T) {
	offering := serviceOfferingFixture("r-target-mysql", "MySQL")

	resourceID, err := resolveResourceFromServiceOffering(offering, "mySQL")

	require.NoError(t, err)
	assert.Equal(t, "r-target-mysql", resourceID)

	resourceID, err = resolveResourceFromServiceOffering(offering, "r-target-mysql")

	require.NoError(t, err)
	assert.Equal(t, "r-target-mysql", resourceID)
}

func TestResolveResourceFromServiceOffering_NotFound(t *testing.T) {
	_, err := resolveResourceFromServiceOffering(serviceOfferingFixture("r-target-mysql", "MySQL"), "Postgres")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "target resource not found")
}

func TestResolveResourceFromServiceOffering_DeduplicatesSameResourceAcrossOfferings(t *testing.T) {
	offering := serviceOfferingFixture("r-target-mysql", "MySQL")
	offering.Offerings = append(offering.Offerings, offering.Offerings[0])

	resourceID, err := resolveResourceFromServiceOffering(offering, "MySQL")

	require.NoError(t, err)
	assert.Equal(t, "r-target-mysql", resourceID)
}

func servicesFixture(services ...openapiclientv1.DescribeServiceResult) *openapiclientv1.ListServiceResult {
	return &openapiclientv1.ListServiceResult{
		Ids:      []string{},
		Services: services,
	}
}

func serviceFixture(serviceID, serviceName, environmentID, environmentName, productTierID, productTierName string) openapiclientv1.DescribeServiceResult {
	return openapiclientv1.DescribeServiceResult{
		Id:   serviceID,
		Name: serviceName,
		ServiceEnvironments: []openapiclientv1.ServiceEnvironment{
			{
				Id:   environmentID,
				Name: environmentName,
				ServicePlans: []openapiclientv1.ServicePlan{
					{
						Name:          productTierName,
						ProductTierID: productTierID,
					},
				},
			},
		},
	}
}

func serviceOfferingFixture(resourceID, resourceName string) *openapiclientv1.DescribeServiceOfferingResult {
	return &openapiclientv1.DescribeServiceOfferingResult{
		Offerings: []openapiclientv1.ServiceOffering{
			{
				ResourceParameters: []openapiclientv1.ResourceEntity{
					{
						Name:       resourceName,
						ResourceId: resourceID,
					},
				},
			},
		},
	}
}
