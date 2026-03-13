package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeDiff_NewFile(t *testing.T) {
	lines := computeDiff("", "line1\nline2\nline3", "test.yaml")
	require.NotEmpty(t, lines)
	// First line is hunk header
	require.Equal(t, byte('@'), lines[0].kind)
	require.Contains(t, lines[0].text, "new file")
	// All other lines are additions
	for _, l := range lines[1:] {
		require.Equal(t, byte('+'), l.kind)
	}
}

func TestComputeDiff_ModifiedFile(t *testing.T) {
	old := "line1\nline2\nline3"
	modified := "line1\nmodified\nline3"
	lines := computeDiff(old, modified, "test.yaml")
	require.NotEmpty(t, lines)

	var hasAdd, hasRemove bool
	for _, l := range lines {
		if l.kind == '+' {
			hasAdd = true
		}
		if l.kind == '-' {
			hasRemove = true
		}
	}
	require.True(t, hasAdd, "diff should contain additions")
	require.True(t, hasRemove, "diff should contain removals")
}

func TestComputeDiff_Identical(t *testing.T) {
	old := "line1\nline2"
	lines := computeDiff(old, old, "test.yaml")
	// Should have hunk header + context lines only (no +/-)
	for _, l := range lines {
		if l.kind == '@' {
			continue
		}
		require.Equal(t, byte(' '), l.kind, "identical content should only have context lines")
	}
}

func TestDiffReview_StartReview_SkipsIdentical(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	os.Chdir(tmpDir) //nolint:errcheck

	// Write existing file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte("same content"), 0600))

	dr := newDiffReview()
	dr.startReview([]fileBlock{
		{name: "test.yaml", lang: "yaml", content: "same content"},
	})

	require.False(t, dr.active, "should not activate review for identical files")
}

func TestDiffReview_AcceptWrite(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	os.Chdir(tmpDir) //nolint:errcheck

	dr := newDiffReview()
	dr.startReview([]fileBlock{
		{name: "new.yaml", lang: "yaml", content: "services:\n  web:\n    image: nginx\n"},
	})

	require.True(t, dr.active)
	require.NotNil(t, dr.current())
	require.Equal(t, "new.yaml", dr.current().filename)

	filename, err := dr.accept()
	require.NoError(t, err)
	require.Equal(t, "new.yaml", filename)

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(tmpDir, "new.yaml"))
	require.NoError(t, err)
	require.Equal(t, "services:\n  web:\n    image: nginx\n", string(data))
}

func TestDiffReview_RejectDoesNotWrite(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	os.Chdir(tmpDir) //nolint:errcheck

	dr := newDiffReview()
	dr.startReview([]fileBlock{
		{name: "reject.yaml", lang: "yaml", content: "content"},
	})

	require.True(t, dr.active)

	filename := dr.reject()
	require.Equal(t, "reject.yaml", filename)

	// File should not exist
	_, err := os.ReadFile(filepath.Join(tmpDir, "reject.yaml"))
	require.Error(t, err)
}

func TestDiffReview_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	os.Chdir(tmpDir) //nolint:errcheck

	dr := newDiffReview()
	dr.startReview([]fileBlock{
		{name: "a.yaml", lang: "yaml", content: "a content"},
		{name: "b.yaml", lang: "yaml", content: "b content"},
	})

	require.True(t, dr.active)
	require.Equal(t, "a.yaml", dr.current().filename)

	// Accept first
	_, err := dr.accept()
	require.NoError(t, err)
	require.True(t, dr.advance())
	require.Equal(t, "b.yaml", dr.current().filename)

	// Reject second
	dr.reject()
	require.False(t, dr.advance())
	require.False(t, dr.active)

	summary := dr.summary()
	require.Contains(t, summary, "a.yaml")
	require.Contains(t, summary, "b.yaml")
	require.Contains(t, summary, "Saved")
	require.Contains(t, summary, "Rejected")
}

func TestDiffReview_Subdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	os.Chdir(tmpDir) //nolint:errcheck

	dr := newDiffReview()
	dr.startReview([]fileBlock{
		{name: "subdir/file.yaml", lang: "yaml", content: "nested content"},
	})

	require.True(t, dr.active)

	filename, err := dr.accept()
	require.NoError(t, err)
	require.Equal(t, "subdir/file.yaml", filename)

	data, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "file.yaml"))
	require.NoError(t, err)
	require.Equal(t, "nested content", string(data))
}

func TestDiffReview_View(t *testing.T) {
	dr := newDiffReview()
	dr.width = 80
	dr.height = 20
	dr.changes = []pendingChange{
		{
			filename:   "test.yaml",
			newContent: "new content",
			oldContent: "old content",
			diffLines: []diffLine{
				{kind: '@', text: "@@ test.yaml @@"},
				{kind: '-', text: "old content"},
				{kind: '+', text: "new content"},
			},
		},
	}
	dr.active = true

	view := dr.View()
	require.Contains(t, view, "test.yaml")
	require.Contains(t, view, "[y] Accept")
	require.Contains(t, view, "[n] Reject")
}

func TestSplitLines(t *testing.T) {
	require.Nil(t, splitLines(""))
	require.Equal(t, []string{"a", "b"}, splitLines("a\nb"))
	require.Equal(t, []string{"single"}, splitLines("single"))
}
