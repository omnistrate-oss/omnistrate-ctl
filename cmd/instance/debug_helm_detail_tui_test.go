package instance

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHelmValuesEnterTogglesSelectedNode(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabValues,
		valuesTree: buildHelmValuesTree(map[string]interface{}{
			"parent": map[string]interface{}{
				"child": "value",
			},
		}, ""),
	}

	visibleNodes := flattenOutputTree(model.valuesTree)
	if len(visibleNodes) != 2 {
		t.Fatalf("expected 2 visible nodes before toggle, got %d", len(visibleNodes))
	}
	if !visibleNodes[0].expandable || !visibleNodes[0].expanded {
		t.Fatalf("expected root node to start expanded")
	}

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd on first toggle, got %v", cmd)
	}
	updated, ok := updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	visibleNodes = flattenOutputTree(updated.valuesTree)
	if len(visibleNodes) != 1 {
		t.Fatalf("expected 1 visible node after collapse, got %d", len(visibleNodes))
	}
	if visibleNodes[0].expanded {
		t.Fatalf("expected root node to collapse on enter")
	}

	updatedAny, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd on second toggle, got %v", cmd)
	}
	updated, ok = updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	visibleNodes = flattenOutputTree(updated.valuesTree)
	if len(visibleNodes) != 2 {
		t.Fatalf("expected 2 visible nodes after re-expand, got %d", len(visibleNodes))
	}
	if !visibleNodes[0].expanded {
		t.Fatalf("expected root node to expand again on enter")
	}
}

func TestRenderHelmValuesTabClampsLongRows(t *testing.T) {
	values := map[string]interface{}{
		"alpha":  "short",
		"script": "line one\n" + strings.Repeat("x", 120) + "\nline three",
		"zeta":   "tail",
	}

	model := helmDetailModel{
		activeTab:    helmTabValues,
		width:        72,
		height:       14,
		helmData:     &HelmData{ChartRepoName: "repo/chart", ChartValues: values},
		valuesTree:   buildHelmValuesTree(values, ""),
		valuesCursor: 1,
	}

	rendered := model.renderHelmValuesTab()
	if !strings.Contains(rendered, "Chart Values") {
		t.Fatalf("expected chart values header in output")
	}

	maxWidth := model.helmContentWidth()
	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > maxWidth {
			t.Fatalf("rendered line width %d exceeded content width %d: %q", lipgloss.Width(line), maxWidth, line)
		}
	}
}

func TestHelmValuesUpFromBottomKeepsViewportFixed(t *testing.T) {
	model := helmDetailModel{
		activeTab: helmTabValues,
		width:     72,
		height:    15, // yields 3 visible value rows
		wfErrors:  &workflowErrorsState{},
		valuesTree: buildHelmValuesTree(map[string]interface{}{
			"a": 1,
			"b": 2,
			"c": 3,
			"d": 4,
			"e": 5,
			"f": 6,
		}, ""),
	}

	for range 5 {
		updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		if cmd != nil {
			t.Fatalf("expected nil cmd while moving down, got %v", cmd)
		}
		updated, ok := updatedAny.(helmDetailModel)
		if !ok {
			t.Fatalf("expected helmDetailModel, got %T", updatedAny)
		}
		model = updated
	}

	if model.valuesCursor != 5 || model.valuesScroll != 3 {
		t.Fatalf("expected cursor/scroll at bottom to be 5/3, got %d/%d", model.valuesCursor, model.valuesScroll)
	}

	updatedAny, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd while moving up, got %v", cmd)
	}
	updated, ok := updatedAny.(helmDetailModel)
	if !ok {
		t.Fatalf("expected helmDetailModel, got %T", updatedAny)
	}

	if updated.valuesCursor != 4 {
		t.Fatalf("expected cursor to move to 4, got %d", updated.valuesCursor)
	}
	if updated.valuesScroll != 3 {
		t.Fatalf("expected viewport to stay anchored at scroll 3, got %d", updated.valuesScroll)
	}
}
