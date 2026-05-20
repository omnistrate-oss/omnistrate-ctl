package instance

import (
	"testing"

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
	plan := &PlanDAG{
		Nodes: map[string]PlanDAGNode{
			"res-writer": {ID: "res-writer", Key: "writer", Name: "Writer"},
			"res-reader": {ID: "res-reader", Key: "reader", Name: "Reader"},
		},
	}

	instanceData := &openapiclientfleet.ResourceInstance{
		ActiveBreakpoints: []openapiclientfleet.WorkflowBreakpointWithStatus{
			{Id: "writer", Status: "HIT"},
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

func TestDeploymentTypesByResourceIdentityMarksGenericDeploymentAsCompose(t *testing.T) {
	resourceID := "r-postgres"
	resourceName := "postgres"
	summaries := []openapiclientfleet.ResourceVersionSummary{
		{
			ResourceId:                             &resourceID,
			ResourceName:                           &resourceName,
			GenericResourceDeploymentConfiguration: &openapiclientfleet.GenericResourceDeploymentConfiguration{},
		},
	}

	types := deploymentTypesByResourceIdentity(summaries)

	if got := types.byID[resourceID]; got != "Compose" {
		t.Fatalf("expected compose deployment type by ID, got %q", got)
	}
	if got := types.byName[resourceName]; got != "Compose" {
		t.Fatalf("expected compose deployment type by name, got %q", got)
	}
}

func TestMergePlanNodeDeploymentTypeUsesComposeForGenericResource(t *testing.T) {
	deploymentTypes := resourceDeploymentTypes{
		byID: map[string]string{
			"r-postgres": "Compose",
		},
		byName: map[string]string{},
	}
	node := PlanDAGNode{ID: "r-postgres", Key: "postgres", Name: "postgres"}

	if got := mergePlanNodeDeploymentType(node, "Resource", deploymentTypes); got != "Compose" {
		t.Fatalf("expected compose deployment type, got %q", got)
	}
}

func TestMergePlanNodeDeploymentTypePreservesSpecificType(t *testing.T) {
	deploymentTypes := resourceDeploymentTypes{
		byID: map[string]string{
			"r-operator": "Compose",
		},
		byName: map[string]string{},
	}
	node := PlanDAGNode{ID: "r-operator", Key: "operator", Name: "operator"}

	if got := mergePlanNodeDeploymentType(node, "OperatorCRD", deploymentTypes); got != "OperatorCRD" {
		t.Fatalf("expected specific type to be preserved, got %q", got)
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
