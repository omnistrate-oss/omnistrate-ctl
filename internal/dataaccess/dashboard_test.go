package dataaccess

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestDashboardServiceGetDashboardCatalogIncludesCustomerAndInternalMetrics(t *testing.T) {
	instanceID := "instance-123"
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id: &instanceID,
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{ // #nosec -- test data
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.example.com",
					"instanceOrgId":              "customer-org",
					"instanceOrgPassword":        "customer-value",
					"serviceProviderOrgId":       "sp-customer-org",
					"serviceProviderOrgPassword": "sp-customer-value",
					"serviceAccountUsername":     "sa-instance-123",
					"serviceAccountPassword":     "glsa_example_value",
					"dashboards": map[string]interface{}{
						"overview": map[string]interface{}{
							"description":   "Overview",
							"dashboardLink": "https://grafana.example.com/d/overview",
						},
					},
					"additionalMetrics": map[string]interface{}{
						"vllm": map[string]interface{}{
							"dashboards": map[string]interface{}{
								"gpu": map[string]interface{}{
									"title": "NVIDIA GPU",
								},
							},
						},
					},
				},
				internalMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.internal.example.com",
					"serviceProviderOrgId":       "sp-org",
					"serviceProviderOrgPassword": "sp-value",
					"dashboards": map[string]interface{}{
						"networking": map[string]interface{}{
							"description":   "Networking",
							"dashboardLink": "https://grafana.internal.example.com/d/networking",
						},
					},
				},
			},
		},
	}

	catalog, err := service.GetDashboardCatalog(instance)
	require.NoError(t, err)
	require.Equal(t, instanceID, catalog.InstanceID)
	require.Equal(t, customerMetricsFeatureKey, catalog.PreferredFeatureKey)
	require.Len(t, catalog.Features, 2)

	customer := catalog.Features[0]
	require.Equal(t, customerMetricsFeatureKey, customer.Key)
	require.Equal(t, "Customer", customer.Label)
	require.Equal(t, "https://grafana.example.com", customer.GrafanaEndpoint)
	require.Equal(t, "sp-customer-org", customer.GrafanaUIUsername)
	require.Equal(t, "sp-customer-value", customer.GrafanaUIPassword)
	require.Equal(t, "provider", customer.GrafanaUILoginScope)
	require.Equal(t, "sa-instance-123", customer.ServiceAccountName)
	require.Equal(t, "glsa_example_value", customer.ServiceAccountToken)
	require.Len(t, customer.Dashboards, 1)
	require.Equal(t, DashboardRef{Name: "overview", Description: "Overview", URL: "https://grafana.example.com/d/overview"}, customer.Dashboards[0])
	require.Len(t, customer.DashboardDefinitions, 1)
	require.Equal(t, DashboardDefinition{Source: "vllm", Name: "gpu", Title: "NVIDIA GPU"}, customer.DashboardDefinitions[0])

	internal := catalog.Features[1]
	require.Equal(t, internalMetricsFeatureKey, internal.Key)
	require.Equal(t, "Internal", internal.Label)
	require.Equal(t, "https://grafana.internal.example.com", internal.GrafanaEndpoint)
	require.Equal(t, "sp-org", internal.GrafanaUIUsername)
	require.Equal(t, "sp-value", internal.GrafanaUIPassword)
	require.Equal(t, "provider", internal.GrafanaUILoginScope)
	require.Len(t, internal.Dashboards, 1)
	require.Equal(t, DashboardRef{Name: "networking", Description: "Networking", URL: "https://grafana.internal.example.com/d/networking"}, internal.Dashboards[0])
}

func TestDashboardServiceGetDashboardInfoPrefersCustomerMetrics(t *testing.T) {
	instanceID := "instance-456"
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id: &instanceID,
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.example.com",
					"instanceOrgId":              "customer-org",
					"instanceOrgPassword":        "customer-value",
					"serviceProviderOrgId":       "sp-customer-org",
					"serviceProviderOrgPassword": "sp-customer-value",
				},
				internalMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.internal.example.com",
					"serviceProviderOrgId":       "sp-org",
					"serviceProviderOrgPassword": "sp-value",
				},
			},
		},
	}

	info, err := service.GetDashboardInfo(instance)
	require.NoError(t, err)
	require.Equal(t, instanceID, info.InstanceID)
	require.Equal(t, customerMetricsFeatureKey, info.MetricsFeatureKey)
	require.Equal(t, "Customer", info.MetricsFeatureLabel)
	require.Equal(t, "sp-customer-org", info.GrafanaLoginUsername)
	require.Equal(t, "sp-customer-value", info.GrafanaLoginPassword)
	require.Equal(t, "provider", info.GrafanaLoginScope)
}

func TestDashboardServiceGetDashboardInfoPrefersServiceProviderCredentials(t *testing.T) {
	instanceID := "instance-789"
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id: &instanceID,
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.example.com",
					"instanceOrgId":              "stale-org",
					"instanceOrgPassword":        "stale-value",
					"serviceProviderOrgId":       "provider-org",
					"serviceProviderOrgPassword": "provider-value",
					"auth": map[string]interface{}{
						"username": "auth-user",
						"password": "auth-value",
					},
				},
			},
		},
	}

	info, err := service.GetDashboardInfo(instance)
	require.NoError(t, err)
	require.Equal(t, "provider-org", info.GrafanaLoginUsername)
	require.Equal(t, "provider-value", info.GrafanaLoginPassword)
	require.Equal(t, "provider", info.GrafanaLoginScope)
}

func TestDashboardServiceGetDashboardCatalogSharesProviderCredentialsForSameGrafanaEndpoint(t *testing.T) {
	instanceID := "instance-shared"
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id: &instanceID,
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":             true,
					"grafanaEndpoint":     "https://grafana.example.com/",
					"instanceOrgId":       "customer-org",
					"instanceOrgPassword": "customer-value",
					"dashboards": map[string]interface{}{
						"overview": map[string]interface{}{
							"description":   "Overview",
							"dashboardLink": "https://grafana.example.com/d/overview",
						},
					},
				},
				internalMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.example.com",
					"serviceProviderOrgId":       "provider-org",
					"serviceProviderOrgPassword": "provider-value",
					"dashboards": map[string]interface{}{
						"networking": map[string]interface{}{
							"description":   "Networking",
							"dashboardLink": "https://grafana.example.com/d/networking",
						},
					},
				},
			},
		},
	}

	catalog, err := service.GetDashboardCatalog(instance)
	require.NoError(t, err)
	require.Len(t, catalog.Features, 2)

	customer := catalog.Features[0]
	require.Equal(t, customerMetricsFeatureKey, customer.Key)
	require.Equal(t, "provider-org", customer.GrafanaUIUsername)
	require.Equal(t, "provider-value", customer.GrafanaUIPassword)
	require.Equal(t, "provider", customer.GrafanaUILoginScope)

	internal := catalog.Features[1]
	require.Equal(t, internalMetricsFeatureKey, internal.Key)
	require.Equal(t, "provider-org", internal.GrafanaUIUsername)
	require.Equal(t, "provider-value", internal.GrafanaUIPassword)
	require.Equal(t, "provider", internal.GrafanaUILoginScope)
}

func TestDashboardServiceGetDashboardCatalogDoesNotShareProviderCredentialsAcrossGrafanaEndpoints(t *testing.T) {
	instanceID := "instance-separate"
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			Id: &instanceID,
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":             true,
					"grafanaEndpoint":     "https://grafana.customer.example.com",
					"instanceOrgId":       "customer-org",
					"instanceOrgPassword": "customer-value",
				},
				internalMetricsFeatureKey: map[string]interface{}{
					"enabled":                    true,
					"grafanaEndpoint":            "https://grafana.provider.example.com",
					"serviceProviderOrgId":       "provider-org",
					"serviceProviderOrgPassword": "provider-value",
				},
			},
		},
	}

	catalog, err := service.GetDashboardCatalog(instance)
	require.NoError(t, err)
	require.Len(t, catalog.Features, 2)

	customer := catalog.Features[0]
	require.Equal(t, customerMetricsFeatureKey, customer.Key)
	require.Equal(t, "customer-org", customer.GrafanaUIUsername)
	require.Equal(t, "customer-value", customer.GrafanaUIPassword)
	require.Equal(t, "customer", customer.GrafanaUILoginScope)
}

func TestDashboardServiceGetDashboardInfoDerivesEndpointFromDashboardLink(t *testing.T) {
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":             true,
					"instanceOrgId":       "customer-org",
					"instanceOrgPassword": "customer-value",
					"dashboards": map[string]interface{}{
						"overview": map[string]interface{}{
							"description":   "Overview",
							"dashboardLink": "https://grafana.example.com/d/overview",
						},
					},
				},
			},
		},
	}

	info, err := service.GetDashboardInfo(instance)
	require.NoError(t, err)
	require.Equal(t, "https://grafana.example.com", info.GrafanaEndpoint)
	require.Len(t, info.Dashboards, 1)
	require.Equal(t, "overview", info.Dashboards[0].Name)
}

func TestDashboardServiceGetDashboardInfoDisabled(t *testing.T) {
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled": false,
				},
			},
		},
	}

	_, err := service.GetDashboardInfo(instance)
	require.ErrorContains(t, err, "METRICS is disabled")
}

func TestDashboardServiceGetDashboardInfoMissingCredentials(t *testing.T) {
	service := NewDashboardService()

	instance := &openapiclientfleet.ResourceInstance{
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			ProductTierFeatures: map[string]interface{}{
				customerMetricsFeatureKey: map[string]interface{}{
					"enabled":         true,
					"grafanaEndpoint": "https://grafana.example.com",
				},
			},
		},
	}

	_, err := service.GetDashboardInfo(instance)
	require.ErrorContains(t, err, "METRICS is enabled for this instance, but Grafana access credentials are not available")
}
