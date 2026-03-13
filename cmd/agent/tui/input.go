package tui

import (
	"strings"
)

// inputArea is a simple multiline text input.
type inputArea struct {
	lines     []string
	cursorRow int
	cursorCol int
	width     int
	height    int // max visible lines
}

func newInputArea() inputArea {
	return inputArea{
		lines:  []string{""},
		height: 3,
	}
}

func (ia *inputArea) Value() string {
	return strings.Join(ia.lines, "\n")
}

func (ia *inputArea) Reset() {
	ia.lines = []string{""}
	ia.cursorRow = 0
	ia.cursorCol = 0
}

func (ia *inputArea) InsertChar(ch rune) {
	line := ia.lines[ia.cursorRow]
	ia.lines[ia.cursorRow] = line[:ia.cursorCol] + string(ch) + line[ia.cursorCol:]
	ia.cursorCol++
}

func (ia *inputArea) InsertNewline() {
	line := ia.lines[ia.cursorRow]
	before := line[:ia.cursorCol]
	after := line[ia.cursorCol:]
	ia.lines[ia.cursorRow] = before
	// Insert after current row
	newLines := make([]string, 0, len(ia.lines)+1)
	newLines = append(newLines, ia.lines[:ia.cursorRow+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, ia.lines[ia.cursorRow+1:]...)
	ia.lines = newLines
	ia.cursorRow++
	ia.cursorCol = 0
}

func (ia *inputArea) Backspace() {
	if ia.cursorCol > 0 {
		line := ia.lines[ia.cursorRow]
		ia.lines[ia.cursorRow] = line[:ia.cursorCol-1] + line[ia.cursorCol:]
		ia.cursorCol--
	} else if ia.cursorRow > 0 {
		// Merge with previous line
		prevLine := ia.lines[ia.cursorRow-1]
		curLine := ia.lines[ia.cursorRow]
		ia.lines[ia.cursorRow-1] = prevLine + curLine
		ia.lines = append(ia.lines[:ia.cursorRow], ia.lines[ia.cursorRow+1:]...)
		ia.cursorRow--
		ia.cursorCol = len(prevLine)
	}
}

func (ia *inputArea) MoveLeft() {
	if ia.cursorCol > 0 {
		ia.cursorCol--
	} else if ia.cursorRow > 0 {
		ia.cursorRow--
		ia.cursorCol = len(ia.lines[ia.cursorRow])
	}
}

func (ia *inputArea) MoveRight() {
	if ia.cursorCol < len(ia.lines[ia.cursorRow]) {
		ia.cursorCol++
	} else if ia.cursorRow < len(ia.lines)-1 {
		ia.cursorRow++
		ia.cursorCol = 0
	}
}

func (ia *inputArea) MoveUp() {
	if ia.cursorRow > 0 {
		ia.cursorRow--
		if ia.cursorCol > len(ia.lines[ia.cursorRow]) {
			ia.cursorCol = len(ia.lines[ia.cursorRow])
		}
	}
}

func (ia *inputArea) MoveDown() {
	if ia.cursorRow < len(ia.lines)-1 {
		ia.cursorRow++
		if ia.cursorCol > len(ia.lines[ia.cursorRow]) {
			ia.cursorCol = len(ia.lines[ia.cursorRow])
		}
	}
}

func (ia *inputArea) View() string {
	prompt := inputPromptStyle.Render("> ")
	promptWidth := 2

	var sb strings.Builder
	visibleLines := ia.lines
	startRow := 0
	if len(visibleLines) > ia.height {
		startRow = len(visibleLines) - ia.height
		visibleLines = visibleLines[startRow:]
	}

	for i, line := range visibleLines {
		actualRow := startRow + i
		if i > 0 {
			sb.WriteByte('\n')
		}

		if i == 0 {
			sb.WriteString(prompt)
		} else {
			sb.WriteString(strings.Repeat(" ", promptWidth))
		}

		// Truncate line to width
		displayLine := line
		maxWidth := ia.width - promptWidth
		if maxWidth > 0 && len(displayLine) > maxWidth {
			displayLine = displayLine[:maxWidth]
		}

		// Insert cursor
		if actualRow == ia.cursorRow {
			col := ia.cursorCol
			if col > len(displayLine) {
				col = len(displayLine)
			}
			before := displayLine[:col]
			after := displayLine[col:]
			sb.WriteString(before)
			sb.WriteString("█") // block cursor
			sb.WriteString(after)
		} else {
			sb.WriteString(displayLine)
		}
	}

	return sb.String()
}
