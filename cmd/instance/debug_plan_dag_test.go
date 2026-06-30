package instance

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func TestComputePlanLevels(t *testing.T) {
	nodes := map[string]PlanDAGNode{
		"A": {ID: "A", Name: "A"},
		"B": {ID: "B", Name: "B"},
		"C": {ID: "C", Name: "C"},
	}
	edges := []PlanDAGEdge{
		{From: "A", To: "C"},
		{From: "B", To: "C"},
	}

	levels, hasCycle := computePlanLevels(nodes, edges)
	if hasCycle {
		t.Fatalf("unexpected cycle detected")
	}
	if len(levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(levels))
	}
}

func TestDagTopLevelTabNames(t *testing.T) {
	if len(dagTabNames) != dagNumTabs {
		t.Fatalf("dagTabNames length %d does not match dagNumTabs %d", len(dagTabNames), dagNumTabs)
	}
	if dagTabNames[dagTabResources] != "Resource Details" {
		t.Fatalf("expected resource details tab, got %q", dagTabNames[dagTabResources])
	}
	if dagTabNames[dagTabMetrics] != "Metrics" {
		t.Fatalf("expected metrics tab, got %q", dagTabNames[dagTabMetrics])
	}
}

func TestDagTopLevelTabSwitching(t *testing.T) {
	model := newDagModel(DebugData{
		PlanDAG: &PlanDAG{
			Nodes: map[string]PlanDAGNode{
				"r-postgres": {ID: "r-postgres", Key: "postgres", Name: "postgres", Type: "Resource"},
			},
			Levels: [][]string{{"r-postgres"}},
		},
	})
	model.width = 100
	model.height = 30
	model.rebuildLayout()

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("expected nil cmd on top-level tab switch, got %v", cmd)
	}
	updated := updatedAny.(dagModel)
	if updated.activeTab != dagTabMetrics {
		t.Fatalf("expected metrics tab after tab key, got %d", updated.activeTab)
	}

	updatedAny, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = updatedAny.(dagModel)
	if updated.activeTab != dagTabResources {
		t.Fatalf("expected resource details tab after shift+tab, got %d", updated.activeTab)
	}
}

func TestTerminalHyperlinkRejectsControlCharacters(t *testing.T) {
	label := "dashboard"

	if got := terminalHyperlink("https://grafana.example.com/d/overview", label); got == label {
		t.Fatalf("expected safe URL to render as terminal hyperlink")
	}
	for _, unsafeURL := range []string{
		"https://grafana.example.com/d/overview\x1b]0;owned",
		"https://grafana.example.com/d/overview\nnext",
		"https://grafana.example.com/d/overview\x07",
	} {
		if got := terminalHyperlink(unsafeURL, label); got != label {
			t.Fatalf("expected unsafe URL %q to render plain label, got %q", unsafeURL, got)
		}
	}
}

func TestMetricsTreeShowsCredentialsOnlyWhenExpanded(t *testing.T) {
	model := newDagModel(DebugData{
		DashboardCatalog: &dataaccess.DashboardCatalog{
			Features: []dataaccess.DashboardFeatureInfo{
				{
					Key:               "METRICS",
					Label:             "Customer",
					GrafanaEndpoint:   "https://grafana.example.com",
					GrafanaUIUsername: "org-user",
					GrafanaUIPassword: "plain-secret",
					Dashboards: []dataaccess.DashboardRef{
						{Name: "overview", URL: "https://grafana.example.com/d/overview"},
					},
				},
			},
		},
	})
	model.width = 100
	model.height = 30

	collapsed, _ := model.metricsTreeLines(96)
	collapsedText := strings.Join(collapsed, "\n")
	if strings.Contains(collapsedText, "org-user") || strings.Contains(collapsedText, "plain-secret") {
		t.Fatalf("expected collapsed metrics tree to omit credentials, got %q", collapsedText)
	}

	updatedAny, _ := model.updateMetricsDashboard(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedAny.(dagModel)
	expanded, _ := updated.metricsTreeLines(96)
	expandedText := strings.Join(expanded, "\n")
	if !strings.Contains(expandedText, "username: org-user") {
		t.Fatalf("expected expanded metrics tree to include username, got %q", expandedText)
	}
	if !strings.Contains(expandedText, "password: plain-secret") {
		t.Fatalf("expected expanded metrics tree to include password, got %q", expandedText)
	}
}

func TestComputePlanLevelsCycle(t *testing.T) {
	nodes := map[string]PlanDAGNode{
		"A": {ID: "A", Name: "A"},
		"B": {ID: "B", Name: "B"},
	}
	edges := []PlanDAGEdge{
		{From: "A", To: "B"},
		{From: "B", To: "A"},
	}

	levels, hasCycle := computePlanLevels(nodes, edges)
	if !hasCycle {
		t.Fatalf("expected cycle to be detected")
	}
	if len(levels) == 0 {
		t.Fatalf("expected levels to be populated for cycle case")
	}
}

func TestAttachBreakpointStatuses(t *testing.T) {
	startTerraformPlan := "StartTerraformPlan"
	completeTerraformPlan := "CompleteTerraformPlan"

	plan := &PlanDAG{
		Nodes: map[string]PlanDAGNode{
			"res-writer": {ID: "res-writer", Key: "writer", Name: "Writer"},
			"res-reader": {ID: "res-reader", Key: "reader", Name: "Reader"},
		},
	}

	instanceData := &openapiclientfleet.ResourceInstance{
		ActiveBreakpoints: []openapiclientfleet.WorkflowBreakpointWithStatus{
			{Id: "writer", Event: &startTerraformPlan, Status: "HIT"},
			{Id: "writer", Event: &completeTerraformPlan, Status: "pending"},
			{Id: "res-reader", Status: "pending"},
		},
	}

	attachBreakpointStatuses(plan, instanceData)

	if got := plan.BreakpointByID["res-writer"]; got != "hit" {
		t.Fatalf("expected writer breakpoint status to be hit, got %q", got)
	}
	if got := plan.BreakpointByKey["writer"]; got != "hit" {
		t.Fatalf("expected writer key breakpoint status to be hit, got %q", got)
	}
	if got := plan.BreakpointByID["res-reader"]; got != "pending" {
		t.Fatalf("expected reader breakpoint status to be pending, got %q", got)
	}
}

func TestBreakpointStatusForNode(t *testing.T) {
	plan := &PlanDAG{
		BreakpointByID: map[string]string{
			"res-writer": "hit",
		},
	}

	status, ok := breakpointStatusForNode(plan, PlanDAGNode{ID: "res-writer", Key: "writer"})
	if !ok {
		t.Fatalf("expected breakpoint status to be found")
	}
	if status != "hit" {
		t.Fatalf("expected status hit, got %q", status)
	}
}

func TestComposeResourceTypeTagAndIcon(t *testing.T) {
	tag := formatTypeTag("DockerCompose")
	if tag != "Compose" {
		t.Fatalf("expected Compose tag, got %q", tag)
	}

	icon, _ := iconForType(tag, cardTheme{icon: "255"})
	if icon != 'C' {
		t.Fatalf("expected Compose icon C, got %q", icon)
	}
}

func TestEmptyResourceTypeTagAndIcon(t *testing.T) {
	tag := formatTypeTag("")
	if tag != "Compose" {
		t.Fatalf("expected empty type to render as Compose, got %q", tag)
	}
}

func TestResourceTypeOpensComposeDetail(t *testing.T) {
	model := dagModel{
		debugData: DebugData{},
		plan: &PlanDAG{
			Nodes: map[string]PlanDAGNode{
				"r-postgres": {ID: "r-postgres", Key: "postgres", Name: "postgres", Type: "Resource"},
			},
			Levels: [][]string{{"r-postgres"}},
		},
		selectableNodes: []string{"r-postgres"},
		cursorIndex:     0,
		width:           100,
		height:          30,
	}

	updated, cmd := model.openNodeDetail()
	updatedModel := updated.(dagModel)

	if !updatedModel.inDetail {
		t.Fatalf("expected resource node to enter detail view")
	}
	if updatedModel.detailModel == nil {
		t.Fatalf("expected resource node to open a detail model")
	}
	if _, ok := updatedModel.detailModel.(composeDetailModel); !ok {
		t.Fatalf("expected resource node to open compose detail model, got %T", updatedModel.detailModel)
	}
	if cmd == nil {
		t.Fatalf("expected detail init command")
	}
}

func TestEmptyResourceTypeOpensComposeDetail(t *testing.T) {
	model := dagModel{
		debugData: DebugData{},
		plan: &PlanDAG{
			Nodes: map[string]PlanDAGNode{
				"r-postgres": {ID: "r-postgres", Key: "postgres", Name: "postgres"},
			},
			Levels: [][]string{{"r-postgres"}},
		},
		selectableNodes: []string{"r-postgres"},
		cursorIndex:     0,
		width:           100,
		height:          30,
	}

	updated, cmd := model.openNodeDetail()
	updatedModel := updated.(dagModel)

	if !updatedModel.inDetail {
		t.Fatalf("expected empty-type node to enter detail view")
	}
	if updatedModel.detailModel == nil {
		t.Fatalf("expected empty-type node to open a detail model")
	}
	if _, ok := updatedModel.detailModel.(composeDetailModel); !ok {
		t.Fatalf("expected empty-type node to open compose detail model, got %T", updatedModel.detailModel)
	}
	if cmd == nil {
		t.Fatalf("expected detail init command")
	}
}

func TestPlanHasHitBreakpoint(t *testing.T) {
	planWithoutHit := &PlanDAG{
		BreakpointByID: map[string]string{
			"res-writer": "pending",
		},
	}
	if planHasHitBreakpoint(planWithoutHit) {
		t.Fatalf("expected no hit breakpoints")
	}

	planWithHit := &PlanDAG{
		BreakpointByKey: map[string]string{
			"writer": "hit",
		},
	}
	if !planHasHitBreakpoint(planWithHit) {
		t.Fatalf("expected hit breakpoints")
	}
}
