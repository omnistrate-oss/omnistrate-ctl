package instance

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/assert"
)

func TestDashboardCommand(t *testing.T) {
	assert.NotNil(t, dashboardCmd)
	assert.Equal(t, "dashboard [instance-id]", dashboardCmd.Use)
	assert.Contains(t, dashboardCmd.Short, "Grafana dashboard")
}

func TestDashboardCommandFlags(t *testing.T) {
	assert.NotNil(t, dashboardCmd.Flag("output"))
	assert.Equal(t, "text", dashboardCmd.Flag("output").DefValue)
}

func TestRenderDashboardSnapshot(t *testing.T) {
	catalog := &dataaccess.DashboardCatalog{
		InstanceID:          "instance-123",
		PreferredFeatureKey: "METRICS",
		Features: []dataaccess.DashboardFeatureInfo{
			{ // #nosec -- test data
				Key:                 "METRICS",
				Label:               "Customer",
				GrafanaEndpoint:     "https://grafana.example.com",
				GrafanaUIUsername:   "customer-org",
				GrafanaUIPassword:   "customer-secret",
				ServiceAccountName:  "sa-instance-123",
				ServiceAccountToken: "glsa_example_token",
				Dashboards: []dataaccess.DashboardRef{
					{Name: "overview", Description: "Overview", URL: "https://grafana.example.com/d/overview"},
				},
			},
			{
				Key:               "METRICS#INTERNAL",
				Label:             "Internal",
				GrafanaEndpoint:   "https://grafana.internal.example.com",
				GrafanaUIUsername: "sp-org",
				GrafanaUIPassword: "sp-secret",
				Dashboards: []dataaccess.DashboardRef{
					{Name: "networking", Description: "Networking", URL: "https://grafana.internal.example.com/d/networking"},
				},
			},
		},
	}

	snapshot := renderDashboardSnapshot(catalog)
	for _, expected := range []string{
		"Instance Dashboard",
		"instance-123",
		"Customer (METRICS)",
		"Internal (METRICS#INTERNAL)",
		"overview",
		"networking",
		"Grafana UI",
		"Username: customer-org",
		"Grafana API",
		"Service account token: glsa_example_token",
	} {
		assert.Contains(t, snapshot, expected)
	}
}
