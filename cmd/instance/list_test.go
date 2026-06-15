package instance

import (
	"context"
	"errors"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/require"
)

func TestFormatListedInstance(t *testing.T) {
	instanceID := "instance-123"
	region := "us-east-1"
	status := "RUNNING"
	tierVersion := "v1.2.3"
	resourceName := "postgres"

	instance := &openapiclientfleet.ResourceInstance{
		CloudProvider:   "aws",
		ProductTierName: "prod",
		ServiceEnvName:  "Production",
		ServiceName:     "PostgreSQL",
		SubscriptionId:  "sub-123",
		TierVersion:     "v1.2.2",
		ResourceVersionSummaries: []openapiclientfleet.ResourceVersionSummary{
			{ResourceName: &resourceName},
		},
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id:             &instanceID,
			Region:         &region,
			Status:         &status,
			TierVersion:    &tierVersion,
			SubscriptionId: listStringPtr("sub-override"),
			CustomTags:     []openapiclientfleet.CustomTag{{Key: "env", Value: "prod"}},
		},
	}

	formatted := formatListedInstance(instance, false)

	require.Equal(t, "instance-123", formatted.InstanceID)
	require.Equal(t, "PostgreSQL", formatted.Service)
	require.Equal(t, "Production", formatted.Environment)
	require.Equal(t, "prod", formatted.Plan)
	require.Equal(t, "v1.2.3", formatted.Version)
	require.Equal(t, "postgres", formatted.Resource)
	require.Equal(t, "aws", formatted.CloudProvider)
	require.Equal(t, "us-east-1", formatted.Region)
	require.Equal(t, "RUNNING", formatted.Status)
	require.Equal(t, "sub-override", formatted.SubscriptionID)
	require.Equal(t, "env=prod", formatted.Tags)
}

func TestFetchListedInstancesUsesFleetListForEachServiceEnvironment(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-1", "s-2"}}, nil
	}
	listEnvironments := func(_ context.Context, _ string, serviceID string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		switch serviceID {
		case "s-1":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-1", "se-2"}}, nil
		case "s-2":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-3"}}, nil
		default:
			t.Fatalf("unexpected service ID %q", serviceID)
			return nil, nil
		}
	}

	var calls []string
	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, options *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		require.NotNil(t, options)
		require.NotNil(t, options.Filter)
		require.Equal(t, "excludeCloudAccounts", *options.Filter)
		require.NotNil(t, options.ExcludeDetail)
		require.True(t, *options.ExcludeDetail)
		require.NotNil(t, options.ExcludeNetworkTopology)
		require.True(t, *options.ExcludeNetworkTopology)
		require.NotNil(t, options.ExcludeHAStatus)
		require.True(t, *options.ExcludeHAStatus)
		require.NotNil(t, options.ExcludeIntegrations)
		require.True(t, *options.ExcludeIntegrations)
		require.NotNil(t, options.ExcludeMaintenanceTasks)
		require.True(t, *options.ExcludeMaintenanceTasks)

		calls = append(calls, serviceID+"/"+environmentID)
		instanceID := serviceID + "-" + environmentID + "-instance"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Equal(t, []string{"s-1/se-1", "s-1/se-2", "s-2/se-3"}, calls)
	require.Len(t, instances, 3)
	require.Equal(t, "s-1-se-1-instance", instances[0].ConsumptionResourceInstanceResult.GetId())
	require.Equal(t, "s-2-se-3-instance", instances[2].ConsumptionResourceInstanceResult.GetId())
}

func TestFetchListedInstancesSkipsHostClusterNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-1", "s-2"}}, nil
	}
	listEnvironments := func(_ context.Context, _ string, serviceID string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		switch serviceID {
		case "s-1":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-broken"}}, nil
		case "s-2":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-valid"}}, nil
		default:
			t.Fatalf("unexpected service ID %q", serviceID)
			return nil, nil
		}
	}

	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, _ *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		if serviceID == "s-1" && environmentID == "se-broken" {
			return nil, errors.New("not_found\nDetail: Invalid request: failed to describe resource instance: failed to query host cluster: host cluster not found: record not found")
		}

		instanceID := "instance-valid"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "instance-valid", instances[0].ConsumptionResourceInstanceResult.GetId())
}

func TestFetchListedInstancesSkipsDeletedServicesWhenListingEnvironments(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-deleted", "s-valid"}}, nil
	}
	listEnvironments := func(_ context.Context, _ string, serviceID string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		switch serviceID {
		case "s-deleted":
			return nil, errors.New("bad_request\nDetail: Invalid request: service not found")
		case "s-valid":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-valid"}}, nil
		default:
			t.Fatalf("unexpected service ID %q", serviceID)
			return nil, nil
		}
	}
	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, _ *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		instanceID := serviceID + "-" + environmentID + "-instance"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "s-valid-se-valid-instance", instances[0].ConsumptionResourceInstanceResult.GetId())
}

func TestFetchListedInstancesSkipsDeletedServicesWhenListingInstances(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-deleted", "s-valid"}}, nil
	}
	listEnvironments := func(context.Context, string, string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-valid"}}, nil
	}
	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, _ *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		if serviceID == "s-deleted" {
			return nil, errors.New("bad_request\nDetail: Invalid request: service not found")
		}

		instanceID := serviceID + "-" + environmentID + "-instance"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "s-valid-se-valid-instance", instances[0].ConsumptionResourceInstanceResult.GetId())
}

func TestFetchListedInstancesSkipsAuthFailuresWhenListingEnvironments(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-inaccessible", "s-valid"}}, nil
	}
	listEnvironments := func(_ context.Context, _ string, serviceID string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		switch serviceID {
		case "s-inaccessible":
			return nil, errors.New("auth_failure\nDetail: ")
		case "s-valid":
			return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-valid"}}, nil
		default:
			t.Fatalf("unexpected service ID %q", serviceID)
			return nil, nil
		}
	}
	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, _ *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		instanceID := serviceID + "-" + environmentID + "-instance"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "s-valid-se-valid-instance", instances[0].ConsumptionResourceInstanceResult.GetId())
}

func TestFetchListedInstancesSkipsAuthFailuresWhenListingInstances(t *testing.T) {
	ctx := context.Background()
	listServices := func(context.Context, string) (*openapiclientv1.ListServiceResult, error) {
		return &openapiclientv1.ListServiceResult{Ids: []string{"s-inaccessible", "s-valid"}}, nil
	}
	listEnvironments := func(context.Context, string, string) (*openapiclientv1.ListServiceEnvironmentsResult, error) {
		return &openapiclientv1.ListServiceEnvironmentsResult{Ids: []string{"se-valid"}}, nil
	}
	listInstances := func(_ context.Context, _ string, serviceID, environmentID string, _ *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error) {
		if serviceID == "s-inaccessible" {
			return nil, errors.New("auth_failure\nDetail: ")
		}

		instanceID := serviceID + "-" + environmentID + "-instance"
		return []openapiclientfleet.ResourceInstance{
			{
				ServiceName:    serviceID,
				ServiceEnvName: environmentID,
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					Id: &instanceID,
				},
			},
		}, nil
	}

	instances, err := fetchListedInstances(ctx, "token", listServices, listEnvironments, listInstances)

	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "s-valid-se-valid-instance", instances[0].ConsumptionResourceInstanceResult.GetId())
}

func listStringPtr(value string) *string {
	return &value
}
