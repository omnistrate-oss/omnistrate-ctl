package tui

import "testing"

func TestSoftWrapLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxWidth int
		want     []string
	}{
		{
			name:     "short line is not wrapped",
			line:     "hello",
			maxWidth: 10,
			want:     []string{"hello"},
		},
		{
			name:     "exact width is not wrapped",
			line:     "hello",
			maxWidth: 5,
			want:     []string{"hello"},
		},
		{
			name:     "long line wraps at width boundary",
			line:     "abcdefgh",
			maxWidth: 3,
			want:     []string{"abc", "def", "gh"},
		},
		{
			name:     "zero width returns line unchanged",
			line:     "abcdef",
			maxWidth: 0,
			want:     []string{"abcdef"},
		},
		{
			name:     "negative width returns line unchanged",
			line:     "abcdef",
			maxWidth: -1,
			want:     []string{"abcdef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SoftWrapLine(tt.line, tt.maxWidth)
			if len(got) != len(tt.want) {
				t.Fatalf("SoftWrapLine(%q, %d) = %v, want %v", tt.line, tt.maxWidth, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExpandLinesToVisual(t *testing.T) {
	source := []string{"abcdef", "x", ""}
	got := ExpandLinesToVisual(source, 3)

	want := []VisualLine{
		{Text: "abc", SourceNum: 1},
		{Text: "def", SourceNum: 0}, // continuation of source line 1
		{Text: "x", SourceNum: 2},
		{Text: "", SourceNum: 3},
	}

	if len(got) != len(want) {
		t.Fatalf("ExpandLinesToVisual returned %d lines, want %d: %+v", len(got), len(want), got)
	}
	for i := range got {
		if got[i].Text != want[i].Text || got[i].SourceNum != want[i].SourceNum {
			t.Errorf("visual line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestExpandLinesToVisualEmpty(t *testing.T) {
	if got := ExpandLinesToVisual(nil, 10); len(got) != 0 {
		t.Errorf("expected no visual lines for nil input, got %+v", got)
	}
}
