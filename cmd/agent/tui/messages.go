package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/provider"
)

// StreamChunkMsg wraps a streaming chunk from the provider.
type StreamChunkMsg struct {
	Chunk provider.StreamChunk
}

// MCPConnectedMsg indicates the MCP bridge connected successfully.
type MCPConnectedMsg struct {
	ToolCount int
}

// MCPErrorMsg indicates the MCP bridge failed to connect.
type MCPErrorMsg struct {
	Err error
}

// ToolResultMsg carries the result of a tool invocation.
type ToolResultMsg struct {
	ToolCallID string
	Name       string
	Result     string
	IsError    bool
}

// SendMsg triggers sending the current input.
type SendMsg struct{}

// ErrorMsg is a generic error message.
type ErrorMsg struct {
	Err error
}

// WaitForChunk returns a command that reads the next chunk from the stream.
func WaitForChunk(ch <-chan provider.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return StreamChunkMsg{Chunk: provider.StreamChunk{Done: true}}
		}
		return StreamChunkMsg{Chunk: chunk}
	}
}
