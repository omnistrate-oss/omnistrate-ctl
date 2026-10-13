package dataaccess

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateResourceInstanceWithNonexistentCustomNetwork(t *testing.T) {
	testutils.IntegrationTest(t)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(t, err)
	login, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
	require.NoError(t, err)
	require.NotEmpty(t, login.JWTToken)
	token := login.JWTToken

	// Enable CUSTOM_NETWORKS so the request reaches the network lookup instead
	// of failing because the service model does not support custom networks.
	spec, err := os.ReadFile("testfiles/instance_custom_network.yaml")
	require.NoError(t, err)
	build, err := dataaccess.BuildServiceFromComposeSpec(ctx, token, openapiclientv1.BuildServiceFromComposeSpecRequest2{
		Name:               "instance-network-" + testutils.RandomTestSuffix(),
		Environment:        utils.ToPtr("dev"),
		EnvironmentType:    utils.ToPtr("DEV"),
		FileContent:        base64.StdEncoding.EncodeToString(spec),
		Release:            utils.ToPtr(true),
		ReleaseAsPreferred: utils.ToPtr(true),
	})
	require.NoError(t, err)
	require.NotNil(t, build)
	require.NotEmpty(t, build.ServiceID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		assert.NoError(t, dataaccess.DeleteService(cleanupCtx, token, build.ServiceID))
	})

	var serviceOffering *openapiclientfleet.DescribeServiceOfferingResult
	var offering openapiclientfleet.ServiceOffering
	var resource openapiclientfleet.ResourceEntity
	// The fleet inventory can lag behind the build. Resolve the actual target
	// before asserting NotFound, so a missing service or resource cannot pass.
	require.Eventually(t, func() bool {
		result, describeErr := dataaccess.DescribeServiceOffering(ctx, token, build.ServiceID, build.ProductTierID, "")
		if describeErr != nil || result == nil || result.ConsumptionDescribeServiceOfferingResult == nil {
			return false
		}
		serviceOffering = result.ConsumptionDescribeServiceOfferingResult
		for _, candidate := range serviceOffering.Offerings {
			if candidate.ServiceEnvironmentID != build.ServiceEnvironmentID || candidate.ProductTierID != build.ProductTierID {
				continue
			}
			for _, candidateResource := range candidate.ResourceParameters {
				if strings.EqualFold(candidateResource.Name, "web") && candidateResource.UrlKey != "" {
					offering = candidate
					resource = candidateResource
					return true
				}
			}
		}
		return false
	}, 2*time.Minute, 3*time.Second, "the built service plan and web resource must be available in fleet inventory")

	const customNetworkID = "n-123456789"
	result, err := dataaccess.CreateResourceInstance(ctx, token,
		serviceOffering.ServiceProviderId,
		serviceOffering.ServiceURLKey,
		offering.ServiceAPIVersion,
		offering.ServiceEnvironmentURLKey,
		offering.ServiceModelURLKey,
		offering.ProductTierURLKey,
		resource.UrlKey,
		openapiclientfleet.FleetCreateResourceInstanceRequest2{
			CloudProvider:   utils.ToPtr("aws"),
			Region:          utils.ToPtr("ca-central-1"),
			CustomNetworkId: utils.ToPtr(customNetworkID),
			RequestParams:   map[string]any{},
		})
	// If a regression accepts the request, clean up the unexpected instance.
	if result != nil && result.GetId() != "" {
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
			defer cleanupCancel()
			assert.NoError(t, dataaccess.DeleteResourceInstance(cleanupCtx, token,
				build.ServiceID, build.ServiceEnvironmentID, resource.ResourceId, result.GetId(), true))
		})
	}
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Regexp(t, `(?i)(not[_ ]?found|does not exist|invalid custom network)`, err.Error())
	assert.Contains(t, strings.ToLower(err.Error()), "network")
	assert.Contains(t, err.Error(), customNetworkID)
}
