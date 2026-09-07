package instance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommandFlags(t *testing.T) {
	require.Equal(t, "Create an instance deployment", createCmd.Short)

	flag := createCmd.Flags().Lookup("customer-account-id")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "account customer list")

	flag = createCmd.Flags().Lookup("cloud-provider-native-network-id")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, cloudProviderNativeNetworkIDParamKey)

	flag = createCmd.Flags().Lookup("onprem-platform")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "installer-backed")

	flag = createCmd.Flags().Lookup("breakpoints")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "id-or-key:event")

	flag = createCmd.Flags().Lookup("network-type")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "PUBLIC")
	assert.Contains(t, flag.Usage, "INTERNAL")
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
		"customer-account-id", "cloud-provider-native-network-id", "onprem-platform", "tags", "breakpoints",
		"subscription-id", "instance-id", "wait", "network-type", "customer-email",
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

	updated, err := applyCustomerAccountIDParam(params, &openapiclientfleet.ServiceOffering{}, "", "")
	require.NoError(t, err)
	assert.Equal(t, params, updated)
}

func TestApplyCustomerAccountIDParam_BYOARequiresCustomerAccount(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		nil,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-account-id")
	assert.Contains(t, err.Error(), "--onprem-platform")
	assert.Contains(t, err.Error(), "account customer list")
}

func TestApplyCustomerAccountIDParam_BYOAWithOnpremPlatformDoesNotRequireCustomerAccount(t *testing.T) {
	updated, err := applyCustomerAccountIDParam(
		map[string]any{"existing": "value"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
		"Generic",
	)

	require.NoError(t, err)
	assert.Equal(t, "value", updated["existing"])
	assert.NotContains(t, updated, customerAccountConfigIDParamKey)
}

func TestApplyCustomerAccountIDParam_BYOAAllowsMagicParam(t *testing.T) {
	params := map[string]any{customerAccountConfigIDParamKey: "instance-existing"}

	updated, err := applyCustomerAccountIDParam(
		params,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
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
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supported for BYOA service plans")
}

func TestApplyCustomerAccountIDParam_RejectsDuplicateMagicParam(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		map[string]any{customerAccountConfigIDParamKey: "instance-existing"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"instance-abcd1234",
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), customerAccountConfigIDParamKey)
}

func TestApplyCustomerAccountIDParam_AddsCustomerAccountID(t *testing.T) {
	updated, err := applyCustomerAccountIDParam(
		map[string]any{"existing": "value"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: "byoa"},
		"instance-abcd1234",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "value", updated["existing"])
	assert.Equal(t, "instance-abcd1234", updated[customerAccountConfigIDParamKey])
}

func TestApplyCloudProviderNativeNetworkIDParam_NoNativeNetworkID(t *testing.T) {
	params := map[string]any{"existing": "value"}

	updated := applyCloudProviderNativeNetworkIDParam(params, "")

	assert.Equal(t, params, updated)
}

func TestApplyCloudProviderNativeNetworkIDParam_OverwritesExistingMagicParam(t *testing.T) {
	updated := applyCloudProviderNativeNetworkIDParam(
		map[string]any{cloudProviderNativeNetworkIDParamKey: "vpc-existing"},
		"vpc-abcd1234",
	)

	assert.Equal(t, "vpc-abcd1234", updated[cloudProviderNativeNetworkIDParamKey])
}

func TestApplyCloudProviderNativeNetworkIDParam_IgnoresEmptyMagicParam(t *testing.T) {
	updated := applyCloudProviderNativeNetworkIDParam(
		map[string]any{cloudProviderNativeNetworkIDParamKey: ""},
		"vpc-abcd1234",
	)

	assert.Equal(t, "vpc-abcd1234", updated[cloudProviderNativeNetworkIDParamKey])
}

func TestApplyCloudProviderNativeNetworkIDParam_AddsNativeNetworkID(t *testing.T) {
	updated := applyCloudProviderNativeNetworkIDParam(
		map[string]any{"existing": "value"},
		"vpc-abcd1234",
	)

	assert.Equal(t, "value", updated["existing"])
	assert.Equal(t, "vpc-abcd1234", updated[cloudProviderNativeNetworkIDParamKey])
}

func TestValidateCreateCloudTargetRequiresCloudProviderWithoutOnpremPlatform(t *testing.T) {
	err := validateCreateCloudTarget("", "us-east-2", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--cloud-provider")
	assert.Contains(t, err.Error(), "--onprem-platform")
}

func TestValidateCreateCloudTargetRequiresRegionWithoutOnpremPlatform(t *testing.T) {
	err := validateCreateCloudTarget("aws", "", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--region")
	assert.Contains(t, err.Error(), "--onprem-platform")
}

func TestValidateCreateCloudTargetAllowsMissingCloudProviderAndRegionWithOnpremPlatform(t *testing.T) {
	err := validateCreateCloudTarget("", "", "Generic")

	require.NoError(t, err)
}

func TestValidateCreateCloudTargetRejectsCloudProviderWithOnpremPlatform(t *testing.T) {
	err := validateCreateCloudTarget("byoc-onprem", "", "Generic")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--cloud-provider")
	assert.Contains(t, err.Error(), "--onprem-platform")
}

func TestValidateCreateCloudTargetRejectsRegionWithOnpremPlatform(t *testing.T) {
	err := validateCreateCloudTarget("", "us-east-2", "Generic")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--region")
	assert.Contains(t, err.Error(), "--onprem-platform")
}

func TestApplyCloudProviderToCreateRequest(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyCloudProviderToCreateRequest(&request, " aws ")

	require.NotNil(t, request.CloudProvider)
	assert.Equal(t, "aws", *request.CloudProvider)
}

func TestApplyCloudProviderToCreateRequestIgnoresEmptyValue(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyCloudProviderToCreateRequest(&request, " ")

	assert.Nil(t, request.CloudProvider)
}

func TestApplyRegionToCreateRequest(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyRegionToCreateRequest(&request, " us-east-2 ")

	require.NotNil(t, request.Region)
	assert.Equal(t, "us-east-2", *request.Region)
}

func TestApplyRegionToCreateRequestIgnoresEmptyValue(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyRegionToCreateRequest(&request, " ")

	assert.Nil(t, request.Region)
}

func TestApplyOnpremPlatformToCreateRequest(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyOnpremPlatformToCreateRequest(&request, " Generic ")

	require.NotNil(t, request.OnpremPlatform)
	assert.Equal(t, "Generic", *request.OnpremPlatform)
}

func TestApplyOnpremPlatformToCreateRequestIgnoresEmptyValue(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyOnpremPlatformToCreateRequest(&request, " ")

	assert.Nil(t, request.OnpremPlatform)
}

func TestApplyNetworkTypeToCreateRequest(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyNetworkTypeToCreateRequest(&request, " INTERNAL ")

	require.NotNil(t, request.NetworkType)
	assert.Equal(t, "INTERNAL", *request.NetworkType)
}

func TestApplyNetworkTypeToCreateRequestIgnoresEmptyValue(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}

	applyNetworkTypeToCreateRequest(&request, " ")

	assert.Nil(t, request.NetworkType)
}

func TestCreateCommandUse_IncludesNetworkType(t *testing.T) {
	assert.Contains(t, createCmd.Use, "--network-type")
}

func TestCreateRequestOmitsCloudProviderAndRegionWhenOnlyOnpremPlatformIsSet(t *testing.T) {
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{}
	applyCloudProviderToCreateRequest(&request, "")
	applyRegionToCreateRequest(&request, "")
	applyOnpremPlatformToCreateRequest(&request, "Generic")

	raw, err := json.Marshal(request)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, "Generic", payload["onprem_platform"])
	assert.NotContains(t, payload, "cloud_provider")
	assert.NotContains(t, payload, "region")
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

func TestSelectOfferingForEnvironment_PicksRequestedEnvironmentNotFirstOffering(t *testing.T) {
	offerings := []openapiclientfleet.ServiceOffering{
		{
			ServiceEnvironmentID:     "se-dev",
			ServiceEnvironmentName:   "Dev",
			ServiceEnvironmentURLKey: "dev",
			ProductTierID:            "pt-byoa",
		},
		{
			ServiceEnvironmentID:     "se-prod",
			ServiceEnvironmentName:   "Production",
			ServiceEnvironmentURLKey: "prod",
			ProductTierID:            "pt-byoa",
		},
	}

	offering, err := selectOfferingForEnvironment(offerings, "se-prod", "pt-byoa")

	require.NoError(t, err)
	assert.Equal(t, "prod", offering.ServiceEnvironmentURLKey)
}

func TestSelectOfferingForEnvironment_MatchesProductTierWithinEnvironment(t *testing.T) {
	offerings := []openapiclientfleet.ServiceOffering{
		{
			ServiceEnvironmentID:     "se-prod",
			ServiceEnvironmentURLKey: "prod",
			ProductTierID:            "pt-hosted",
			ProductTierURLKey:        "hosted",
		},
		{
			ServiceEnvironmentID:     "se-prod",
			ServiceEnvironmentURLKey: "prod",
			ProductTierID:            "pt-byoa",
			ProductTierURLKey:        "byoa",
		},
	}

	offering, err := selectOfferingForEnvironment(offerings, "se-prod", "pt-byoa")

	require.NoError(t, err)
	assert.Equal(t, "byoa", offering.ProductTierURLKey)
}

func TestSelectOfferingForEnvironment_ErrorsWhenEnvironmentMissing(t *testing.T) {
	offerings := []openapiclientfleet.ServiceOffering{
		{
			ServiceEnvironmentID:     "se-dev",
			ServiceEnvironmentURLKey: "dev",
			ProductTierID:            "pt-byoa",
		},
	}

	_, err := selectOfferingForEnvironment(offerings, "se-prod", "pt-byoa")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "se-prod")
}

func TestSelectOfferingForEnvironment_ErrorsOnEmptyOfferings(t *testing.T) {
	_, err := selectOfferingForEnvironment(nil, "se-prod", "pt-byoa")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no service offerings")
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

// --- Regression coverage: the resolved subscription ID must reach the create-instance
// request wire, not just the pure resolveInstanceSubscriptionID helper.
//
// TestResolveInstanceSubscriptionID* above prove resolveInstanceSubscriptionID itself is
// correct, but nothing exercised runCreate's wiring: `subscriptionID, err =
// resolveInstanceSubscriptionID(...)` at the call site in runCreate. If that were ever
// changed from `=` to `:=`, the reassignment would silently shadow the outer subscriptionID,
// --customer-email would stop doing anything, and every test above would still pass green.
// This test drives runCreate end-to-end against a stub HTTP server standing in for the
// Omnistrate API and asserts the resolved subscription ID reaches the create-instance request
// body.

// fakeJWT builds a syntactically valid, unsigned JWT whose exp claim is far enough in the
// future for config.IsTokenExpired to treat it as fresh. GetTokenWithLogin only decodes the
// exp claim locally to decide whether a refresh is needed; the signature itself is never
// verified client-side (that happens server-side), so a placeholder signature segment is fine
// for a stub server.
func fakeJWT(t *testing.T, exp time.Time) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(map[string]int64{"exp": exp.Unix()})
	require.NoError(t, err)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	return header + "." + payload + ".sig"
}

// startCreateInstanceTestServer points the SDK's v1 and fleet clients at a stub server for the
// duration of the test. Mirrors internal/dataaccess/subscription_test.go's
// startSubscriptionTestServer; duplicated locally because that helper is unexported across the
// package boundary.
func startCreateInstanceTestServer(t *testing.T, handler http.HandlerFunc) {
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

// v1ServiceOfferingFixture builds a fully populated v1 ServiceOffering, i.e. every property
// the SDK's generated UnmarshalJSON requires. Unlike the deliberately sparse
// serviceOfferingFixture above (which is only ever consumed in-process, never round-tripped
// through JSON), this one must survive an actual HTTP response decode.
func v1ServiceOfferingFixture(environmentID, productTierID, resourceID, resourceName string) openapiclientv1.ServiceOffering {
	return openapiclientv1.ServiceOffering{
		ProductTierDocumentation: "doc",
		ProductTierID:            productTierID,
		ProductTierName:          "test-plan",
		ProductTierPricing:       map[string]interface{}{},
		ProductTierSupport:       "community",
		ProductTierType:          "PAID",
		ProductTierURLKey:        "plan-key",
		ProductTierVersion:       "1.0",
		ResourceParameters: []openapiclientv1.ResourceEntity{
			{Name: resourceName, ResourceId: resourceID, UrlKey: "res-v1-key"},
		},
		ServiceAPIID:                 "api-test",
		ServiceAPIVersion:            "v1",
		ServiceEnvironmentID:         environmentID,
		ServiceEnvironmentName:       "prod",
		ServiceEnvironmentType:       "PROD",
		ServiceEnvironmentURLKey:     "env-key",
		ServiceEnvironmentVisibility: "PUBLIC",
		ServiceModelID:               "model-test",
		ServiceModelName:             "test-model",
		ServiceModelStatus:           "ACTIVE",
		ServiceModelType:             "STANDARD",
		ServiceModelURLKey:           "model-key",
	}
}

// externalDescribeServiceOfferingFixture wraps v1ServiceOfferingFixture in the response
// envelope ExternalDescribeServiceOffering (the v1 API resource resolution calls) expects.
func externalDescribeServiceOfferingFixture(serviceID string, offerings ...openapiclientv1.ServiceOffering) *openapiclientv1.DescribeServiceOfferingResult {
	return &openapiclientv1.DescribeServiceOfferingResult{
		ServiceId:           serviceID,
		ServiceName:         "test-service",
		ServiceOrgId:        "org-abc123",
		ServiceProviderId:   "sp-test",
		ServiceProviderName: "test-provider",
		ServiceURLKey:       "svc-url-key",
		Offerings:           offerings,
	}
}

// fleetServiceOfferingFixture builds a fully populated fleet ServiceOffering matching the
// environment/plan/resource the test resolves via getResource.
func fleetServiceOfferingFixture(environmentID, productTierID, resourceName, resourceURLKey string) openapiclientfleet.ServiceOffering {
	return openapiclientfleet.ServiceOffering{
		ProductTierDocumentation: "doc",
		ProductTierID:            productTierID,
		ProductTierName:          "test-plan",
		ProductTierPricing:       map[string]interface{}{},
		ProductTierSupport:       "community",
		ProductTierType:          "PAID",
		ProductTierURLKey:        "plan-key",
		ProductTierVersion:       "1.0",
		ResourceParameters: []openapiclientfleet.ResourceEntity{
			{Name: resourceName, ResourceId: "res-fleet-test", UrlKey: resourceURLKey},
		},
		ServiceAPIID:                 "api-test",
		ServiceAPIVersion:            "v1",
		ServiceEnvironmentID:         environmentID,
		ServiceEnvironmentName:       "prod",
		ServiceEnvironmentType:       "PROD",
		ServiceEnvironmentURLKey:     "env-key",
		ServiceEnvironmentVisibility: "PUBLIC",
		ServiceModelID:               "model-test",
		ServiceModelName:             "test-model",
		ServiceModelStatus:           "ACTIVE",
		ServiceModelType:             "STANDARD",
		ServiceModelURLKey:           "model-key",
	}
}

// fleetDescribeServiceOfferingFixture wraps fleetServiceOfferingFixture in the response
// envelope the fleet DescribeServiceOffering endpoint returns.
func fleetDescribeServiceOfferingFixture(serviceID string, offerings ...openapiclientfleet.ServiceOffering) *openapiclientfleet.InventoryDescribeServiceOfferingResult {
	return &openapiclientfleet.InventoryDescribeServiceOfferingResult{
		ConsumptionDescribeServiceOfferingResult: &openapiclientfleet.DescribeServiceOfferingResult{
			ServiceId:           serviceID,
			ServiceName:         "test-service",
			ServiceOrgId:        "org-abc123",
			ServiceProviderId:   "sp-test",
			ServiceProviderName: "test-provider",
			ServiceURLKey:       "svc-url-key",
			Offerings:           offerings,
		},
	}
}

// resourceInstanceFixture builds the minimal fleet ResourceInstance that satisfies the SDK's
// required-property validation, for the DescribeResourceInstance response after creation.
func resourceInstanceFixture(serviceID, environmentID, subscriptionID string) openapiclientfleet.ResourceInstance {
	return openapiclientfleet.ResourceInstance{
		CloudProvider:                     "aws",
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{},
		EnvironmentId:                     environmentID,
		InputParams:                       map[string]interface{}{},
		InstanceDebugCommands:             []string{},
		IntegrationsStatus:                []openapiclientfleet.IntegrationStatus{},
		OrganizationId:                    "org-abc123",
		OrganizationName:                  "test-org",
		ProductTierId:                     "pt-test",
		ProductTierName:                   "test-plan",
		ProductTierType:                   "PAID",
		ResourceVersionSummaries:          []openapiclientfleet.ResourceVersionSummary{},
		ServiceEnvName:                    "prod",
		ServiceId:                         serviceID,
		ServiceModelId:                    "model-test",
		ServiceModelName:                  "test-model",
		ServiceModelType:                  "STANDARD",
		ServiceName:                       "test-service",
		SubscriptionId:                    subscriptionID,
		SubscriptionOwnerName:             "test-customer",
		TierVersion:                       "1.0",
		TierVersionReleasedAt:             "2026-09-07T00:00:00Z",
		TierVersionReleasedByUserId:       "user-test",
		TierVersionReleasedByUserName:     "test-customer",
		TierVersionStatus:                 "ACTIVE",
	}
}

// newTestCreateCommand builds a *cobra.Command exposing exactly the flags runCreate reads.
// It intentionally does not reuse the package-level createCmd (avoiding shared mutable flag
// state across tests) and does not go through the real root command tree (cmd/root.go imports
// this package, so importing it back here would cycle); "output" in particular is normally a
// persistent flag inherited from RootCmd, so it is registered locally here instead.
func newTestCreateCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "create"}
	flags := cmd.Flags()
	flags.String("service", "", "")
	flags.String("environment", "", "")
	flags.String("plan", "", "")
	flags.String("version", "preferred", "")
	flags.String("resource", "", "")
	flags.String("cloud-provider", "", "")
	flags.String("region", "", "")
	flags.String("param", "", "")
	flags.String("param-file", "", "")
	flags.String("customer-account-id", "", "")
	flags.String("cloud-provider-native-network-id", "", "")
	flags.String("network-type", "", "")
	flags.String("onprem-platform", "", "")
	flags.String("tags", "", "")
	flags.String("breakpoints", "", "")
	flags.String("subscription-id", "", "")
	flags.String("customer-email", "", "")
	flags.String("instance-id", "", "")
	flags.Bool("wait", false, "")
	flags.String("output", "table", "")
	return cmd
}

func TestRunCreateWiresResolvedSubscriptionIDIntoCreateRequest(t *testing.T) {
	// Isolate the config directory so GetTokenWithLogin reads/writes a throwaway auth
	// config under a temp HOME, never the real user's ~/.omnistrate. go-homedir caches the
	// resolved home directory process-wide, so the cache must be reset after pointing HOME
	// at the temp dir (to pick it up) and again after HOME is restored (so later tests, and
	// the real environment, aren't left resolving into a deleted temp dir). t.Cleanup runs
	// LIFO, so registering the reset before t.Setenv makes it run *after* Setenv's own
	// restore-cleanup.
	t.Cleanup(func() { homedir.Reset() })
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()

	token := fakeJWT(t, time.Now().Add(time.Hour))
	require.NoError(t, config.CreateOrUpdateAuthConfig(config.AuthConfig{Token: token}))

	const (
		serviceID         = "s-test"
		environmentID     = "se-test"
		productTierID     = "pt-test"
		resourceName      = "myResource"
		resourceURLKey    = "myresource-key"
		wantSubscription  = "sub-test"
		createdInstanceID = "instance-test"
	)

	var (
		createRequestSeen bool
		createRequestBody map[string]any
	)

	startCreateInstanceTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2022-09-01-00/user":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "user-test"}))
		case r.Method == http.MethodGet && r.URL.Path == "/2022-09-01-00/service":
			require.NoError(t, json.NewEncoder(w).Encode(servicesFixture(
				serviceFixture(serviceID, "test-service", environmentID, "prod", productTierID, "test-plan"),
			)))
		case r.Method == http.MethodGet && r.URL.Path == "/2022-09-01-00/service-offering/"+serviceID:
			require.NoError(t, json.NewEncoder(w).Encode(externalDescribeServiceOfferingFixture(
				serviceID,
				v1ServiceOfferingFixture(environmentID, productTierID, "res-v1-test", resourceName),
			)))
		case r.Method == http.MethodGet && r.URL.Path == "/2022-09-01-00/fleet/service-offering/"+serviceID:
			require.NoError(t, json.NewEncoder(w).Encode(fleetDescribeServiceOfferingFixture(
				serviceID,
				fleetServiceOfferingFixture(environmentID, productTierID, resourceName, resourceURLKey),
			)))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/2022-09-01-00/fleet/resource-instance/"):
			createRequestSeen = true
			require.NoError(t, json.NewDecoder(r.Body).Decode(&createRequestBody))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": createdInstanceID}))
		case r.Method == http.MethodGet && r.URL.Path == "/2022-09-01-00/fleet/service/"+serviceID+"/environment/"+environmentID+"/instance/"+createdInstanceID:
			require.NoError(t, json.NewEncoder(w).Encode(resourceInstanceFixture(serviceID, environmentID, wantSubscription)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	// Stub only the customer-email-to-subscription lookup (already covered by
	// TestResolveInstanceSubscriptionID* above) so this test's HTTP surface stays limited to
	// what runCreate itself talks to.
	original := resolveInstanceSubscriptionByEmail
	t.Cleanup(func() { resolveInstanceSubscriptionByEmail = original })
	resolveInstanceSubscriptionByEmail = func(context.Context, string, dataaccess.CustomerSubscriptionLookup) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		return &openapiclientfleet.FleetDescribeSubscriptionResult{Id: wantSubscription}, nil
	}

	cmd := newTestCreateCommand()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set("service", serviceID))
	require.NoError(t, cmd.Flags().Set("environment", environmentID))
	require.NoError(t, cmd.Flags().Set("plan", productTierID))
	require.NoError(t, cmd.Flags().Set("resource", resourceName))
	require.NoError(t, cmd.Flags().Set("cloud-provider", "aws"))
	require.NoError(t, cmd.Flags().Set("region", "us-east-2"))
	require.NoError(t, cmd.Flags().Set("customer-email", "customer@example.com"))
	require.NoError(t, cmd.Flags().Set("output", "json"))

	err := runCreate(cmd, nil)
	require.NoError(t, err)

	require.True(t, createRequestSeen, "expected the create-resource-instance request to reach the stub server")
	assert.Equal(t, wantSubscription, createRequestBody["subscriptionId"],
		"the subscription resolved from --customer-email must reach the create-instance request body")
}
