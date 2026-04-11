package account

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerCreateCommandDoesNotExposePlacementFlags(t *testing.T) {
	require.Nil(t, customerCreateCmd.Flags().Lookup("cloud-provider"))
	require.Nil(t, customerCreateCmd.Flags().Lookup("region"))
	require.Nil(t, customerCreateCmd.Flags().Lookup("version"))
	require.NoError(t, customerCreateCmd.Args(customerCreateCmd, []string{}))
	require.Error(t, customerCreateCmd.Args(customerCreateCmd, []string{"unexpected-name"}))
	require.NotNil(t, customerCreateCmd.Flags().Lookup("service"))
	require.NotNil(t, customerCreateCmd.Flags().Lookup("environment"))
	require.NotNil(t, customerCreateCmd.Flags().Lookup("plan"))
	require.NotNil(t, customerCreateCmd.Flags().Lookup(customerEmailFlag))
}

func TestCloudAccountParamsFromFlags_SharesProviderParsing(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	require.NoError(t, os.WriteFile(privateKeyPath, []byte("PEM DATA"), 0600))

	bindingsPath := filepath.Join(tempDir, "bindings.yaml")
	require.NoError(t, os.WriteFile(bindingsPath, []byte(`
bindings:
  - projectId: project-1
    serviceAccountId: service-account-1
    publicKeyId: public-key-1
    privateKeyPEMFile: private.pem
`), 0600))

	cmd := &cobra.Command{}
	addCloudAccountProviderFlags(cmd)
	require.NoError(t, cmd.Flags().Set(nebiusTenantIDFlag, "tenant-1"))
	require.NoError(t, cmd.Flags().Set(nebiusBindingsFileFlag, bindingsPath))

	params, err := cloudAccountParamsFromFlags(cmd, "customer-nebius")
	require.NoError(t, err)
	require.Equal(t, "customer-nebius", params.Name)
	require.Equal(t, "tenant-1", params.NebiusTenantID)
	require.Len(t, params.NebiusBindings, 1)
	require.Equal(t, "project-1", params.NebiusBindings[0].ProjectID)
	require.Equal(t, "PEM DATA", params.NebiusBindings[0].PrivateKeyPEM)
}

func TestFindCustomerAccountResource(t *testing.T) {
	resources := []openapiclient.ResourceEntity{
		{
			Name:       "Primary",
			ResourceId: "r-primary",
			UrlKey:     "primary",
		},
		{
			Name:       customerAccountResourceName,
			ResourceId: "r-injectedaccountconfigpt123",
			UrlKey:     customerAccountResourceKey,
		},
	}

	resource, err := findCustomerAccountResource(resources)
	require.NoError(t, err)
	require.NotNil(t, resource)
	assert.Equal(t, customerAccountResourceKey, resource.UrlKey)
	assert.Equal(t, "r-injectedaccountconfigpt123", resource.ResourceId)
}

func TestBuildCustomerAccountRequestParamsWithDerivedValues_Nebius(t *testing.T) {
	params := CloudAccountParams{
		Name:           "customer-nebius",
		NebiusTenantID: "tenant-1",
		NebiusBindings: []openapiclient.NebiusAccountBindingInput{
			{
				ProjectID:        "project-1",
				ServiceAccountID: "service-account-1",
				PublicKeyID:      "public-key-1",
				PrivateKeyPEM:    "pem-data",
			},
		},
	}

	inputParameters := []openapiclient.DescribeInputParameterResult{
		{Name: customerAccountNebiusTenantIDName, Key: "nebius_tenant_id"},
		{Name: customerAccountNebiusBindingsName, Key: "nebius_bindings"},
	}

	requestParams, err := buildCustomerAccountRequestParamsWithDerivedValues(params, inputParameters, "")
	require.NoError(t, err)
	require.Len(t, requestParams, 2)
	assert.Equal(t, "tenant-1", requestParams["nebius_tenant_id"])

	bindings, ok := requestParams["nebius_bindings"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, bindings, 1)
	assert.Equal(t, map[string]any{
		"projectID":        "project-1",
		"serviceAccountID": "service-account-1",
		"publicKeyID":      "public-key-1",
		"privateKeyPEM":    "pem-data",
	}, bindings[0])

	_, hasCloudProvider := requestParams["cloud_provider"]
	_, hasRegion := requestParams["region"]
	assert.False(t, hasCloudProvider)
	assert.False(t, hasRegion)
}

func TestBuildCustomerAccountRequestParamsWithDerivedValues_AWS(t *testing.T) {
	params := CloudAccountParams{
		Name:         "customer-aws",
		AwsAccountID: "123456789012",
	}

	inputParameters := []openapiclient.DescribeInputParameterResult{
		{Name: customerAccountIacToolName, Key: "account_configuration_method"},
		{Name: customerAccountAWSAccountIDName, Key: "aws_account_id"},
		{Name: customerAccountAWSBootstrapRoleName, Key: "aws_bootstrap_role_arn"},
	}

	requestParams, err := buildCustomerAccountRequestParamsWithDerivedValues(params, inputParameters, "")
	require.NoError(t, err)
	assert.Equal(t, "CloudFormation", requestParams["account_configuration_method"])
	assert.Equal(t, "123456789012", requestParams["aws_account_id"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/omnistrate-bootstrap-role", requestParams["aws_bootstrap_role_arn"])
}

func TestCustomerAccountInputParameters(t *testing.T) {
	parameters := customerAccountInputParameters()
	keysByName := map[string]string{}
	for _, parameter := range parameters {
		keysByName[parameter.Name] = parameter.Key
	}

	assert.Equal(t, "account_configuration_method", keysByName[customerAccountIacToolName])
	assert.Equal(t, "aws_account_id", keysByName[customerAccountAWSAccountIDName])
	assert.Equal(t, "aws_bootstrap_role_arn", keysByName[customerAccountAWSBootstrapRoleName])
	assert.Equal(t, "gcp_project_id", keysByName[customerAccountGCPProjectIDName])
	assert.Equal(t, "gcp_project_number", keysByName[customerAccountGCPProjectNumberName])
	assert.Equal(t, "gcp_service_account_email", keysByName[customerAccountGCPServiceAccountName])
	assert.Equal(t, "azure_subscription_id", keysByName[customerAccountAzureSubIDName])
	assert.Equal(t, "azure_tenant_id", keysByName[customerAccountAzureTenantIDName])
	assert.Equal(t, "nebius_tenant_id", keysByName[customerAccountNebiusTenantIDName])
	assert.Equal(t, "nebius_bindings", keysByName[customerAccountNebiusBindingsName])
}

func TestExtractCustomerAccountConfigID(t *testing.T) {
	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			ResultParams: map[string]any{
				customerAccountResultAccountIDKey: "ac-123",
			},
		},
	}

	assert.Equal(t, "ac-123", extractCustomerAccountConfigID(instance))
	assert.Equal(t, "", extractCustomerAccountConfigID(nil))
}

func TestResolveCustomerAccountSubscription_NonProductionUsesProvidedSubscription(t *testing.T) {
	target := &customerAccountTarget{
		ServiceID:       "svc-1",
		EnvironmentID:   "env-1",
		EnvironmentType: "DEV",
		ProductTierID:   "pt-1",
	}

	originalLookup := resolveCustomerSubscriptionByEmail
	originalDescribeCurrentUser := describeCurrentUserFn
	t.Cleanup(func() {
		resolveCustomerSubscriptionByEmail = originalLookup
		describeCurrentUserFn = originalDescribeCurrentUser
	})

	resolveCustomerSubscriptionByEmail = func(context.Context, string, string, string, string, string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		t.Fatal("subscription lookup should not be called")
		return nil, nil
	}
	describeCurrentUserFn = func(context.Context, string) (*openapiclient.DescribeUserResult, error) {
		t.Fatal("describe user should not be called")
		return nil, nil
	}

	subscriptionID, err := resolveCustomerAccountSubscription(
		context.Background(),
		"token",
		target,
		"sub-123",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "sub-123", subscriptionID)
}

func TestResolveCustomerAccountSubscription_ProductionDefaultsToCallingUserSubscription(t *testing.T) {
	target := &customerAccountTarget{
		ServiceID:       "svc-1",
		ServiceName:     "postgres",
		EnvironmentID:   "env-1",
		EnvironmentName: "prod",
		EnvironmentType: "PROD",
		ProductTierID:   "pt-1",
		ProductTierName: "customer-hosted",
	}

	originalLookup := resolveCustomerSubscriptionByEmail
	originalDescribeCurrentUser := describeCurrentUserFn
	t.Cleanup(func() {
		resolveCustomerSubscriptionByEmail = originalLookup
		describeCurrentUserFn = originalDescribeCurrentUser
	})

	describeCurrentUserFn = func(ctx context.Context, token string) (*openapiclient.DescribeUserResult, error) {
		assert.Equal(t, "token", token)
		return &openapiclient.DescribeUserResult{
			Email: strPtr("caller@example.com"),
		}, nil
	}
	resolveCustomerSubscriptionByEmail = func(ctx context.Context, token, serviceID, environmentID, planID, customerEmail string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		assert.Equal(t, "token", token)
		assert.Equal(t, "svc-1", serviceID)
		assert.Equal(t, "env-1", environmentID)
		assert.Equal(t, "pt-1", planID)
		assert.Equal(t, "caller@example.com", customerEmail)
		return &openapiclientfleet.FleetDescribeSubscriptionResult{Id: "sub-456"}, nil
	}

	subscriptionID, err := resolveCustomerAccountSubscription(
		context.Background(),
		"token",
		target,
		"",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "sub-456", subscriptionID)
}

func TestResolveCustomerAccountSubscription_ProductionFailsWhenCallingUserEmailIsUnavailable(t *testing.T) {
	target := &customerAccountTarget{
		EnvironmentName: "prod",
		EnvironmentType: "PROD",
	}

	originalLookup := resolveCustomerSubscriptionByEmail
	originalDescribeCurrentUser := describeCurrentUserFn
	t.Cleanup(func() {
		resolveCustomerSubscriptionByEmail = originalLookup
		describeCurrentUserFn = originalDescribeCurrentUser
	})

	resolveCustomerSubscriptionByEmail = func(context.Context, string, string, string, string, string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		t.Fatal("subscription lookup should not be called")
		return nil, nil
	}
	describeCurrentUserFn = func(context.Context, string) (*openapiclient.DescribeUserResult, error) {
		return &openapiclient.DescribeUserResult{}, nil
	}

	_, err := resolveCustomerAccountSubscription(context.Background(), "token", target, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "current user email is unavailable")
}

func TestResolveCustomerAccountSubscription_RejectsConflictingFlags(t *testing.T) {
	target := &customerAccountTarget{
		EnvironmentType: "DEV",
	}

	_, err := resolveCustomerAccountSubscription(
		context.Background(),
		"token",
		target,
		"sub-123",
		"customer@example.com",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both --customer-email and --subscription-id")
}

func TestResolveCustomerAccountSubscription_ProductionAllowsSubscriptionIDOverride(t *testing.T) {
	target := &customerAccountTarget{
		EnvironmentType: "PRODUCTION",
	}

	originalLookup := resolveCustomerSubscriptionByEmail
	originalDescribeCurrentUser := describeCurrentUserFn
	t.Cleanup(func() {
		resolveCustomerSubscriptionByEmail = originalLookup
		describeCurrentUserFn = originalDescribeCurrentUser
	})

	resolveCustomerSubscriptionByEmail = func(context.Context, string, string, string, string, string) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
		t.Fatal("subscription lookup should not be called")
		return nil, nil
	}
	describeCurrentUserFn = func(context.Context, string) (*openapiclient.DescribeUserResult, error) {
		t.Fatal("describe user should not be called")
		return nil, nil
	}

	subscriptionID, err := resolveCustomerAccountSubscription(
		context.Background(),
		"token",
		target,
		"sub-123",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "sub-123", subscriptionID)
}
