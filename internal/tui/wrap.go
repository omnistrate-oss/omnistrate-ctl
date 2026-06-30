// Package tui provides small, shared helpers for building bubbletea/lipgloss
// terminal UIs: line wrapping for scrollable viewers and clipboard access.
package tui

// VisualLine represents a single visual line in a wrapped code/log viewer.
// SourceNum is the 1-based source line number for the first visual line of a
// source line, or 0 for continuation (wrapped) lines.
type VisualLine struct {
	Text      string
	SourceNum int
}

// SoftWrapLine wraps a single line at character boundaries to fit within
// maxWidth runes. Returns a slice of strings, each no wider than maxWidth
// runes. A maxWidth <= 0 returns the line unchanged.
func SoftWrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}
	runes := []rune(line)
	if len(runes) <= maxWidth {
		return []string{line}
	}
	var result []string
	for len(runes) > maxWidth {
		result = append(result, string(runes[:maxWidth]))
		runes = runes[maxWidth:]
	}
	if len(runes) > 0 {
		result = append(result, string(runes))
	}
	return result
}

// ExpandLinesToVisual wraps each source line and builds a flat slice of visual
// lines. Wrapping operates on raw (un-styled) text so callers can apply syntax
// highlighting per visual line without splitting ANSI escape codes.
func ExpandLinesToVisual(sourceLines []string, maxWidth int) []VisualLine {
	var vlines []VisualLine
	for i, line := range sourceLines {
		wrapped := SoftWrapLine(line, maxWidth)
		for j, wl := range wrapped {
			num := 0
			if j == 0 {
				num = i + 1
			}
			vlines = append(vlines, VisualLine{Text: wl, SourceNum: num})
		}
	}
	return vlines
}
