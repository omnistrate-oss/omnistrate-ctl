package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fileEntry represents a file or directory in the panel.
type fileEntry struct {
	name  string
	path  string
	isDir bool
	depth int
}

// filePanel shows the local directory tree with file selection and preview.
type filePanel struct {
	entries   []fileEntry
	cursor    int
	scrollY   int
	width     int
	height    int
	visible   bool
	cwd       string
	preview   string // content of selected file
	expanded  map[string]bool
}

func newFilePanel() filePanel {
	cwd, _ := os.Getwd()
	fp := filePanel{
		cwd:      cwd,
		expanded: make(map[string]bool),
	}
	fp.refresh()
	return fp
}

// refresh reloads the directory listing.
func (fp *filePanel) refresh() {
	fp.entries = nil
	fp.scanDir(fp.cwd, 0)
	fp.updatePreview()
}

func (fp *filePanel) scanDir(dir string, depth int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fullPath := filepath.Join(dir, e.Name())
		fp.entries = append(fp.entries, fileEntry{
			name:  e.Name(),
			path:  fullPath,
			isDir: e.IsDir(),
			depth: depth,
		})
		// Recurse into expanded directories
		if e.IsDir() && fp.expanded[fullPath] {
			fp.scanDir(fullPath, depth+1)
		}
	}
}

func (fp *filePanel) moveUp() {
	if fp.cursor > 0 {
		fp.cursor--
		fp.adjustScroll()
		fp.updatePreview()
	}
}

func (fp *filePanel) moveDown() {
	if fp.cursor < len(fp.entries)-1 {
		fp.cursor++
		fp.adjustScroll()
		fp.updatePreview()
	}
}

func (fp *filePanel) adjustScroll() {
	// Keep cursor visible
	if fp.cursor < fp.scrollY {
		fp.scrollY = fp.cursor
	}
	previewLines := fp.previewHeight()
	listHeight := fp.height - previewLines - 2 // -2 for header + separator
	if listHeight < 3 {
		listHeight = 3
	}
	if fp.cursor >= fp.scrollY+listHeight {
		fp.scrollY = fp.cursor - listHeight + 1
	}
}

func (fp *filePanel) previewHeight() int {
	h := fp.height / 3
	if h < 5 {
		h = 5
	}
	if h > 15 {
		h = 15
	}
	return h
}

// toggleExpand expands or collapses a directory.
func (fp *filePanel) toggleExpand() {
	if fp.cursor >= len(fp.entries) {
		return
	}
	entry := fp.entries[fp.cursor]
	if !entry.isDir {
		return
	}
	if fp.expanded[entry.path] {
		delete(fp.expanded, entry.path)
	} else {
		fp.expanded[entry.path] = true
	}
	fp.refresh()
}

func (fp *filePanel) updatePreview() {
	if fp.cursor >= len(fp.entries) {
		fp.preview = ""
		return
	}
	entry := fp.entries[fp.cursor]
	if entry.isDir {
		fp.preview = fmt.Sprintf("📁 %s/", entry.name)
		return
	}

	data, err := os.ReadFile(entry.path)
	if err != nil {
		fp.preview = fmt.Sprintf("(cannot read: %v)", err)
		return
	}
	content := string(data)

	// Cap preview size
	maxChars := fp.width * fp.previewHeight()
	if maxChars > 2000 {
		maxChars = 2000
	}
	if len(content) > maxChars {
		content = content[:maxChars] + "\n... (truncated)"
	}
	fp.preview = content
}

// selectedEntry returns the currently selected entry, if any.
func (fp *filePanel) selectedEntry() *fileEntry {
	if fp.cursor >= len(fp.entries) {
		return nil
	}
	return &fp.entries[fp.cursor]
}

// selectedContent returns the content of the selected file for referencing in chat.
func (fp *filePanel) selectedContent() (name string, content string) {
	entry := fp.selectedEntry()
	if entry == nil || entry.isDir {
		return "", ""
	}
	data, err := os.ReadFile(entry.path)
	if err != nil {
		return entry.name, ""
	}
	return entry.name, string(data)
}

// View renders the file panel.
func (fp *filePanel) View() string {
	if !fp.visible || fp.width < 10 {
		return ""
	}

	var sb strings.Builder

	// Header
	header := filePanelHeaderStyle.Width(fp.width).Render(" 📂 Files")
	sb.WriteString(header)
	sb.WriteByte('\n')

	// File list
	previewH := fp.previewHeight()
	listHeight := fp.height - previewH - 2 // header + separator
	if listHeight < 3 {
		listHeight = 3
	}

	endIdx := fp.scrollY + listHeight
	if endIdx > len(fp.entries) {
		endIdx = len(fp.entries)
	}

	linesRendered := 0
	for i := fp.scrollY; i < endIdx; i++ {
		entry := fp.entries[i]
		line := fp.renderEntry(entry, i == fp.cursor)

		// Truncate to panel width
		if len([]rune(stripANSI(line))) > fp.width {
			runes := []rune(line)
			if len(runes) > fp.width {
				line = string(runes[:fp.width-1]) + "…"
			}
		}

		sb.WriteString(line)
		sb.WriteByte('\n')
		linesRendered++
	}

	// Pad remaining list space
	for linesRendered < listHeight {
		sb.WriteByte('\n')
		linesRendered++
	}

	// Separator
	sb.WriteString(filePanelSepStyle.Width(fp.width).Render(strings.Repeat("─", fp.width)))
	sb.WriteByte('\n')

	// Preview
	previewLines := strings.Split(fp.preview, "\n")
	for i := 0; i < previewH; i++ {
		if i < len(previewLines) {
			line := previewLines[i]
			// Truncate to width
			runes := []rune(line)
			if len(runes) > fp.width-1 {
				line = string(runes[:fp.width-2]) + "…"
			}
			sb.WriteString(filePanelPreviewStyle.Render(line))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func (fp *filePanel) renderEntry(entry fileEntry, selected bool) string {
	indent := strings.Repeat("  ", entry.depth)
	icon := "  "
	if entry.isDir {
		if fp.expanded[entry.path] {
			icon = "📂"
		} else {
			icon = "📁"
		}
	}

	name := entry.name
	line := fmt.Sprintf("%s%s %s", indent, icon, name)

	if selected {
		return filePanelSelectedStyle.Render(line)
	}
	if entry.isDir {
		return filePanelDirStyle.Render(line)
	}
	return filePanelFileStyle.Render(line)
}
