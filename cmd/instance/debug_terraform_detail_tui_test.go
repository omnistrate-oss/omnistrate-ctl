package instance

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
