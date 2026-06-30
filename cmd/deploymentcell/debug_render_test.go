package deploymentcell

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func lineCount(s string) int { return strings.Count(s, "\n") + 1 }

func assertWithinWidth(t *testing.T, view string, maxWidth int) {
	t.Helper()
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > maxWidth {
			t.Errorf("line %d exceeds terminal width %d (got %d): %q", i, maxWidth, w, line)
		}
	}
}

// TestListAndDetailFitTerminal is the regression test for the original bug:
// large amenity content used to overflow the terminal and reflow the layout.
// Both screens must render to exactly the terminal height and never wider than
// the terminal width, regardless of content size, and the detail body must
// scroll.
func TestListAndDetailFitTerminal(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("{\n")
	for i := 0; i < 300; i++ {
		sb.WriteString(fmt.Sprintf("  \"key%d\": \"a deliberately long value used to force soft wrapping across the viewport width %d\",\n", i, i))
	}
	sb.WriteString("  \"last\": true\n}\n")
	enc := base64.StdEncoding.EncodeToString([]byte(sb.String()))

	data := deploymentCellDebugData{
		DeploymentCellID: "hc-123",
		AmenityStatuses: []deploymentCellAmenityStatus{
			{Name: "postgres-amenity", Type: amenityTypeHelm, DesiredStatus: "DEPLOYING"},
			{Name: "redis-amenity", Type: amenityTypeHelm, DesiredStatus: "READY"},
		},
		AmenityArtifacts: []deploymentCellAmenityArtifact{
			{AmenityName: "postgres-amenity", ArtifactKind: artifactHelmValuesRendered, PayloadBase64: &enc},
		},
	}

	const w, h = 80, 24
	var m tea.Model = newDeploymentCellDebugModel(data)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})

	listView := m.View()
	if got := lineCount(listView); got != h {
		t.Errorf("list view height = %d, want %d", got, h)
	}
	assertWithinWidth(t, listView, w)

	// Open the selected amenity's detail screen.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	detailView := m.View()
	if got := lineCount(detailView); got != h {
		t.Errorf("detail view height = %d, want %d (content must not overflow)", got, h)
	}
	assertWithinWidth(t, detailView, w)

	// Page down: still fits, and the visible window changed.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := m.View()
	if got := lineCount(scrolled); got != h {
		t.Errorf("scrolled view height = %d, want %d", got, h)
	}
	assertWithinWidth(t, scrolled, w)
	if scrolled == detailView {
		t.Error("expected detail content to change after paging down")
	}

	// esc returns to the list screen. The detail emits backToListMsg via a Cmd,
	// which the runtime would execute — simulate that here.
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		m, _ = m.Update(cmd())
	}
	listAgain := m.View()
	if got := lineCount(listAgain); got != h {
		t.Errorf("after esc, list view height = %d, want %d", got, h)
	}
	if !strings.Contains(listAgain, "enter: open") {
		t.Errorf("after esc, expected to be back on the list screen:\n%s", listAgain)
	}
}

// TestDetailHandlesTinyTerminal ensures clamping prevents panics / overflow on
// very small terminals.
func TestDetailHandlesTinyTerminal(t *testing.T) {
	data := deploymentCellDebugData{
		DeploymentCellID: "hc-1",
		AmenityStatuses:  []deploymentCellAmenityStatus{{Name: "a", Type: amenityTypeHelm, DesiredStatus: "READY"}},
	}
	var m tea.Model = newDeploymentCellDebugModel(data)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 10, Height: 4})
	_ = m.View() // must not panic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m.View() // detail must not panic
}
