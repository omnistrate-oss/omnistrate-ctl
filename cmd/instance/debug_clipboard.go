package instance

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/tui"
)

// clipboardResultMsg is sent after a clipboard copy attempt.
type clipboardResultMsg struct {
	err error
}

// copyToClipboardCmd returns a tea.Cmd that copies text to the system clipboard.
func copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		err := copyToClipboard(text)
		return clipboardResultMsg{err: err}
	}
}

func copyToClipboard(text string) error {
	return tui.CopyToClipboard(text)
}
