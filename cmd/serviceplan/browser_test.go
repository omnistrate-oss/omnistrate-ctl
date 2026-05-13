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
	details     map[string]servicePlanEnvironmentDetails
	calls       []string
	form        servicePlanDeploymentForm
	formErr     error
	formCalls   []string
	launchID    string
	launchErr   error
	launchReq   *servicePlanDeploymentLaunchRequest
	launchCalls int
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
			{InstanceID: "acct-1", CloudProvider: "aws", Status: "RUNNING", Region: "us-west-2", CustomerEmail: "alice@example.com", SubscriptionID: "sub-1"},
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
	require.Contains(t, model.modal.Rows[0].Text, "acct-1")
	require.Contains(t, model.modal.Rows[0].Text, "alice@example.com")
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
	require.Contains(t, model.renderDeploymentForm(), "Resource")

	for i := 0; i < 4; i++ {
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
