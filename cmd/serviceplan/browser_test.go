package serviceplan

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	displaymodel "github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/require"
)

type fakeServicePlanBrowserLoader struct {
	details map[string]servicePlanEnvironmentDetails
	calls   []string
}

func (f *fakeServicePlanBrowserLoader) LoadEnvironmentDetails(_ context.Context, _ string, env servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error) {
	key := env.cacheKey()
	f.calls = append(f.calls, key)
	return f.details[key], nil
}

func loadSelectedTestDetails(t *testing.T, model servicePlanBrowserModel) servicePlanBrowserModel {
	t.Helper()

	cmd := model.selectedDetailsLoadCmd()
	require.NotNil(t, cmd)

	updated, nextCmd := model.Update(cmd())
	require.Nil(t, nextCmd)
	return updated.(servicePlanBrowserModel)
}

func TestBuildServicePlanBrowserCatalogGroupsPlanNamesByService(t *testing.T) {
	services := testBrowserServices()

	catalog := buildServicePlanBrowserCatalog(services, nil)

	require.Len(t, catalog.Services, 1)
	require.Equal(t, "Postgres", catalog.Services[0].Name)
	require.Len(t, catalog.Services[0].Plans, 1)
	require.Equal(t, "Standard", catalog.Services[0].Plans[0].Name)
	require.Len(t, catalog.Services[0].Plans[0].Environments, 2)
	require.Equal(t, "dev", catalog.Services[0].Plans[0].Environments[0].Name)
	require.Equal(t, "prod", catalog.Services[0].Plans[0].Environments[1].Name)
}

func TestBuildServicePlanBrowserCatalogAppliesFilters(t *testing.T) {
	services := testBrowserServices()

	catalog := buildServicePlanBrowserCatalog(services, []map[string]string{{"environment": "prod"}})

	require.Len(t, catalog.Services, 1)
	require.Len(t, catalog.Services[0].Plans, 1)
	require.Len(t, catalog.Services[0].Plans[0].Environments, 1)
	require.Equal(t, "prod", catalog.Services[0].Plans[0].Environments[0].Name)
}

func TestServicePlanBrowserAccordionAndTabSwitching(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)

	require.Len(t, model.leftItems(), 2)
	require.Len(t, loader.calls, 1)

	model.list.Select(0)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Len(t, model.leftItems(), 1)

	model.list.Select(0)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(servicePlanBrowserModel)
	require.Len(t, model.leftItems(), 2)

	model.list.Select(1)
	model.syncSelectionFromList()
	model.focus = servicePlanBrowserFocusDetails
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 1, model.activeTab)
	model = loadSelectedTestDetails(t, model)
	require.Len(t, loader.calls, 2)
}

func TestServicePlanBrowserSelectionUpdatesPlanDetailsWithoutEnter(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithTwoPlans(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)

	require.Equal(t, "Standard", model.selectedPlan().Name)
	require.Contains(t, model.viewport.View(), "Postgres / Standard")
	require.Len(t, loader.calls, 1)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, "Premium", model.selectedPlan().Name)
	require.Contains(t, model.viewport.View(), "Postgres / Premium")
	require.Contains(t, model.viewport.View(), "Loading details")
	require.Len(t, loader.calls, 1)

	model = loadSelectedTestDetails(t, model)
	require.Len(t, loader.calls, 2)
}

func TestServicePlanBrowserTabSwitchesTopEnvironmentFromAnyPane(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 1, model.activeTab)
	require.Equal(t, "prod", model.activeEnvironmentName())
	require.Equal(t, "prod", model.selectedEnvironment().Name)
	model = loadSelectedTestDetails(t, model)

	model.focus = servicePlanBrowserFocusDetails
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 0, model.activeTab)
	require.Equal(t, "dev", model.activeEnvironmentName())
}

func TestServicePlanBrowserTabCyclesTopEnvironments(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	model.activeTab = 1
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 0, model.activeTab)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 1, model.activeTab)
}

func TestServicePlanBrowserEnvironmentTabsFilterServiceAccordion(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithTwoPlans(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	require.Equal(t, "dev", model.activeEnvironmentName())
	require.Len(t, model.leftItems(), 3)
	require.Equal(t, "Standard", model.leftItems()[1].title)
	require.Empty(t, model.leftItems()[1].description)
	require.Equal(t, "Premium", model.leftItems()[2].title)
	require.Empty(t, model.leftItems()[2].description)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, "prod", model.activeEnvironmentName())
	require.Len(t, model.leftItems(), 2)
	require.Equal(t, "Standard", model.leftItems()[1].title)
	require.Empty(t, model.leftItems()[1].description)
	require.Equal(t, "Standard", model.selectedPlan().Name)
	require.Equal(t, "prod", model.selectedEnvironment().Name)
}

func TestServicePlanBrowserServiceSummaryOmitsEnvironmentCountsFromPlanBullets(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithTwoPlans(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	content := model.renderServiceContent(model.leftItems()[0], 80)

	require.Contains(t, content, "Standard")
	require.Contains(t, content, "Premium")
	require.NotContains(t, content, "environment(s)")
	require.NotContains(t, content, "Standard:")
}

func TestServicePlanBrowserEscapeMovesFocusBackToPlans(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.focus = servicePlanBrowserFocusDetails

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, servicePlanBrowserFocusLeft, model.focus)
}

func TestServicePlanBrowserLeftArrowMovesFocusBackToPlans(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.focus = servicePlanBrowserFocusDetails

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, servicePlanBrowserFocusLeft, model.focus)
}

func TestServicePlanBrowserEnterFocusesDetailsPane(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)
	model.list.Select(1)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, servicePlanBrowserFocusDetails, model.focus)
}

func TestServicePlanBrowserEnvironmentTabsUseDebugTabStyle(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	tabs := model.renderEnvironmentTabs(80)

	require.Contains(t, tabs, "╭")
	require.Contains(t, tabs, "┴")
	require.Contains(t, tabs, "dev")
	require.Contains(t, tabs, "prod")
	model.activeTab = 1
	require.Contains(t, model.renderEnvironmentTabs(80), "┘")
	require.Equal(t, "82", string(servicePlanEnvironmentColor("Dev")))
	require.Equal(t, "160", string(servicePlanEnvironmentColor("Prod")))
}

func TestServicePlanBrowserDetailCursorAutoScrollsViewport(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)
	model.focus = servicePlanBrowserFocusDetails
	model.viewport.Height = 4

	model.moveDetailCursor(6)

	require.Greater(t, model.viewport.YOffset, 0)
	require.Contains(t, model.viewport.View(), "Users")
}

func TestServicePlanBrowserDetailCursorCanReturnToTopOfPlanDetails(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)
	model.focus = servicePlanBrowserFocusDetails
	model.viewport.Height = 4

	model.moveDetailCursor(6)
	require.Greater(t, model.viewport.YOffset, 0)

	for i := 0; i < 7; i++ {
		model.moveDetailCursor(-1)
	}

	require.Equal(t, -1, model.detailCursor)
	require.Equal(t, 0, model.viewport.YOffset)
	require.Contains(t, model.viewport.View(), "Postgres / Standard")

	model.moveDetailCursor(1)

	require.Equal(t, 0, model.detailCursor)
	require.Greater(t, model.viewport.YOffset, 0)
	require.Contains(t, model.viewport.View(), "Deployment model")
}

func TestServicePlanBrowserEnvironmentSwitchReturnsViewportToTop(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)
	model.focus = servicePlanBrowserFocusDetails
	model.viewport.Height = 4
	model.moveDetailCursor(6)
	require.Greater(t, model.viewport.YOffset, 0)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, -1, model.detailCursor)
	require.Equal(t, 0, model.viewport.YOffset)
	require.Equal(t, "prod", model.activeEnvironmentName())
	require.Contains(t, model.viewport.View(), "Postgres / Standard")
}

func TestServicePlanBrowserPlanItemsIncludeHostingBadges(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	items := model.leftItems()

	require.Equal(t, "Omnistrate Hosted", items[1].hostingBadge.Label)
	require.Equal(t, "BYOC", items[2].hostingBadge.Label)
	require.Equal(t, "Hosted", items[3].hostingBadge.Label)
	require.Equal(t, "Hosted", items[4].hostingBadge.Label)
}

func TestServicePlanHostingBadgeMapping(t *testing.T) {
	tests := []struct {
		name      string
		modelType string
		tierType  string
		want      string
	}{
		{name: "omnistrate hosted model type", modelType: "OMNISTRATE_HOSTED", tierType: "SHARED", want: "Omnistrate Hosted"},
		{name: "omnistrate hosted tier type", modelType: "OMNISTRATE_MULTI_TENANCY", tierType: "OMNISTRATE_HOSTED", want: "Omnistrate Hosted"},
		{name: "customer hosted model type", modelType: "CUSTOMER_HOSTED", tierType: "CUSTOM_TENANCY", want: "Hosted"},
		{name: "byoa alias", modelType: "BYOA", tierType: "DEDICATED", want: "BYOC"},
		{name: "hosted fallback", modelType: "", tierType: "DEDICATED", want: "Hosted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, servicePlanHostingBadgeForValues(tt.modelType, tt.tierType).Label)
		})
	}
}

func TestServicePlanBrowserModalFiltering(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)
	model.focus = servicePlanBrowserFocusDetails
	model.detailCursor = 4

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.NotNil(t, model.modal)
	require.Equal(t, servicePlanBrowserModalDeployments, model.modal.Kind)

	filtered := filterServicePlanModalRows(model.modal.Rows, "aws")
	require.Len(t, filtered, 1)
	require.Contains(t, filtered[0].Text, "aws")
}

func TestDedupeServicePlanUsersUsesUserIDThenEmail(t *testing.T) {
	users := []openapiclientfleet.User{
		{UserId: "user-1", Email: "first@example.com"},
		{UserId: "user-1", Email: "second@example.com"},
		{Email: "shared@example.com"},
		{Email: "SHARED@example.com"},
		{Email: "unique@example.com"},
	}

	unique := dedupeServicePlanUsers(users)

	require.Len(t, unique, 3)
	require.Equal(t, "user-1", unique[0].UserId)
	require.Equal(t, "shared@example.com", unique[1].Email)
	require.Equal(t, "unique@example.com", unique[2].Email)
}

func TestProductTierSummaryHelpers(t *testing.T) {
	featureName := "CUSTOM_TERRAFORM_POLICY"
	scope := "service"
	features := map[string]bool{"AUDIT": true, "DISABLED": false}
	readiness := map[string]map[string]string{"aws": {"us-east-1": "ready"}}
	productTier := &openapiclient.DescribeProductTierResult{
		AwsRegions:                    []string{"us-west-2"},
		GcpRegions:                    []string{"us-central1"},
		CloudProvidersConfigReadiness: &readiness,
		EnabledFeatures: []openapiclient.ProductTierFeatureDetail{
			{Feature: &featureName, Scope: &scope},
		},
		Features: &features,
	}

	clouds, regions := productTierCloudsAndRegions(productTier)
	enabled := productTierEnabledFeatures(productTier)

	require.ElementsMatch(t, []string{"aws", "gcp"}, clouds)
	require.ElementsMatch(t, []string{"us-east-1", "us-west-2", "us-central1"}, regions)
	require.ElementsMatch(t, []string{"AUDIT", "CUSTOM_TERRAFORM_POLICY (service)"}, enabled)
}

func TestFormatServicePlansFromServicesSupportsJSONParityPath(t *testing.T) {
	filterMaps, err := utils.ParseFilters([]string{"environment:dev"}, utils.GetSupportedFilterKeys(displaymodel.ServicePlanVersion{}))
	require.NoError(t, err)

	plans, err := formatServicePlansFromServices(testBrowserServices(), filterMaps, false)

	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Equal(t, "plan-dev", plans[0].PlanID)
	require.Equal(t, "Standard", plans[0].PlanName)
}

func testBrowserServices() []openapiclient.DescribeServiceResult {
	return []openapiclient.DescribeServiceResult{
		{
			Id:   "svc-1",
			Name: "Postgres",
			ServiceEnvironments: []openapiclient.ServiceEnvironment{
				{
					Id:   "env-dev",
					Name: "dev",
					ServicePlans: []openapiclient.ServicePlan{
						{
							Name:          "Standard",
							ProductTierID: "plan-dev",
							TierType:      "OMNISTRATE_HOSTED",
							ModelType:     "OMNISTRATE_MULTI_TENANCY",
						},
					},
				},
				{
					Id:   "env-prod",
					Name: "prod",
					ServicePlans: []openapiclient.ServicePlan{
						{
							Name:          "Standard",
							ProductTierID: "plan-prod",
							TierType:      "OMNISTRATE_HOSTED",
							ModelType:     "OMNISTRATE_MULTI_TENANCY",
						},
					},
				},
			},
		},
	}
}

func testBrowserServicesWithTwoPlans() []openapiclient.DescribeServiceResult {
	services := testBrowserServices()
	services[0].ServiceEnvironments[0].ServicePlans = append(services[0].ServiceEnvironments[0].ServicePlans, openapiclient.ServicePlan{
		Name:          "Premium",
		ProductTierID: "plan-premium",
		TierType:      "OMNISTRATE_HOSTED",
		ModelType:     "OMNISTRATE_DEDICATED_TENANCY",
	})
	return services
}

func testBrowserServicesWithHostingModels() []openapiclient.DescribeServiceResult {
	return []openapiclient.DescribeServiceResult{
		{
			Id:   "svc-1",
			Name: "Postgres",
			ServiceEnvironments: []openapiclient.ServiceEnvironment{
				{
					Id:   "env-dev",
					Name: "dev",
					ServicePlans: []openapiclient.ServicePlan{
						{Name: "Omni", ProductTierID: "plan-omni", ModelType: "OMNISTRATE_HOSTED", TierType: "SHARED"},
						{Name: "Byoc", ProductTierID: "plan-byoc", ModelType: "BYOA", TierType: "DEDICATED"},
						{Name: "Hosted", ProductTierID: "plan-hosted", ModelType: "", TierType: "DEDICATED"},
						{Name: "Customer Hosted", ProductTierID: "plan-customer-hosted", ModelType: "CUSTOMER_HOSTED", TierType: "CUSTOM_TENANCY"},
					},
				},
			},
		},
	}
}

func testBrowserDetails(catalog servicePlanBrowserCatalog) map[string]servicePlanEnvironmentDetails {
	details := map[string]servicePlanEnvironmentDetails{}
	for _, service := range catalog.Services {
		for _, plan := range service.Plans {
			for _, env := range plan.Environments {
				details[env.cacheKey()] = servicePlanEnvironmentDetails{
					DeploymentModel:          "OMNISTRATE_HOSTED / OMNISTRATE_MULTI_TENANCY",
					EnabledFeatures:          []string{"AUDIT"},
					Clouds:                   []string{"aws"},
					Regions:                  []string{"us-west-2"},
					DeploymentsCount:         2,
					ActiveSubscriptionsCount: 1,
					UniqueUsersCount:         1,
					Deployments: []servicePlanDeploymentRow{
						{ID: "inst-1", Status: "RUNNING", Cloud: "aws", Region: "us-west-2", Owner: "Alice"},
						{ID: "inst-2", Status: "RUNNING", Cloud: "gcp", Region: "us-central1", Owner: "Bob"},
					},
					Subscriptions: []servicePlanSubscriptionRow{
						{ID: "sub-1", Status: "ACTIVE", RootUserEmail: "alice@example.com", RootUserName: "Alice", InstanceCount: 2},
					},
					Users: []servicePlanUserRow{
						{ID: "user-1", Email: "alice@example.com", Name: "Alice", Status: "ACTIVE", OrgName: "Acme"},
					},
				}
			}
		}
	}
	return details
}
