package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// chatMessage represents a rendered message in the chat.
type chatMessage struct {
	role    string // "user", "assistant", "system", "tool"
	content string
	files   []fileBlock // extracted code blocks with filenames
}

// fileBlock represents a code block that has a filename.
type fileBlock struct {
	name    string
	lang    string
	content string
}

// chatView renders the message history.
type chatView struct {
	messages []chatMessage
	scrollY  int
	width    int
	height   int
}

func newChatView() chatView {
	return chatView{}
}

func (cv *chatView) addMessage(role, content string) {
	msg := chatMessage{
		role:    role,
		content: content,
		files:   extractFileBlocks(content),
	}
	cv.messages = append(cv.messages, msg)
	cv.scrollToBottom()
}

// appendToLast appends text to the last message (for streaming).
func (cv *chatView) appendToLast(text string) {
	if len(cv.messages) == 0 {
		cv.addMessage("assistant", text)
		return
	}
	last := &cv.messages[len(cv.messages)-1]
	last.content += text
	last.files = extractFileBlocks(last.content)
	cv.scrollToBottom()
}

func (cv *chatView) scrollToBottom() {
	lines := cv.renderLines()
	if len(lines) > cv.height {
		cv.scrollY = len(lines) - cv.height
	}
}

func (cv *chatView) scrollUp(n int) {
	cv.scrollY -= n
	if cv.scrollY < 0 {
		cv.scrollY = 0
	}
}

func (cv *chatView) scrollDown(n int) {
	lines := cv.renderLines()
	cv.scrollY += n
	maxScroll := len(lines) - cv.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if cv.scrollY > maxScroll {
		cv.scrollY = maxScroll
	}
}

func (cv *chatView) renderLines() []string {
	var lines []string
	for _, msg := range cv.messages {
		lines = append(lines, cv.renderMessage(msg)...)
		lines = append(lines, "") // spacing between messages
	}
	return lines
}

func (cv *chatView) renderMessage(msg chatMessage) []string {
	var label string
	switch msg.role {
	case "user":
		label = userLabelStyle.Render("You")
	case "assistant":
		label = assistantLabelStyle.Render("AI")
	case "system":
		label = systemLabelStyle.Render("System")
	case "tool":
		label = toolLabelStyle.Render("Tool")
	default:
		label = msg.role
	}

	var lines []string
	lines = append(lines, label+":")

	// Render content, replacing file blocks with styled cards
	content := msg.content
	for _, fb := range msg.files {
		card := renderFileCard(fb, cv.width-4)
		// Replace the raw code block with the card
		rawBlock := "```" + fb.lang + " " + fb.name + "\n" + fb.content + "\n```"
		content = strings.Replace(content, rawBlock, card, 1)
	}

	// Word-wrap the content
	for _, line := range strings.Split(content, "\n") {
		if cv.width > 4 {
			wrapped := wrapText(line, cv.width-2)
			lines = append(lines, strings.Split(wrapped, "\n")...)
		} else {
			lines = append(lines, line)
		}
	}

	return lines
}

func renderFileCard(fb fileBlock, width int) string {
	if width < 20 {
		width = 20
	}

	header := fileNameStyle.Render("📄 " + fb.name)
	actions := fileActionStyle.Render("[s]ave  [a]pply")

	contentLines := strings.Split(fb.content, "\n")
	maxLines := 15
	if len(contentLines) > maxLines {
		contentLines = contentLines[:maxLines]
		contentLines = append(contentLines, lipgloss.NewStyle().Foreground(dimColor).Render("... (truncated)"))
	}
	body := strings.Join(contentLines, "\n")

	card := header + "\n" + body + "\n" + actions

	return fileCardBorder.Width(width).Render(card)
}

func (cv *chatView) View() string {
	lines := cv.renderLines()

	// Apply scroll
	if cv.scrollY > 0 && cv.scrollY < len(lines) {
		lines = lines[cv.scrollY:]
	}

	// Truncate to fit height
	if len(lines) > cv.height {
		lines = lines[:cv.height]
	}

	// Pad to fill height
	for len(lines) < cv.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// extractFileBlocks finds code blocks that have a filename annotation.
// Format: ```lang filename\ncontent\n```
func extractFileBlocks(content string) []fileBlock {
	var blocks []fileBlock
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "```") && len(line) > 3 {
			// Parse language and optional filename
			rest := strings.TrimPrefix(line, "```")
			parts := strings.Fields(rest)
			if len(parts) >= 2 {
				lang := parts[0]
				name := parts[1]

				// Collect content until closing ```
				var contentLines []string
				i++
				for i < len(lines) {
					if strings.TrimSpace(lines[i]) == "```" {
						break
					}
					contentLines = append(contentLines, lines[i])
					i++
				}

				blocks = append(blocks, fileBlock{
					name:    name,
					lang:    lang,
					content: strings.Join(contentLines, "\n"),
				})
			}
		}
	}

	return blocks
}

func wrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	for len(text) > width {
		// Find last space before width
		idx := strings.LastIndex(text[:width], " ")
		if idx <= 0 {
			idx = width
		}
		result.WriteString(text[:idx])
		result.WriteByte('\n')
		text = strings.TrimLeft(text[idx:], " ")
	}
	result.WriteString(text)
	return result.String()
}
