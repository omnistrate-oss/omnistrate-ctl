package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("39")  // Blue
	secondaryColor = lipgloss.Color("205") // Pink
	accentColor    = lipgloss.Color("42")  // Green
	errorColor     = lipgloss.Color("196") // Red
	dimColor       = lipgloss.Color("240") // Gray
	toolColor      = lipgloss.Color("214") // Orange

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	mcpConnectedStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	mcpDisconnectedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	// Chat messages
	userLabelStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	systemLabelStyle = lipgloss.NewStyle().
				Foreground(dimColor).
				Italic(true)

	toolLabelStyle = lipgloss.NewStyle().
			Foreground(toolColor).
			Bold(true)

	// File card
	fileCardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(0, 1)

	fileNameStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	fileActionStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Input area
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	// General
	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Diff review
	diffHeaderStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Bold(true).
			Padding(0, 1)

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")) // green

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // red

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // blue
			Bold(true)

	diffActionStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("220")).
			Padding(0, 1)

	// File panel
	filePanelHeaderStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252")).
				Bold(true)

	filePanelSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("24")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	filePanelDirStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")) // blue

	filePanelFileStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	filePanelPreviewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	filePanelSepStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)
