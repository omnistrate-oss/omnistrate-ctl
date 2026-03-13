package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilePanel_Refresh(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.yaml"), []byte("content1"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))

	fp := newFilePanel()
	require.GreaterOrEqual(t, len(fp.entries), 3)

	// Check entries include our files
	names := make([]string, 0, len(fp.entries))
	for _, e := range fp.entries {
		names = append(names, e.name)
	}
	require.Contains(t, names, "file1.yaml")
	require.Contains(t, names, "file2.txt")
	require.Contains(t, names, "subdir")
}

func TestFilePanel_Navigation(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.yaml"), []byte("aaa"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.yaml"), []byte("bbb"), 0600))

	fp := newFilePanel()
	fp.height = 30
	fp.width = 40

	require.Equal(t, 0, fp.cursor)
	fp.moveDown()
	require.Equal(t, 1, fp.cursor)
	fp.moveUp()
	require.Equal(t, 0, fp.cursor)
	// Don't go negative
	fp.moveUp()
	require.Equal(t, 0, fp.cursor)
}

func TestFilePanel_ExpandDir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "mydir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "mydir", "inner.txt"), []byte("inner"), 0600))

	fp := newFilePanel()
	// Find the directory entry
	dirIdx := -1
	for i, e := range fp.entries {
		if e.name == "mydir" {
			dirIdx = i
			break
		}
	}
	require.NotEqual(t, -1, dirIdx)
	initialCount := len(fp.entries)

	fp.cursor = dirIdx
	fp.toggleExpand()

	// Should have more entries now (inner.txt added)
	require.Greater(t, len(fp.entries), initialCount)

	// Collapse
	fp.cursor = dirIdx
	fp.toggleExpand()
	require.Equal(t, initialCount, len(fp.entries))
}

func TestFilePanel_Preview(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	content := "services:\n  web:\n    image: nginx\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte(content), 0600))

	fp := newFilePanel()
	fp.width = 60
	fp.height = 20

	// Find spec.yaml
	for i, e := range fp.entries {
		if e.name == "spec.yaml" {
			fp.cursor = i
			fp.updatePreview()
			break
		}
	}

	require.Contains(t, fp.preview, "nginx")
}

func TestFilePanel_SelectedContent(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte("key: value"), 0600))

	fp := newFilePanel()
	// Select the file
	for i, e := range fp.entries {
		if e.name == "test.yaml" {
			fp.cursor = i
			break
		}
	}

	name, content := fp.selectedContent()
	require.Equal(t, "test.yaml", name)
	require.Equal(t, "key: value", content)
}

func TestFilePanel_View(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("world"), 0600))

	fp := newFilePanel()
	fp.visible = true
	fp.width = 40
	fp.height = 20

	view := fp.View()
	require.Contains(t, view, "Files")
	require.Contains(t, view, "hello.txt")
}
