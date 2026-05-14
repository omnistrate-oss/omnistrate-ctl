package serviceplan

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	displaymodel "github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/require"
)

type fakeServicePlanBrowserLoader struct {
	details        map[string]servicePlanEnvironmentDetails
	calls          []string
	form           servicePlanDeploymentForm
	formErr        error
	formCalls      []string
	launchID       string
	launchErr      error
	launchReq      *servicePlanDeploymentLaunchRequest
	launchCalls    int
	connectAccount servicePlanCustomerCloudAccountRow
	connectErr     error
	connectReq     *servicePlanCustomerCloudAccountConnectRequest
	connectCalls   int
	refreshAccount servicePlanCustomerCloudAccountRow
	refreshErr     error
	refreshReq     *servicePlanCustomerCloudAccountActionRequest
	refreshCalls   int
	deleteErr      error
	deleteReq      *servicePlanCustomerCloudAccountActionRequest
	deleteCalls    int
	retryAccount   servicePlanCustomerCloudAccountRow
	retryErr       error
	retryReq       *servicePlanCustomerCloudAccountActionRequest
	retryCalls     int
}

func (f *fakeServicePlanBrowserLoader) LoadEnvironmentDetails(_ context.Context, _ string, env servicePlanBrowserEnvironment) (servicePlanEnvironmentDetails, error) {
	key := env.cacheKey()
	f.calls = append(f.calls, key)
	return f.details[key], nil
}

func (f *fakeServicePlanBrowserLoader) LoadDeploymentForm(_ context.Context, _ string, env servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error) {
	f.formCalls = append(f.formCalls, env.cacheKey())
	return f.form, f.formErr
}

func (f *fakeServicePlanBrowserLoader) LaunchDeployment(_ context.Context, _ string, request servicePlanDeploymentLaunchRequest) (string, error) {
	f.launchCalls++
	f.launchReq = &request
	if f.launchID == "" {
		f.launchID = "inst-created"
	}
	return f.launchID, f.launchErr
}

func (f *fakeServicePlanBrowserLoader) CreateCustomerCloudAccount(_ context.Context, _ string, request servicePlanCustomerCloudAccountConnectRequest) (servicePlanCustomerCloudAccountRow, error) {
	f.connectCalls++
	f.connectReq = &request
	if f.connectAccount.InstanceID == "" {
		f.connectAccount = servicePlanCustomerCloudAccountRow{InstanceID: "acct-created", CloudProvider: request.CloudProvider, Status: "READY"}
	}
	return f.connectAccount, f.connectErr
}

func (f *fakeServicePlanBrowserLoader) RefreshCustomerCloudAccount(_ context.Context, _ string, request servicePlanCustomerCloudAccountActionRequest) (servicePlanCustomerCloudAccountRow, error) {
	f.refreshCalls++
	f.refreshReq = &request
	if f.refreshAccount.InstanceID == "" {
		f.refreshAccount = request.Account
		f.refreshAccount.Status = "READY"
	}
	return f.refreshAccount, f.refreshErr
}

func (f *fakeServicePlanBrowserLoader) DeleteCustomerCloudAccount(_ context.Context, _ string, request servicePlanCustomerCloudAccountActionRequest) error {
	f.deleteCalls++
	f.deleteReq = &request
	return f.deleteErr
}

func (f *fakeServicePlanBrowserLoader) RetryCustomerCloudAccount(_ context.Context, _ string, request servicePlanCustomerCloudAccountActionRequest) (servicePlanCustomerCloudAccountRow, error) {
	f.retryCalls++
	f.retryReq = &request
	if f.retryAccount.InstanceID == "" {
		f.retryAccount = request.Account
		f.retryAccount.Status = "READY"
	}
	return f.retryAccount, f.retryErr
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

	model.list.Select(0)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Len(t, model.leftItems(), 1)

	model.list.Select(0)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
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
	require.Equal(t, "Service Plans", model.list.Title)
}

func TestServicePlanBrowserLeftPaneUsesTreeAccordion(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithTwoPlans(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	items := model.leftItems()

	require.True(t, items[0].isService)
	require.True(t, items[0].expanded)
	require.Empty(t, items[0].description)
	require.Equal(t, "- ", servicePlanBrowserLeftItemTreePrefix(items[0]))
	require.Equal(t, "  ├─ ", servicePlanBrowserLeftItemTreePrefix(items[1]))
	require.Equal(t, "  └─ ", servicePlanBrowserLeftItemTreePrefix(items[2]))
	require.NotContains(t, model.View(), "plan name(s)")
	require.NotContains(t, model.View(), "Service Plans ·")
}

func TestServicePlanBrowserHeaderOmitsCatalogChecks(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)

	view := model.View()

	require.Contains(t, view, "Service Plan Browser")
	require.NotContains(t, view, "Service catalog loaded")
	require.NotContains(t, view, "Plan details available")
}

func TestServicePlanBrowserTabbedBodyDoesNotWrapPaneBorders(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithTwoPlans(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)
	model.setSize(120, 32)

	view := model.renderEnvironmentTabsWithBody(model.width, model.renderBrowserBody())

	require.NotContains(t, view, "╰")
	for _, line := range strings.Split(view, "\n") {
		require.LessOrEqual(t, lipgloss.Width(line), model.width)
	}
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
	activeStyle := servicePlanEnvironmentTabStyle(true, servicePlanTabBorderWithBottom("┘", " ", "└"))
	inactiveStyle := servicePlanEnvironmentTabStyle(false, servicePlanTabBorderWithBottom("┴", "─", "┴"))
	require.Equal(t, lipgloss.Color("230"), activeStyle.GetForeground())
	require.True(t, activeStyle.GetBold())
	require.Equal(t, lipgloss.Color("245"), inactiveStyle.GetForeground())
	require.False(t, inactiveStyle.GetFaint())
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

func TestServicePlanBrowserBYOCDetailsShowCloudAccounts(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)
	model.list.Select(2)
	model.syncSelectionFromList()
	env := model.selectedEnvironment()
	require.NotNil(t, env)
	model.detailCache[env.cacheKey()] = servicePlanEnvironmentDetails{
		DeploymentModel:            "DEDICATED / BYOA",
		CustomerCloudAccountsCount: 1,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-1", CloudProvider: "aws", Status: "RUNNING", Region: "us-west-2", CustomerEmail: "alice@example.com", SubscriptionID: "sub-1", AWSAccountID: "123456789012"},
		},
	}

	rows := model.detailRows()
	require.Equal(t, "Cloud accounts", rows[len(rows)-1].Label)
	require.Equal(t, "1 connected", rows[len(rows)-1].Value)

	model.focus = servicePlanBrowserFocusDetails
	model.detailCursor = len(rows) - 1
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.NotNil(t, model.modal)
	require.Equal(t, servicePlanBrowserModalCloudAccounts, model.modal.Kind)
	require.Contains(t, model.modal.Rows[0].Text, "AWS account 123456789012")
	require.NotContains(t, model.modal.Rows[0].Text, "acct-1")
	require.Contains(t, model.modal.Rows[0].Text, "alice@example.com")
}

func TestServicePlanDeploymentResourcesSkipInjectedCustomerAccountResource(t *testing.T) {
	resources := servicePlanDeploymentResources([]openapiclientfleet.ResourceEntity{
		{ResourceId: "r-injectedaccountconfigpt123", Name: "Cloud Provider Account", UrlKey: "omnistrateCloudAccountConfig"},
		{ResourceId: "r-api", Name: "API", UrlKey: "api"},
		{ResourceId: "r-worker", UrlKey: "worker"},
		{ResourceId: "r-missing-key", Name: "Missing URL key"},
		{ResourceId: "r-old", Name: "Old", UrlKey: "old", IsDeprecated: true},
	})

	require.Len(t, resources, 2)
	require.Equal(t, "r-api", resources[0].ID)
	require.Equal(t, "API", resources[0].Name)
	require.Equal(t, "api", resources[0].URLKey)
	require.Equal(t, "r-worker", resources[1].Name)
}

func TestServicePlanInputParametersNotFoundIsEmptyParameterMetadata(t *testing.T) {
	require.True(t, servicePlanInputParametersNotFound(errors.New("not_found\nDetail: Invalid request: failed to query input parameter: record not found")))
	require.True(t, servicePlanInputParametersNotFound(errors.New("NOT_FOUND: input parameter record not found")))
	require.False(t, servicePlanInputParametersNotFound(errors.New("not_found\nDetail: service offering record not found")))
	require.False(t, servicePlanInputParametersNotFound(errors.New("internal error: failed to query input parameter")))
	require.False(t, servicePlanInputParametersNotFound(nil))
}

func TestServicePlanBrowserDeploymentHotkeyLoadsFormAndLaunchesWithDefaults(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, nil)
	model.list.Select(2)
	model.syncSelectionFromList()
	env := *model.selectedEnvironment()
	parameter := servicePlanDeploymentParameter{
		Key:          "instance_type",
		DisplayName:  "Instance type",
		Type:         "string",
		Required:     true,
		DefaultValue: "small",
	}
	loader := &fakeServicePlanBrowserLoader{
		form: servicePlanDeploymentForm{
			Environment: env,
			Version:     "1.0",
			Resources: []servicePlanDeploymentResource{
				{ID: "res-1", Name: "API", URLKey: "api"},
			},
			CloudProviders: []string{"aws"},
			RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
			Parameters:     []servicePlanDeploymentParameter{parameter},
			ParametersByResource: map[string][]servicePlanDeploymentParameter{
				"res-1": []servicePlanDeploymentParameter{parameter},
			},
			CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
				{InstanceID: "acct-1", CloudProvider: "aws", Status: "RUNNING"},
			},
			RequiresCustomerAccount: true,
		},
		launchID: "inst-1",
	}
	model.loadDeploymentForm = func(env servicePlanBrowserEnvironment) (servicePlanDeploymentForm, error) {
		return loader.LoadDeploymentForm(context.Background(), "token", env)
	}
	model.launchDeployment = func(request servicePlanDeploymentLaunchRequest) (string, error) {
		return loader.LaunchDeployment(context.Background(), "token", request)
	}

	updated, cmd := model.Update(tea.KeyMsg{Runes: []rune{'d'}, Type: tea.KeyRunes})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.loadingDeploymentForm)
	require.Len(t, loader.formCalls, 0)

	form, err := loader.LoadDeploymentForm(context.Background(), "token", env)
	require.NoError(t, err)
	updated, cmd = model.Update(servicePlanDeploymentFormLoadedMsg{form: form})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.NotNil(t, model.deploymentForm)
	require.Equal(t, servicePlanDeploymentStepCloud, model.deploymentForm.currentStep())
	require.NotContains(t, model.deploymentForm.Steps, servicePlanDeploymentStepResource)
	require.Contains(t, model.renderDeploymentForm(), "Step 1/5: Cloud Provider")
	require.NotContains(t, model.renderDeploymentForm(), "Back")
	require.NotContains(t, model.renderDeploymentForm(), "Next")

	for i := 0; i < 3; i++ {
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		model = updated.(servicePlanBrowserModel)
	}
	require.Contains(t, model.renderDeploymentForm(), "small")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Contains(t, model.renderDeploymentForm(), "Review")

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.Launching)
	updated, cmd = model.Update(cmd())
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.False(t, model.deploymentForm.Launching)
	require.Equal(t, 1, loader.launchCalls)
	require.Equal(t, "API", loader.launchReq.ResourceName)
	require.Equal(t, "aws", loader.launchReq.CloudProvider)
	require.Equal(t, "us-west-2", loader.launchReq.Region)
	require.Equal(t, "acct-1", loader.launchReq.CustomerAccountID)
	require.Equal(t, "small", loader.launchReq.Params["instance_type"])
	require.Contains(t, model.renderDeploymentForm(), "Deployment launched: inst-1")
	require.Contains(t, model.renderDeploymentForm(), "omnistrate-ctl instance describe inst-1")
	require.Contains(t, model.renderDeploymentForm(), "omnistrate-ctl instance list-endpoints inst-1")
	require.Contains(t, model.renderDeploymentForm(), "omnistrate-ctl instance debug inst-1")
}

func TestServicePlanDeploymentWizardSkipsResourceStepForSingleResource(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Resources:      []servicePlanDeploymentResource{{ID: "res-api", Name: "API", URLKey: "api"}},
		CloudProviders: []string{"aws"},
		RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
	}, 80)

	require.Equal(t, "API", state.ResourceName)
	require.NotContains(t, state.Steps, servicePlanDeploymentStepResource)
	require.Equal(t, servicePlanDeploymentStepCloud, state.currentStep())
	require.Equal(t, 3, len(state.Steps))
}

func TestServicePlanDeploymentFormRefreshesParametersWhenResourceChanges(t *testing.T) {
	form := servicePlanDeploymentForm{
		Resources: []servicePlanDeploymentResource{
			{ID: "res-api", Name: "API", URLKey: "api"},
			{ID: "res-worker", Name: "Worker", URLKey: "worker"},
		},
		CloudProviders: []string{"aws"},
		RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
		Parameters: []servicePlanDeploymentParameter{
			{Key: "api_size", DisplayName: "API size", Type: "string", DefaultValue: "small"},
		},
		ParametersByResource: map[string][]servicePlanDeploymentParameter{
			"res-api": []servicePlanDeploymentParameter{
				{Key: "api_size", DisplayName: "API size", Type: "string", DefaultValue: "small"},
			},
			"res-worker": []servicePlanDeploymentParameter{
				{Key: "worker_size", DisplayName: "Worker size", Type: "string", DefaultValue: "medium"},
			},
		},
	}

	state := newServicePlanDeploymentFormState(form, 80)
	require.Equal(t, servicePlanDeploymentStepResource, state.currentStep())

	state.SelectionCursor = 1
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))

	require.Equal(t, servicePlanDeploymentStepSystemParams, state.currentStep())
	require.Equal(t, "res-worker", state.ParameterResourceID)
	require.Contains(t, servicePlanDeploymentFormFieldLabels(state.ParamFields), "Worker size")
	require.NotContains(t, servicePlanDeploymentFormFieldLabels(state.ParamFields), "API size")
}

func TestServicePlanDeploymentWizardKeepsParameterStepForNonDefaultResource(t *testing.T) {
	form := servicePlanDeploymentForm{
		Resources: []servicePlanDeploymentResource{
			{ID: "res-api", Name: "API", URLKey: "api"},
			{ID: "res-worker", Name: "Worker", URLKey: "worker"},
		},
		CloudProviders: []string{"aws"},
		RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
		ParametersByResource: map[string][]servicePlanDeploymentParameter{
			"res-api": []servicePlanDeploymentParameter{},
			"res-worker": []servicePlanDeploymentParameter{
				{Key: "worker_size", DisplayName: "Worker size", Type: "string", DefaultValue: "medium"},
			},
		},
	}

	state := newServicePlanDeploymentFormState(form, 80)

	require.Contains(t, state.Steps, servicePlanDeploymentStepSystemParams)
	state.SelectionCursor = 1
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, servicePlanDeploymentStepSystemParams, state.currentStep())
	require.Contains(t, servicePlanDeploymentFormFieldLabels(state.ParamFields), "Worker size")
}

func TestServicePlanDeploymentParameterOptionsUseSelectableValues(t *testing.T) {
	form := servicePlanDeploymentForm{
		Resources: []servicePlanDeploymentResource{
			{ID: "res-api", Name: "API", URLKey: "api"},
		},
		CloudProviders: []string{"aws"},
		RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
		ParametersByResource: map[string][]servicePlanDeploymentParameter{
			"res-api": []servicePlanDeploymentParameter{
				{Key: "size", DisplayName: "Size", Type: "string", Required: true, Options: []string{"small", "large"}},
			},
		},
	}

	state := newServicePlanDeploymentFormState(form, 80)
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, servicePlanDeploymentStepSystemParams, state.currentStep())
	require.True(t, state.moveCurrentParameterOption(1))
	require.Equal(t, "small", state.ParamFields[0].Input.Value())
	require.True(t, state.moveCurrentParameterOption(1))
	require.Equal(t, "large", state.ParamFields[0].Input.Value())
}

func TestServicePlanDeploymentWizardPromptsForCustomerInProd(t *testing.T) {
	form := servicePlanDeploymentForm{
		Environment: servicePlanBrowserEnvironment{Name: "PROD"},
		Resources: []servicePlanDeploymentResource{
			{ID: "res-1", Name: "API", URLKey: "api"},
		},
		CloudProviders: []string{"aws"},
		RegionsByCloud: map[string][]string{"aws": []string{"us-west-2"}},
		Customers: []servicePlanDeploymentCustomer{
			{UserID: "user-1", Email: "alice@example.com", Name: "Alice", OrgName: "Acme"},
		},
	}

	state := newServicePlanDeploymentFormState(form, 80)

	require.Equal(t, servicePlanDeploymentStepCustomer, state.currentStep())
	options := state.currentOptions()
	require.Len(t, options, 2)
	require.Equal(t, "Self", options[0].Label)
	state.SelectionCursor = 1
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, "alice@example.com", state.SelectedCustomer.Email)
}

func TestServicePlanDeploymentCustomerStepIsSearchable(t *testing.T) {
	form := servicePlanDeploymentForm{
		Environment: servicePlanBrowserEnvironment{Name: "PROD"},
		Resources:   []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		Customers: []servicePlanDeploymentCustomer{
			{UserID: "user-1", Email: "alice@example.com", Name: "Alice", OrgName: "Acme"},
			{UserID: "user-2", Email: "bob@example.com", Name: "Bob", OrgName: "Beta"},
		},
	}

	state := newServicePlanDeploymentFormState(form, 80)
	for _, r := range "bob" {
		_, handled := state.updateActiveTextInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		require.True(t, handled)
	}

	options := state.currentOptions()
	require.Len(t, options, 1)
	require.Equal(t, "bob@example.com", options[0].Customer.Email)
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, "bob@example.com", state.SelectedCustomer.Email)
}

func TestServicePlanDeploymentFreeformCloudAndRegionWhenOptionsMissing(t *testing.T) {
	form := servicePlanDeploymentForm{
		Resources: []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
	}

	state := newServicePlanDeploymentFormState(form, 80)
	require.Equal(t, servicePlanDeploymentStepCloud, state.currentStep())
	require.True(t, state.currentStepUsesFreeformInput())
	require.Contains(t, strings.Join(state.renderDeploymentOptionLines(20, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()), "\n"), "No fixed options")

	state.OptionInput.SetValue("custom-cloud")
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, "custom-cloud", state.CloudProvider)
	require.Equal(t, servicePlanDeploymentStepRegion, state.currentStep())
	require.True(t, state.currentStepUsesFreeformInput())

	state.OptionInput.SetValue("custom-region")
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, "custom-region", state.Region)

	request, err := state.launchRequest()
	require.NoError(t, err)
	require.Equal(t, "custom-cloud", request.CloudProvider)
	require.Equal(t, "custom-region", request.Region)
}

func TestServicePlanDeploymentCloudAccountsFilterBySelectedCustomer(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		Customers:               []servicePlanDeploymentCustomer{{UserID: "user-1", Email: "alice@example.com", Name: "Alice"}},
		Subscriptions:           []servicePlanSubscriptionRow{{ID: "sub-1", RootUserID: "user-1", RootUserEmail: "alice@example.com"}},
		CustomerCloudAccounts:   []servicePlanCustomerCloudAccountRow{{InstanceID: "acct-1", CloudProvider: "aws", SubscriptionID: "sub-1", CustomerEmail: "alice@example.com"}, {InstanceID: "acct-2", CloudProvider: "aws", SubscriptionID: "sub-2", CustomerEmail: "bob@example.com"}},
	}, 80)

	state.SelectedCustomer = servicePlanDeploymentCustomer{UserID: "user-1", Email: "alice@example.com", Name: "Alice"}

	accounts := state.filteredCloudAccounts()
	require.Len(t, accounts, 1)
	require.Equal(t, "acct-1", accounts[0].InstanceID)
}

func TestServicePlanDeploymentCloudAccountStepOffersInlineConnectWhenNoAccountMatches(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
	}, 80)

	require.Equal(t, servicePlanDeploymentStepCloud, state.currentStep())
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, servicePlanDeploymentStepCloudAccount, state.currentStep())

	options := state.currentOptions()
	require.Len(t, options, 1)
	require.True(t, options[0].ConnectAccount)
	require.Equal(t, "Connect your aws account", options[0].Label)
	require.Empty(t, state.customerCloudAccountActionButtons(options[0]))
	require.Equal(t, []string{"▸ Connect your aws account"}, state.renderDeploymentOptionLines(20, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()))
	require.Equal(t, "enter: select  esc: cancel", state.deploymentHelpLine())

	require.NoError(t, state.advanceStep(80))
	require.Equal(t, servicePlanDeploymentStepConnectAccount, state.currentStep())
	require.Contains(t, servicePlanDeploymentFormFieldLabels(state.ParamFields), "AWS account ID")

	state.ParamFields[0].Input.SetValue("123456789012")
	request, err := state.customerAccountConnectRequest()
	require.NoError(t, err)
	require.Equal(t, "aws", request.CloudProvider)
	require.Equal(t, "123456789012", request.Values[servicePlanCustomerAccountAWSAccountIDKey])

	params, err := servicePlanCustomerAccountRequestParams(context.Background(), "token", "aws", request.Values)
	require.NoError(t, err)
	require.Equal(t, "CloudFormation", params[servicePlanCustomerAccountIacToolKey])
	require.Equal(t, "123456789012", params[servicePlanCustomerAccountAWSAccountIDKey])
	require.Equal(t, "arn:aws:iam::123456789012:role/omnistrate-bootstrap-role", params[servicePlanCustomerAccountAWSBootstrapRoleKey])
}

func TestServicePlanDeploymentCloudAccountStepUsesCloudIdentityAndFailedActions(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "instance-qu9xnfp2x", CloudProvider: "aws", Status: "FAILED", AWSAccountID: "123456789012", CustomerEmail: "alok@nistro.ai"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))
	require.Equal(t, servicePlanDeploymentStepCloudAccount, state.currentStep())

	options := state.currentOptions()
	require.Len(t, options, 2)
	require.Equal(t, "AWS account 123456789012", options[0].Label)
	require.Equal(t, servicePlanCustomerCloudAccountActionRetry, options[0].AccountAction)
	require.True(t, options[1].ConnectAccount)
	require.Equal(t, "Connect new aws account", options[1].Label)
	buttons := state.customerCloudAccountActionButtons(options[0])
	require.Len(t, buttons, 2)
	require.Equal(t, "Retry", buttons[0].Label)
	require.Equal(t, "Delete", buttons[1].Label)
	rendered := strings.Join(state.renderDeploymentOptionLines(20, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()), "\n")
	require.Contains(t, rendered, "1. AWS account 123456789012")
	require.Contains(t, rendered, "Retry")
	require.Contains(t, rendered, "Delete")
	require.Contains(t, rendered, "\n\n  Connect new aws account")
	require.NotContains(t, rendered, "instance-qu9xnfp2x")

	_, action, ok := state.selectedCustomerCloudAccountAction()
	require.True(t, ok)
	require.Equal(t, servicePlanCustomerCloudAccountActionRetry, action)
	state.moveCustomerCloudAccountActionCursor(1)
	_, action, ok = state.selectedCustomerCloudAccountAction()
	require.True(t, ok)
	require.Equal(t, servicePlanCustomerCloudAccountActionDelete, action)
}

func TestServicePlanDeploymentCloudAccountActionTabSwitchesAccountButtons(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-failed", CloudProvider: "aws", Status: "FAILED", AWSAccountID: "123456789012"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))
	model := servicePlanBrowserModel{deploymentForm: &state, detailPanelWidth: 80}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	_, action, ok := model.deploymentForm.selectedCustomerCloudAccountAction()
	require.True(t, ok)
	require.Equal(t, servicePlanCustomerCloudAccountActionDelete, action)

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	_, action, ok = model.deploymentForm.selectedCustomerCloudAccountAction()
	require.True(t, ok)
	require.Equal(t, servicePlanCustomerCloudAccountActionRetry, action)
}

func TestServicePlanDeploymentPendingCloudAccountShowsConnectionInstructions(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{
				InstanceID:               "acct-pending",
				AccountConfigID:          "ac-123",
				CloudProvider:            "aws",
				Status:                   "DEPLOYING",
				StatusMessage:            "reconciling instance",
				AWSAccountID:             "767925118737",
				AWSCloudFormationURL:     "https://example.com/template.yml",
				AWSCloudFormationNoLBURL: "https://example.com/template-no-lb.yml",
				CustomerEmail:            "alok@nistro.ai",
				SubscriptionID:           "sub-KImOLdo2ke",
			},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))

	rendered := strings.Join(state.renderDeploymentOptionLines(30, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()), "\n")
	require.Contains(t, rendered, "1. AWS account 767925118737")
	require.Contains(t, rendered, "Waiting")
	require.Contains(t, rendered, "Account config ID: ac-123")
	require.Contains(t, rendered, "CloudFormation template URL: https://example.com/template.yml")
	require.NotContains(t, rendered, "CloudFormation no-LB template URL")
	require.Contains(t, state.deploymentHelpLine(), "c: copy template URL")
}

func TestServicePlanDeploymentCloudAccountStatusIsStyled(t *testing.T) {
	rendered := renderServicePlanCustomerCloudAccountDescription(servicePlanCustomerCloudAccountRow{
		Status:         "READY",
		StatusMessage:  "Instance deployed",
		CustomerEmail:  "alok@nistro.ai",
		SubscriptionID: "sub-KImOLdo2ke",
	}, lipgloss.NewStyle().Foreground(lipgloss.Color("245")), "    ")

	require.Contains(t, rendered, "READY")
	require.Contains(t, rendered, "account verified")
	require.NotContains(t, rendered, "Instance deployed")
	require.Contains(t, rendered, "alok@nistro.ai")
	require.Equal(t, lipgloss.Color("82"), servicePlanCustomerCloudAccountStatusStyle("READY", lipgloss.NewStyle()).GetForeground())
}

func TestServicePlanDeploymentCloudAccountBackfillsEmailFromSubscription(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		Subscriptions:           []servicePlanSubscriptionRow{{ID: "sub-KImOLdo2ke", RootUserEmail: "alok@nistro.ai"}},
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-1", CloudProvider: "aws", Status: "READY", StatusMessage: "account verified", AWSAccountID: "767925118737", SubscriptionID: "sub-KImOLdo2ke"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))

	accounts := state.filteredCloudAccounts()
	require.Len(t, accounts, 1)
	require.Equal(t, "alok@nistro.ai", accounts[0].CustomerEmail)

	rendered := strings.Join(state.renderDeploymentOptionLines(20, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle()), "\n")
	require.Contains(t, rendered, "READY")
	require.Contains(t, rendered, "account verified")
	require.Contains(t, rendered, "alok@nistro.ai")
}

func TestServicePlanDeploymentExistingPendingCloudAccountPollsOnCloudAccountStep(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{
		details: testBrowserDetails(catalog),
		refreshAccount: servicePlanCustomerCloudAccountRow{
			InstanceID:           "acct-pending",
			AccountConfigID:      "ac-123",
			CloudProvider:        "aws",
			Status:               "DEPLOYING",
			StatusMessage:        "still reconciling",
			AWSAccountID:         "767925118737",
			AWSCloudFormationURL: "https://example.com/template.yml",
		},
	}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Environment:             catalog.Services[0].Plans[0].Environments[0],
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{
				InstanceID:           "acct-pending",
				AccountConfigID:      "ac-123",
				CloudProvider:        "aws",
				Status:               "DEPLOYING",
				StatusMessage:        "reconciling instance",
				AWSAccountID:         "767925118737",
				AWSCloudFormationURL: "https://example.com/template.yml",
			},
		},
	}, 80)
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, servicePlanDeploymentStepCloudAccount, model.deploymentForm.currentStep())
	require.True(t, model.deploymentForm.CustomerAccountPollScheduled)

	updated, cmd = model.Update(servicePlanCustomerCloudAccountPollTickMsg{})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.False(t, model.deploymentForm.CustomerAccountPollScheduled)

	updated, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, 1, loader.refreshCalls)
	require.True(t, model.deploymentForm.CustomerAccountPollScheduled)
	require.Equal(t, "still reconciling", model.deploymentForm.Form.CustomerCloudAccounts[0].StatusMessage)
	require.Equal(t, servicePlanDeploymentStepCloudAccount, model.deploymentForm.currentStep())
}

func TestServicePlanDeploymentCopiesCloudFormationTemplateURL(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{
				InstanceID:               "acct-pending",
				CloudProvider:            "aws",
				Status:                   "DEPLOYING",
				AWSAccountID:             "767925118737",
				AWSCloudFormationURL:     "https://example.com/template.yml",
				AWSCloudFormationNoLBURL: "https://example.com/template-no-lb.yml",
			},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))

	originalCopy := servicePlanBrowserCopyToClipboard
	var copied string
	servicePlanBrowserCopyToClipboard = func(text string) error {
		copied = text
		return nil
	}
	t.Cleanup(func() {
		servicePlanBrowserCopyToClipboard = originalCopy
	})

	model := servicePlanBrowserModel{deploymentForm: &state, detailPanelWidth: 80}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	msg := cmd()
	require.Equal(t, "https://example.com/template.yml", copied)

	updated, cmd = model.Update(msg)
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, "Copied CloudFormation template URL", model.deploymentForm.Notice)
}

func TestServicePlanDeploymentConnectProgressShowsOnboardingInstructions(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{
		details: testBrowserDetails(catalog),
		connectAccount: servicePlanCustomerCloudAccountRow{
			InstanceID:           "acct-pending",
			AccountConfigID:      "ac-123",
			CloudProvider:        "aws",
			Status:               "DEPLOYING",
			AWSAccountID:         "767925118737",
			AWSCloudFormationURL: "https://example.com/template.yml",
			SubscriptionID:       "sub-new",
		},
		refreshAccount: servicePlanCustomerCloudAccountRow{
			InstanceID:           "acct-pending",
			AccountConfigID:      "ac-123",
			CloudProvider:        "aws",
			Status:               "READY",
			AWSAccountID:         "767925118737",
			AWSCloudFormationURL: "https://example.com/template.yml",
			SubscriptionID:       "sub-new",
		},
	}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.detailPanelWidth = 80
	env := catalog.Services[0].Plans[0].Environments[0]
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Environment:             env,
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
	}, 80)
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	state.ParamFields[0].Input.SetValue("767925118737")
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)

	updated, cmd = model.Update(batch[1]())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.ConnectingAccount)
	rendered := model.renderDeploymentForm()
	require.Contains(t, rendered, "Waiting for account to become READY")
	require.NotContains(t, rendered, "Connecting aws account and waiting for READY")
	require.Contains(t, rendered, "Account config ID: ac-123")
	require.Contains(t, rendered, "CloudFormation template URL: https://example.com/template.yml")
	require.Contains(t, model.deploymentForm.deploymentHelpLine(), "c: copy template URL")

	updated, cmd = model.Update(servicePlanCustomerCloudAccountPollTickMsg{})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	updated, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.False(t, model.deploymentForm.ConnectingAccount)
	require.Equal(t, servicePlanDeploymentStepRegion, model.deploymentForm.currentStep())
	require.Equal(t, 1, loader.refreshCalls)
}

func TestServicePlanCustomerCloudAccountRowExtractsCloudIdentityFromInstance(t *testing.T) {
	row := servicePlanCustomerCloudAccountRowWithInstanceDetails(servicePlanCustomerCloudAccountRow{
		InstanceID: "instance-qu9xnfp2x",
		Status:     "FAILED",
	}, &openapiclientfleet.ResourceInstance{
		CloudProvider: "aws",
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			LaunchInputParams: map[string]any{
				servicePlanCustomerAccountAWSAccountIDKey: "123456789012",
			},
		},
	})

	require.Equal(t, "aws", row.CloudProvider)
	require.Equal(t, "123456789012", row.AWSAccountID)
	require.Equal(t, "AWS account 123456789012", servicePlanCustomerCloudAccountLabel(row))
}

func TestServicePlanDeploymentFormFieldEditsDirectlyAndEnterContinues(t *testing.T) {
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
	}, 80)
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	state.ParamFields[0].Input.SetValue("123")

	model := servicePlanBrowserModel{deploymentForm: &state, detailPanelWidth: 80}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, servicePlanDeploymentStepConnectAccount, model.deploymentForm.currentStep())
	require.Equal(t, "12", model.deploymentForm.ParamFields[0].Input.Value())

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, "124", model.deploymentForm.ParamFields[0].Input.Value())

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Equal(t, "customer cloud account connection is not available", model.deploymentForm.Err)
}

func TestServicePlanDeploymentInlineConnectSelectsNewAccountAndContinues(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{
		details:        testBrowserDetails(catalog),
		connectAccount: servicePlanCustomerCloudAccountRow{InstanceID: "acct-new", CloudProvider: "aws", Status: "READY", SubscriptionID: "sub-new"},
	}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.list.Select(2)
	model.syncSelectionFromList()
	env := model.selectedEnvironment()
	require.NotNil(t, env)

	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		Environment:             *env,
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		Customers:               []servicePlanDeploymentCustomer{{UserID: "user-1", Email: "alice@example.com", Name: "Alice"}},
	}, 80)
	require.NoError(t, state.advanceStep(80))
	require.NoError(t, state.advanceStep(80))
	state.ParamFields[0].Input.SetValue("123456789012")
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.ConnectingAccount)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)
	connected := batch[1]()
	updated, cmd = model.Update(connected)
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, 1, loader.connectCalls)
	require.Equal(t, "123456789012", loader.connectReq.Values[servicePlanCustomerAccountAWSAccountIDKey])
	require.False(t, model.deploymentForm.ConnectingAccount)
	require.Equal(t, "acct-new", model.deploymentForm.CustomerAccountID)
	require.Equal(t, servicePlanDeploymentStepRegion, model.deploymentForm.currentStep())
	require.Len(t, model.deploymentForm.Form.CustomerCloudAccounts, 1)
	require.Equal(t, "acct-new", model.deploymentForm.Form.CustomerCloudAccounts[0].InstanceID)
}

func TestServicePlanDeploymentFailedCloudAccountRetryAction(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{
		details:      testBrowserDetails(catalog),
		retryAccount: servicePlanCustomerCloudAccountRow{InstanceID: "acct-failed", CloudProvider: "aws", Status: "READY", AWSAccountID: "123456789012"},
	}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.detailPanelWidth = 80
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-failed", CloudProvider: "aws", Status: "FAILED", AWSAccountID: "123456789012"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.AccountActionRunning)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)
	updated, cmd = model.Update(batch[1]())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, 1, loader.retryCalls)
	require.Equal(t, "acct-failed", loader.retryReq.Account.InstanceID)
	require.False(t, model.deploymentForm.AccountActionRunning)
	require.Equal(t, "acct-failed", model.deploymentForm.CustomerAccountID)
	require.Equal(t, servicePlanDeploymentStepRegion, model.deploymentForm.currentStep())
}

func TestServicePlanDeploymentFailedCloudAccountDeleteAction(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.detailPanelWidth = 80
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-failed", CloudProvider: "aws", Status: "FAILED", AWSAccountID: "123456789012"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))
	state.AccountActionCursor = 1
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.AccountActionRunning)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)
	updated, cmd = model.Update(batch[1]())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, 1, loader.deleteCalls)
	require.Equal(t, "acct-failed", loader.deleteReq.Account.InstanceID)
	require.False(t, model.deploymentForm.AccountActionRunning)
	require.Empty(t, model.deploymentForm.Form.CustomerCloudAccounts)
	require.Equal(t, servicePlanDeploymentStepCloudAccount, model.deploymentForm.currentStep())
}

func TestServicePlanDeploymentReadyCloudAccountDeleteAction(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServicesWithHostingModels(), nil)
	loader := &fakeServicePlanBrowserLoader{details: testBrowserDetails(catalog)}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model.detailPanelWidth = 80
	state := newServicePlanDeploymentFormState(servicePlanDeploymentForm{
		AccountResource:         servicePlanDeploymentResource{ID: "r-injectedaccountconfig", Name: "Cloud Provider Account", URLKey: "omnistrateCloudAccountConfig"},
		Resources:               []servicePlanDeploymentResource{{ID: "res-1", Name: "API", URLKey: "api"}},
		CloudProviders:          []string{"aws"},
		RegionsByCloud:          map[string][]string{"aws": []string{"us-west-2"}},
		RequiresCustomerAccount: true,
		CustomerCloudAccounts: []servicePlanCustomerCloudAccountRow{
			{InstanceID: "acct-ready", CloudProvider: "aws", Status: "READY", AWSAccountID: "123456789012"},
		},
	}, 80)
	require.NoError(t, state.advanceStep(80))

	options := state.currentOptions()
	require.Len(t, options, 2)
	buttons := state.customerCloudAccountActionButtons(options[0])
	require.Len(t, buttons, 2)
	require.Equal(t, "Use", buttons[0].Label)
	require.Equal(t, "Delete", buttons[1].Label)

	state.AccountActionCursor = 1
	model.deploymentForm = &state

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.True(t, model.deploymentForm.AccountActionRunning)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)
	updated, cmd = model.Update(batch[1]())
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)

	require.Equal(t, 1, loader.deleteCalls)
	require.Equal(t, "acct-ready", loader.deleteReq.Account.InstanceID)
	require.False(t, model.deploymentForm.AccountActionRunning)
	require.Empty(t, model.deploymentForm.Form.CustomerCloudAccounts)
	require.Equal(t, servicePlanDeploymentStepCloudAccount, model.deploymentForm.currentStep())
}

func TestServicePlanBrowserRefreshHotkeyReloadsRightPane(t *testing.T) {
	catalog := buildServicePlanBrowserCatalog(testBrowserServices(), nil)
	details := testBrowserDetails(catalog)
	loader := &fakeServicePlanBrowserLoader{details: details}
	model := newServicePlanBrowserModel(context.Background(), "token", catalog, loader)
	model = loadSelectedTestDetails(t, model)
	env := model.selectedEnvironment()
	require.NotNil(t, env)
	require.Contains(t, model.viewport.View(), "OMNISTRATE_HOSTED")

	updatedDetail := details[env.cacheKey()]
	updatedDetail.DeploymentModel = "UPDATED_MODEL"
	loader.details[env.cacheKey()] = updatedDetail

	updated, cmd := model.Update(tea.KeyMsg{Runes: []rune{'r'}, Type: tea.KeyRunes})
	require.NotNil(t, cmd)
	model = updated.(servicePlanBrowserModel)
	require.Contains(t, model.viewport.View(), "Loading details")

	model = loadSelectedTestDetails(t, model)
	require.Contains(t, model.viewport.View(), "UPDATED_MODEL")
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

func servicePlanDeploymentFormFieldLabels(fields []servicePlanDeploymentFormField) []string {
	labels := make([]string, 0, len(fields))
	for _, field := range fields {
		labels = append(labels, field.Label)
	}
	return labels
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
