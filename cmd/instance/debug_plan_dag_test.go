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
