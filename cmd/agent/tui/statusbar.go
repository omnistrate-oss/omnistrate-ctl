package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// statusBar shows provider info and MCP connection status.
type statusBar struct {
	providerName string
	mcpConnected bool
	mcpToolCount int
	cwd          string
	width        int
	streaming    bool
}

func newStatusBar(providerName, cwd string) statusBar {
	return statusBar{
		providerName: providerName,
		cwd:          cwd,
	}
}

func (sb *statusBar) View() string {
	left := fmt.Sprintf(" 🤖 omnistrate-ctl agent %s", sb.providerName)

	var mcpStatus string
	if sb.mcpConnected {
		mcpStatus = mcpConnectedStyle.Render(fmt.Sprintf("● MCP: %d tools", sb.mcpToolCount))
	} else {
		mcpStatus = mcpDisconnectedStyle.Render("○ MCP: connecting...")
	}

	var streamIndicator string
	if sb.streaming {
		streamIndicator = " " + lipgloss.NewStyle().Foreground(secondaryColor).Render("⟳")
	}

	right := mcpStatus + streamIndicator + " "

	// Calculate padding
	padding := sb.width - lipglossWidth(left) - lipglossWidth(right)
	if padding < 1 {
		padding = 1
	}

	full := left + strings.Repeat(" ", padding) + right
	return statusBarStyle.Width(sb.width).Render(full)
}

func lipglossWidth(s string) int {
	// Simple approximation: count visible characters
	// Strip ANSI sequences for length calculation
	clean := stripANSI(s)
	return len([]rune(clean))
}

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
