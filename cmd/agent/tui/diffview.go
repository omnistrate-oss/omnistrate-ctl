package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pendingChange represents a file change waiting for user approval.
type pendingChange struct {
	filename   string
	newContent string
	oldContent string // empty if new file
	diffLines  []diffLine
	accepted   *bool // nil = pending, true = accepted, false = rejected
}

type diffLine struct {
	kind byte // ' ' context, '+' added, '-' removed, '@' hunk header
	text string
}

// diffReview manages the file change review flow.
type diffReview struct {
	changes      []pendingChange
	currentIndex int
	scrollY      int
	width        int
	height       int
	active       bool
}

func newDiffReview() diffReview {
	return diffReview{}
}

// startReview populates the review with detected file blocks.
// It reads existing files from disk to compute diffs.
func (dr *diffReview) startReview(blocks []fileBlock) {
	dr.changes = nil
	dr.currentIndex = 0
	dr.scrollY = 0

	cwd, _ := os.Getwd()

	for _, block := range blocks {
		pc := pendingChange{
			filename:   block.name,
			newContent: block.content,
		}

		// Read existing file if present
		existingPath := filepath.Join(cwd, block.name)
		if data, err := os.ReadFile(existingPath); err == nil {
			pc.oldContent = string(data)
		}

		// Skip if content is identical
		if pc.oldContent == pc.newContent {
			continue
		}

		pc.diffLines = computeDiff(pc.oldContent, pc.newContent, block.name)
		dr.changes = append(dr.changes, pc)
	}

	dr.active = len(dr.changes) > 0
}

// current returns the current pending change, or nil if done.
func (dr *diffReview) current() *pendingChange {
	if dr.currentIndex >= len(dr.changes) {
		return nil
	}
	return &dr.changes[dr.currentIndex]
}

// accept marks the current change as accepted and writes it to disk.
func (dr *diffReview) accept() (string, error) {
	pc := dr.current()
	if pc == nil {
		return "", nil
	}

	cwd, _ := os.Getwd()
	filePath := filepath.Join(cwd, pc.filename)

	// Create parent directories if needed
	if dir := filepath.Dir(filePath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(filePath, []byte(pc.newContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", pc.filename, err)
	}

	accepted := true
	pc.accepted = &accepted
	return pc.filename, nil
}

// reject marks the current change as rejected.
func (dr *diffReview) reject() string {
	pc := dr.current()
	if pc == nil {
		return ""
	}
	rejected := false
	pc.accepted = &rejected
	return pc.filename
}

// advance moves to the next change. Returns false if no more changes.
func (dr *diffReview) advance() bool {
	dr.currentIndex++
	dr.scrollY = 0
	if dr.currentIndex >= len(dr.changes) {
		dr.active = false
		return false
	}
	return true
}

// summary returns a summary of all changes (accepted/rejected).
func (dr *diffReview) summary() string {
	var accepted, rejected []string
	for _, c := range dr.changes {
		if c.accepted != nil && *c.accepted {
			accepted = append(accepted, c.filename)
		} else {
			rejected = append(rejected, c.filename)
		}
	}
	var parts []string
	if len(accepted) > 0 {
		parts = append(parts, fmt.Sprintf("✅ Saved: %s", strings.Join(accepted, ", ")))
	}
	if len(rejected) > 0 {
		parts = append(parts, fmt.Sprintf("❌ Rejected: %s", strings.Join(rejected, ", ")))
	}
	if len(parts) == 0 {
		return "No file changes."
	}
	return strings.Join(parts, " • ")
}

func (dr *diffReview) scrollUp(n int) {
	dr.scrollY -= n
	if dr.scrollY < 0 {
		dr.scrollY = 0
	}
}

func (dr *diffReview) scrollDown(n int) {
	pc := dr.current()
	if pc == nil {
		return
	}
	dr.scrollY += n
	maxScroll := len(pc.diffLines) - dr.height + 4
	if maxScroll < 0 {
		maxScroll = 0
	}
	if dr.scrollY > maxScroll {
		dr.scrollY = maxScroll
	}
}

// View renders the diff review panel.
func (dr *diffReview) View() string {
	pc := dr.current()
	if pc == nil {
		return ""
	}

	var sb strings.Builder

	// Header
	changeType := "Modified"
	if pc.oldContent == "" {
		changeType = "New file"
	}
	header := fmt.Sprintf(" 📝 %s: %s  (%d/%d)",
		changeType, pc.filename, dr.currentIndex+1, len(dr.changes))
	sb.WriteString(diffHeaderStyle.Width(dr.width).Render(header))
	sb.WriteByte('\n')

	// Diff lines with scroll
	lines := pc.diffLines
	viewHeight := dr.height - 4 // reserve for header, footer, help
	if viewHeight < 3 {
		viewHeight = 3
	}

	startIdx := dr.scrollY
	if startIdx >= len(lines) {
		startIdx = 0
	}
	endIdx := startIdx + viewHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	if startIdx > 0 {
		sb.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more lines above", startIdx)))
		sb.WriteByte('\n')
	}

	for _, dl := range lines[startIdx:endIdx] {
		rendered := renderDiffLine(dl, dr.width)
		sb.WriteString(rendered)
		sb.WriteByte('\n')
	}

	remaining := len(lines) - endIdx
	if remaining > 0 {
		sb.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more lines below", remaining)))
		sb.WriteByte('\n')
	}

	// Pad to fill height
	currentLines := endIdx - startIdx + 2 // +header +footer
	for currentLines < viewHeight {
		sb.WriteByte('\n')
		currentLines++
	}

	// Action bar
	actions := "  [y] Accept & save  [n] Reject  [↑/↓] Scroll"
	sb.WriteString(diffActionStyle.Width(dr.width).Render(actions))

	return sb.String()
}

func renderDiffLine(dl diffLine, width int) string {
	line := dl.text
	if width > 2 && len(line) > width-2 {
		line = line[:width-2]
	}

	switch dl.kind {
	case '+':
		return diffAddStyle.Render("+ " + line)
	case '-':
		return diffRemoveStyle.Render("- " + line)
	case '@':
		return diffHunkStyle.Render(line)
	default:
		return "  " + line
	}
}

// computeDiff generates a simple unified-style diff between old and new content.
func computeDiff(oldContent, newContent, filename string) []diffLine {
	var lines []diffLine

	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	if oldContent == "" {
		// New file — show all lines as additions
		lines = append(lines, diffLine{kind: '@', text: fmt.Sprintf("@@ new file: %s @@", filename)})
		for _, l := range newLines {
			lines = append(lines, diffLine{kind: '+', text: l})
		}
		return lines
	}

	// Simple LCS-based diff
	lines = append(lines, diffLine{kind: '@', text: fmt.Sprintf("@@ %s @@", filename)})

	// Build edit script using LCS
	lcs := lcsMatrix(oldLines, newLines)
	diffOps := backtrack(lcs, oldLines, newLines, len(oldLines), len(newLines))

	// Convert to diff lines with context
	lines = append(lines, diffOps...)

	return lines
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// lcsMatrix builds the LCS length matrix.
func lcsMatrix(a, b []string) [][]int {
	m := len(a)
	n := len(b)
	matrix := make([][]int, m+1)
	for i := range matrix {
		matrix[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				matrix[i][j] = matrix[i-1][j-1] + 1
			} else if matrix[i-1][j] >= matrix[i][j-1] {
				matrix[i][j] = matrix[i-1][j]
			} else {
				matrix[i][j] = matrix[i][j-1]
			}
		}
	}
	return matrix
}

// backtrack produces diff lines from the LCS matrix.
func backtrack(matrix [][]int, a, b []string, i, j int) []diffLine {
	if i == 0 && j == 0 {
		return nil
	}
	if i > 0 && j > 0 && a[i-1] == b[j-1] {
		result := backtrack(matrix, a, b, i-1, j-1)
		return append(result, diffLine{kind: ' ', text: a[i-1]})
	}
	if j > 0 && (i == 0 || matrix[i][j-1] >= matrix[i-1][j]) {
		result := backtrack(matrix, a, b, i, j-1)
		return append(result, diffLine{kind: '+', text: b[j-1]})
	}
	result := backtrack(matrix, a, b, i-1, j)
	return append(result, diffLine{kind: '-', text: a[i-1]})
}
