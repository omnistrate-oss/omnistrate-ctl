package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/mcpbridge"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/provider"
)

// Model is the root Bubble Tea model for the agent TUI.
type Model struct {
	provider provider.Provider
	bridge   *mcpbridge.Bridge
	ctx      context.Context //nolint:containedctx // Bubble Tea models must store context for async commands
	cancel   context.CancelFunc

	// Conversation state
	messages []provider.Message
	stream   <-chan provider.StreamChunk

	// UI components
	chat      chatView
	input     inputArea
	statusBar statusBar
	diff      diffReview
	files     filePanel

	// Window dimensions
	width  int
	height int

	// State flags
	streaming  bool
	quitting   bool
	focusFiles bool // true when file panel has focus
}

// New creates a new TUI model.
func New(p provider.Provider, bridge *mcpbridge.Bridge, systemPrompt string) Model {
	ctx, cancel := context.WithCancel(context.Background())

	cwd, _ := os.Getwd()

	m := Model{
		provider:  p,
		bridge:    bridge,
		ctx:       ctx,
		cancel:    cancel,
		chat:      newChatView(),
		input:     newInputArea(),
		statusBar: newStatusBar(p.Name(), cwd),
		diff:      newDiffReview(),
		files:     newFilePanel(),
	}

	// Add system prompt
	if systemPrompt != "" {
		m.messages = append(m.messages, provider.Message{
			Role:    provider.RoleSystem,
			Content: systemPrompt,
		})
		m.chat.addMessage("system", "Connected. I have access to Omnistrate tools. How can I help?")
	}

	return m
}

// filePanelWidth returns the width of the file panel when visible.
func (m *Model) filePanelWidth() int {
	w := m.width / 4
	if w < 25 {
		w = 25
	}
	if w > 45 {
		w = 45
	}
	return w
}

func (m *Model) recalcLayout() {
	chatWidth := m.width
	if m.files.visible {
		fpw := m.filePanelWidth()
		m.files.width = fpw
		m.files.height = m.height - 6
		chatWidth = m.width - fpw - 1 // -1 for vertical separator
	}
	m.chat.width = chatWidth
	m.chat.height = m.height - 6
	m.input.width = m.width
	m.statusBar.width = m.width
	m.diff.width = m.width
	m.diff.height = m.height - 4
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		connectMCP(m.ctx, m.bridge),
		tea.ClearScreen,
	)
}

func connectMCP(ctx context.Context, bridge *mcpbridge.Bridge) tea.Cmd {
	return func() tea.Msg {
		if err := bridge.Connect(ctx); err != nil {
			return MCPErrorMsg{Err: err}
		}
		return MCPConnectedMsg{ToolCount: len(bridge.Tools())}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case MCPConnectedMsg:
		m.statusBar.mcpConnected = true
		m.statusBar.mcpToolCount = msg.ToolCount
		return m, nil

	case MCPErrorMsg:
		m.chat.addMessage("system", fmt.Sprintf("⚠️  MCP connection failed: %v", msg.Err))
		return m, nil

	case StreamChunkMsg:
		return m.handleStreamChunk(msg.Chunk)

	case ToolResultMsg:
		return m.handleToolResult(msg)

	case streamStartMsg:
		return m.handleStreamStart(msg)

	case ErrorMsg:
		m.streaming = false
		m.statusBar.streaming = false
		m.chat.addMessage("system", fmt.Sprintf("❌ Error: %v", msg.Err))
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Diff review mode — handle y/n keys
	if m.diff.active {
		return m.handleDiffKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		m.cancel()
		return m, tea.Quit

	case "tab":
		if !m.streaming {
			if m.focusFiles && m.files.visible {
				// Switch focus back to chat input
				m.focusFiles = false
			} else if m.files.visible {
				// Switch focus to file panel
				m.focusFiles = true
			} else {
				// Open file panel and focus it
				m.files.visible = true
				m.files.refresh()
				m.focusFiles = true
				m.recalcLayout()
			}
		}
		return m, nil

	case "ctrl+b":
		// Toggle file panel visibility
		if !m.streaming {
			m.files.visible = !m.files.visible
			if !m.files.visible {
				m.focusFiles = false
			} else {
				m.files.refresh()
			}
			m.recalcLayout()
		}
		return m, nil

	case "ctrl+enter", "alt+enter":
		if !m.streaming && strings.TrimSpace(m.input.Value()) != "" {
			return m.sendMessage()
		}
		return m, nil

	case "enter":
		if m.focusFiles {
			// Toggle directory expand or attach file reference
			entry := m.files.selectedEntry()
			if entry != nil && entry.isDir {
				m.files.toggleExpand()
			} else if entry != nil {
				// Insert @filename reference into input
				name, content := m.files.selectedContent()
				if name != "" && content != "" {
					ref := fmt.Sprintf("@%s", name)
					for _, r := range ref {
						m.input.InsertChar(r)
					}
					m.focusFiles = false
				}
			}
			return m, nil
		}
		if !m.streaming {
			m.input.InsertNewline()
		}
		return m, nil

	case "backspace":
		if !m.streaming && !m.focusFiles {
			m.input.Backspace()
		}
		return m, nil

	case "left":
		if !m.focusFiles {
			m.input.MoveLeft()
		}
		return m, nil

	case "right":
		if !m.focusFiles {
			m.input.MoveRight()
		}
		return m, nil

	case "up":
		if m.focusFiles {
			m.files.moveUp()
		} else if m.streaming {
			m.chat.scrollUp(1)
		} else {
			m.input.MoveUp()
		}
		return m, nil

	case "down":
		if m.focusFiles {
			m.files.moveDown()
		} else if m.streaming {
			m.chat.scrollDown(1)
		} else {
			m.input.MoveDown()
		}
		return m, nil

	case "pgup":
		if m.focusFiles {
			m.files.moveUp()
			m.files.moveUp()
			m.files.moveUp()
			m.files.moveUp()
			m.files.moveUp()
		} else {
			m.chat.scrollUp(m.chat.height / 2)
		}
		return m, nil

	case "pgdown":
		if m.focusFiles {
			m.files.moveDown()
			m.files.moveDown()
			m.files.moveDown()
			m.files.moveDown()
			m.files.moveDown()
		} else {
			m.chat.scrollDown(m.chat.height / 2)
		}
		return m, nil

	default:
		if m.focusFiles {
			return m, nil
		}
		if !m.streaming && len(msg.String()) == 1 {
			for _, r := range msg.String() {
				m.input.InsertChar(r)
			}
		} else if !m.streaming && msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				m.input.InsertChar(r)
			}
		}
		return m, nil
	}
}

func (m Model) sendMessage() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	m.input.Reset()

	// Expand @filename references to include file content
	text = m.expandFileRefs(text)

	// Add to conversation
	m.messages = append(m.messages, provider.Message{
		Role:    provider.RoleUser,
		Content: text,
	})
	m.chat.addMessage("user", text)

	// Start streaming
	m.streaming = true
	m.statusBar.streaming = true
	m.chat.addMessage("assistant", "")

	return m, m.startStream()
}

// expandFileRefs replaces @filename tokens with the file's content.
func (m Model) expandFileRefs(text string) string {
	words := strings.Fields(text)
	for _, word := range words {
		if !strings.HasPrefix(word, "@") || len(word) < 2 {
			continue
		}
		filename := strings.TrimPrefix(word, "@")
		cwd, _ := os.Getwd()
		data, err := os.ReadFile(filepath.Join(cwd, filename))
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > 3000 {
			content = content[:3000] + "\n... (truncated)"
		}
		ref := fmt.Sprintf("\n\n--- %s ---\n```\n%s\n```\n", filename, content)
		text = strings.Replace(text, word, word+ref, 1)
	}
	return text
}

func (m Model) startStream() tea.Cmd {
	return func() tea.Msg {
		tools := m.bridge.Tools()
		ch, err := m.provider.SendMessage(m.ctx, m.messages, tools)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		// Read first chunk
		chunk, ok := <-ch
		if !ok {
			return StreamChunkMsg{Chunk: provider.StreamChunk{Done: true}}
		}
		// Store channel for subsequent reads via a closure
		// We return the first chunk and set up for reading more
		return streamStartMsg{firstChunk: chunk, ch: ch}
	}
}

// streamStartMsg carries the channel and first chunk.
type streamStartMsg struct {
	firstChunk provider.StreamChunk
	ch         <-chan provider.StreamChunk
}

func (m Model) handleStreamStart(msg streamStartMsg) (tea.Model, tea.Cmd) {
	m.stream = msg.ch

	// Process the first chunk
	newM, cmd := m.handleStreamChunk(msg.firstChunk)
	return newM, cmd
}

func (m Model) handleStreamChunk(chunk provider.StreamChunk) (tea.Model, tea.Cmd) {
	if chunk.Error != nil {
		m.streaming = false
		m.statusBar.streaming = false
		m.chat.addMessage("system", fmt.Sprintf("❌ Stream error: %v", chunk.Error))
		return m, nil
	}

	if chunk.Text != "" {
		m.chat.appendToLast(chunk.Text)
	}

	if chunk.ToolCall != nil {
		// Record assistant's tool call in conversation
		lastMsg := &m.messages[len(m.messages)-1]
		if lastMsg.Role != provider.RoleAssistant {
			m.messages = append(m.messages, provider.Message{
				Role: provider.RoleAssistant,
			})
		}
		// Add the current streamed text as content
		if len(m.chat.messages) > 0 {
			lastChat := m.chat.messages[len(m.chat.messages)-1]
			m.messages[len(m.messages)-1].Content = lastChat.content
		}
		m.messages[len(m.messages)-1].ToolCalls = append(
			m.messages[len(m.messages)-1].ToolCalls,
			*chunk.ToolCall,
		)

		tc := chunk.ToolCall
		m.chat.addMessage("tool", fmt.Sprintf("🔧 Calling: %s", tc.Name))

		// Execute tool call
		return m, m.callTool(tc.ID, tc.Name, tc.Arguments)
	}

	if chunk.Done {
		m.streaming = false
		m.statusBar.streaming = false

		// Record assistant message in conversation
		if len(m.chat.messages) > 0 {
			lastChat := m.chat.messages[len(m.chat.messages)-1]
			if lastChat.role == "assistant" {
				m.messages = append(m.messages, provider.Message{
					Role:    provider.RoleAssistant,
					Content: lastChat.content,
				})
			}
		}

		// Check for file changes in the response and start diff review
		m.checkForFileChanges()

		return m, nil
	}

	// Continue reading stream
	if m.stream != nil {
		return m, WaitForChunk(m.stream)
	}
	return m, nil
}

func (m Model) callTool(toolCallID, name, arguments string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.bridge.CallTool(m.ctx, name, arguments)
		if err != nil {
			return ToolResultMsg{
				ToolCallID: toolCallID,
				Name:       name,
				Result:     err.Error(),
				IsError:    true,
			}
		}
		return ToolResultMsg{
			ToolCallID: toolCallID,
			Name:       name,
			Result:     result,
		}
	}
}

func (m Model) handleToolResult(msg ToolResultMsg) (tea.Model, tea.Cmd) {
	// Display result in chat
	resultPreview := msg.Result
	if len(resultPreview) > 500 {
		resultPreview = resultPreview[:500] + "... (truncated)"
	}

	if msg.IsError {
		m.chat.addMessage("tool", fmt.Sprintf("❌ %s failed: %s", msg.Name, resultPreview))
	} else {
		m.chat.addMessage("tool", fmt.Sprintf("✅ %s: %s", msg.Name, resultPreview))
	}

	// Add tool result to conversation
	m.messages = append(m.messages, provider.Message{
		Role: provider.RoleTool,
		ToolResult: &provider.ToolResult{
			ToolCallID: msg.ToolCallID,
			Content:    msg.Result,
			IsError:    msg.IsError,
		},
	})

	// Continue conversation — let the AI process the tool result
	m.streaming = true
	m.statusBar.streaming = true
	m.chat.addMessage("assistant", "")

	return m, m.startStream()
}

// checkForFileChanges scans the last assistant message for file blocks
// and starts the diff review flow if changes are detected.
func (m *Model) checkForFileChanges() {
	if len(m.chat.messages) == 0 {
		return
	}
	// Find the last assistant message
	var lastAssistantContent string
	for i := len(m.chat.messages) - 1; i >= 0; i-- {
		if m.chat.messages[i].role == "assistant" {
			lastAssistantContent = m.chat.messages[i].content
			break
		}
	}
	if lastAssistantContent == "" {
		return
	}

	blocks := extractFileBlocks(lastAssistantContent)
	if len(blocks) == 0 {
		return
	}

	m.diff.startReview(blocks)
	if m.diff.active {
		m.diff.width = m.width
		m.diff.height = m.height - 4
	}
}

func (m Model) handleDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		m.cancel()
		return m, tea.Quit

	case "y", "Y":
		filename, err := m.diff.accept()
		if err != nil {
			m.chat.addMessage("system", fmt.Sprintf("❌ Failed to save %s: %v", filename, err))
		} else if filename != "" {
			m.chat.addMessage("system", fmt.Sprintf("✅ Saved: %s", filename))
		}
		if !m.diff.advance() {
			m.chat.addMessage("system", m.diff.summary())
			m.files.refresh()
		}
		return m, nil

	case "n", "N":
		filename := m.diff.reject()
		if filename != "" {
			m.chat.addMessage("system", fmt.Sprintf("⏭  Rejected: %s", filename))
		}
		if !m.diff.advance() {
			m.chat.addMessage("system", m.diff.summary())
			m.files.refresh()
		}
		return m, nil

	case "up", "k":
		m.diff.scrollUp(1)
		return m, nil

	case "down", "j":
		m.diff.scrollDown(1)
		return m, nil

	case "pgup":
		m.diff.scrollUp(m.diff.height / 2)
		return m, nil

	case "pgdown":
		m.diff.scrollDown(m.diff.height / 2)
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 {
		return "Initializing..."
	}

	var sections []string

	// Status bar
	sections = append(sections, m.statusBar.View())

	if m.diff.active {
		// Diff review mode
		sections = append(sections, m.diff.View())
	} else {
		// Main content area: file panel (optional) + chat
		chatContent := m.chat.View()

		if m.files.visible {
			panelContent := m.files.View()
			// Render side by side
			combined := renderSideBySide(panelContent, chatContent, m.filePanelWidth(), m.chat.width, m.height-6)
			sections = append(sections, combined)
		} else {
			sections = append(sections, chatContent)
		}

		// Separator
		sections = append(sections, strings.Repeat("─", m.width))

		// Input area
		sections = append(sections, m.input.View())

		// Help
		var helpText string
		if m.focusFiles {
			helpText = "  Tab: back to chat • Enter: expand/attach • Ctrl+B: hide • Ctrl+C: quit"
		} else {
			helpText = "  Ctrl+Enter: send • Tab: files • @file: attach • Ctrl+C: quit"
		}
		sections = append(sections, helpStyle.Render(helpText))
	}

	return strings.Join(sections, "\n")
}

// renderSideBySide renders two panels side by side with a vertical separator.
func renderSideBySide(left, right string, leftWidth, rightWidth, height int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Pad to same height
	for len(leftLines) < height {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < height {
		rightLines = append(rightLines, "")
	}

	var sb strings.Builder
	maxLines := height
	if len(leftLines) > maxLines {
		maxLines = len(leftLines)
	}

	sep := filePanelSepStyle.Render("│")

	for i := 0; i < height && i < maxLines; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}

		// Pad left panel to fixed width
		lVisible := len([]rune(stripANSI(l)))
		if lVisible < leftWidth {
			l += strings.Repeat(" ", leftWidth-lVisible)
		}

		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(l)
		sb.WriteString(sep)
		sb.WriteString(r)
	}

	return sb.String()
}

// Ensure Model satisfies tea.Model interface.
var _ tea.Model = Model{}
