package instance

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestTerraformTabNames(t *testing.T) {
	require.Equal(t, numTabs, len(tabNames), "tabNames length must match numTabs")
	require.Equal(t, "Progress", tabNames[tabProgress])
	require.Equal(t, "Terraform Files", tabNames[tabTfFiles])
	require.Equal(t, "Terraform Output", tabNames[tabTfOutput])
	require.Equal(t, "Live Logs", tabNames[tabLogs])
	require.Equal(t, "Operation History", tabNames[tabOpHistory])
	require.Equal(t, "Workflow Events", tabNames[tabWfErrors])
}

func TestTerraformDetailTabsOnlySwitchOnTabKeys(t *testing.T) {
	model := newTerraformDetailModel(PlanDAGNode{}, DebugData{})

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd != nil {
		t.Fatalf("expected nil cmd on right key, got %v", cmd)
	}
	updated, ok := updatedAny.(terraformDetailModel)
	if !ok {
		t.Fatalf("expected terraformDetailModel, got %T", updatedAny)
	}
	if updated.activeTab != tabProgress {
		t.Fatalf("expected right key to keep active tab at %d, got %d", tabProgress, updated.activeTab)
	}

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("expected nil cmd on tab key, got %v", cmd)
	}
	updated, ok = updatedAny.(terraformDetailModel)
	if !ok {
		t.Fatalf("expected terraformDetailModel, got %T", updatedAny)
	}
	if updated.activeTab != tabTfFiles {
		t.Fatalf("expected tab key to advance to %d, got %d", tabTfFiles, updated.activeTab)
	}

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd != nil {
		t.Fatalf("expected nil cmd on left key, got %v", cmd)
	}
	updated, ok = updatedAny.(terraformDetailModel)
	if !ok {
		t.Fatalf("expected terraformDetailModel, got %T", updatedAny)
	}
	if updated.activeTab != tabTfFiles {
		t.Fatalf("expected left key to keep active tab at %d, got %d", tabTfFiles, updated.activeTab)
	}

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd != nil {
		t.Fatalf("expected nil cmd on shift+tab key, got %v", cmd)
	}
	updated, ok = updatedAny.(terraformDetailModel)
	if !ok {
		t.Fatalf("expected terraformDetailModel, got %T", updatedAny)
	}
	if updated.activeTab != tabProgress {
		t.Fatalf("expected shift+tab key to return to %d, got %d", tabProgress, updated.activeTab)
	}
}

func TestTerraformOperationSummaryPrefersExecutionState(t *testing.T) {
	got := terraformOperationSummary(
		TerraformExecutionState{
			Operation:       "output",
			Status:          "completed",
			ResourceVersion: "48.0",
			CompletedAt:     "2026-06-08T15:44:32Z",
		},
		&TerraformProgressData{Status: "running", ResourceVersion: "47.0"},
		[]TerraformHistoryEntry{{Operation: "apply", Status: "running"}},
	)
	want := "tf-state output completed (rv 48.0, completed 2026-06-08T15:44:32Z)"
	if got != want {
		t.Fatalf("terraformOperationSummary() = %q, want %q", got, want)
	}
}

func TestTerraformOperationSummaryFallsBackToHistory(t *testing.T) {
	got := terraformOperationSummary(
		TerraformExecutionState{},
		nil,
		[]TerraformHistoryEntry{
			{Operation: "diff", Status: "completed"},
			{Operation: "apply", Status: "failed", CompletedAt: "2026-06-08T15:40:00Z"},
		},
	)
	want := "history apply failed (completed 2026-06-08T15:40:00Z)"
	if got != want {
		t.Fatalf("terraformOperationSummary() = %q, want %q", got, want)
	}
}

func TestIsProgressInFlightUsesExecutionState(t *testing.T) {
	model := newTerraformDetailModel(PlanDAGNode{}, DebugData{})
	model.tfExecutionState = TerraformExecutionState{Operation: "output", Status: "completed", CompletedAt: "2026-06-08T15:44:32Z"}
	model.tfProgress = &TerraformProgressData{Status: "running", TotalResources: 1}

	if model.isProgressInFlight() {
		t.Fatal("expected completed execution state to override stale running progress")
	}
}
