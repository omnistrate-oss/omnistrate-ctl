package instance

import "testing"

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
