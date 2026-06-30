package deploymentcell

import (
	"encoding/base64"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func b64(s string) *string {
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	return &enc
}

func TestRawContentForView(t *testing.T) {
	data := deploymentCellDebugData{
		AmenityArtifacts: []deploymentCellAmenityArtifact{
			{AmenityName: "pg", ArtifactKind: artifactHelmValuesRendered, PayloadBase64: b64("replicaCount: 3\nimage:\n  tag: latest\n")},
		},
	}
	status := deploymentCellAmenityStatus{Name: "pg", Type: amenityTypeHelm}

	t.Run("rendered values converted to pretty json", func(t *testing.T) {
		got := rawContentForView(data, status, "rendered values")
		if !strings.Contains(got, "\"replicaCount\": 3") {
			t.Errorf("expected pretty JSON values, got:\n%s", got)
		}
	})

	t.Run("missing artifact returns placeholder", func(t *testing.T) {
		other := deploymentCellAmenityStatus{Name: "redis", Type: amenityTypeHelm}
		got := rawContentForView(data, other, "rendered values")
		if !strings.Contains(got, "No "+artifactHelmValuesRendered+" artifact found.") {
			t.Errorf("expected not-found placeholder, got: %q", got)
		}
	})
}

func TestAvailableViewsFor(t *testing.T) {
	if got := availableViewsFor(deploymentCellAmenityStatus{Type: amenityTypeHelm}); len(got) != 1 || got[0] != "rendered values" {
		t.Errorf("helm views wrong: %v", got)
	}
	if got := availableViewsFor(deploymentCellAmenityStatus{Type: amenityTypeKubernetesManifest}); len(got) != 1 || got[0] != "rendered manifest" {
		t.Errorf("manifest views wrong: %v", got)
	}
	if got := availableViewsFor(deploymentCellAmenityStatus{Type: "Other"}); len(got) != 1 || got[0] != "status" {
		t.Errorf("default views wrong: %v", got)
	}

	managed := true
	if got := availableViewsFor(deploymentCellAmenityStatus{Type: amenityTypeHelm, IsManaged: &managed}); got != nil {
		t.Errorf("managed amenity should have no views, got: %v", got)
	}
}

func TestManagedDetailShowsBadgeOnly(t *testing.T) {
	managed := true
	status := deploymentCellAmenityStatus{Name: "obs", Type: amenityTypeHelm, IsManaged: &managed}
	// Even with a rendered-values artifact present, a managed amenity must not
	// expose it.
	data := deploymentCellDebugData{
		AmenityArtifacts: []deploymentCellAmenityArtifact{
			{AmenityName: "obs", ArtifactKind: artifactHelmValuesRendered, PayloadBase64: b64("secret: hunter2\n")},
		},
	}
	m := newAmenityDetailModel(data, status, 80, 24)
	view := m.View()
	if !strings.Contains(view, "Omnistrate Managed") {
		t.Errorf("expected managed badge, got:\n%s", view)
	}
	if strings.Contains(view, "hunter2") || strings.Contains(view, "secret") {
		t.Errorf("managed amenity must not expose values:\n%s", view)
	}
	if strings.Contains(view, "Rendered Values") {
		t.Errorf("managed amenity must not show value tabs:\n%s", view)
	}
}

func TestMaxScroll(t *testing.T) {
	cases := []struct{ total, bodyH, want int }{
		{10, 5, 5},
		{5, 5, 0},
		{3, 5, 0},
		{0, 5, 0},
	}
	for _, c := range cases {
		if got := maxScroll(c.total, c.bodyH); got != c.want {
			t.Errorf("maxScroll(%d,%d) = %d, want %d", c.total, c.bodyH, got, c.want)
		}
	}
}

func TestPositionLabel(t *testing.T) {
	cases := []struct {
		scroll, bodyH, total int
		want                 string
	}{
		{0, 5, 3, "[all]"},
		{0, 5, 10, "[top]"},
		{5, 5, 10, "[end]"},
		{3, 5, 20, "[20%]"}, // maxScroll=15, 3*100/15=20
	}
	for _, c := range cases {
		if got := positionLabel(c.scroll, c.bodyH, c.total); got != c.want {
			t.Errorf("positionLabel(%d,%d,%d) = %q, want %q", c.scroll, c.bodyH, c.total, got, c.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	if got := displayName("rendered values"); got != "Rendered Values" {
		t.Errorf("displayName = %q, want %q", got, "Rendered Values")
	}
	if got := displayName("helm logs"); got != "Helm Logs" {
		t.Errorf("displayName = %q, want %q", got, "Helm Logs")
	}
}

func TestIsJSONView(t *testing.T) {
	for _, v := range []string{"rendered values", "template values"} {
		if !isJSONView(v) {
			t.Errorf("expected %q to be a JSON view", v)
		}
	}
	for _, v := range []string{"helm logs", "rendered manifest", "cluster status", "status"} {
		if isJSONView(v) {
			t.Errorf("expected %q to NOT be a JSON view", v)
		}
	}
}

func TestPadOrTruncate(t *testing.T) {
	if got := padOrTruncate("ab", 5); got != "ab   " {
		t.Errorf("pad: got %q", got)
	}
	if got := padOrTruncate("abcdef", 3); got != "abc" {
		t.Errorf("truncate: got %q", got)
	}
	if got := padOrTruncate("abc", 3); got != "abc" {
		t.Errorf("exact: got %q", got)
	}
}

func TestWorkflowLineInDetailHeaderOmitsMissing(t *testing.T) {
	// A status with neither ID should produce an empty workflow segment, so the
	// detail header meta line never shows a bare "workflow=" / "run=".
	status := deploymentCellAmenityStatus{Name: "x", Type: amenityTypeHelm, Source: strPtr("svc")}
	m := newAmenityDetailModel(deploymentCellDebugData{}, status, 80, 24)
	header := m.renderHeader()
	if strings.Contains(header, "workflow=") || strings.Contains(header, "run=") {
		t.Errorf("header should not show workflow/run when absent:\n%s", header)
	}
	if !strings.Contains(header, "source=svc") {
		t.Errorf("header should show source:\n%s", header)
	}
}
