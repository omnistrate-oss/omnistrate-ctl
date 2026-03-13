package provider

import (
	"context"
	"fmt"
	"sync"
)

// Provider is the interface that all AI agent providers must implement.
type Provider interface {
	// Name returns the display name of the provider (e.g., "Claude", "ChatGPT", "Copilot").
	Name() string
	// SendMessage sends a conversation and returns a channel of streaming chunks.
	SendMessage(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error)
	// IsConfigured checks if the provider has the required configuration (API keys, etc.).
	// Returns (configured bool, hint string) where hint explains what's missing.
	IsConfigured() (bool, string)
}

// Role constants for messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message represents a single message in the conversation.
type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolCall represents an AI-requested tool invocation.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolResult represents the result of a tool invocation fed back to the AI.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// Tool describes an MCP tool available for the AI to call.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"` // JSON Schema object
}

// StreamChunk is a piece of a streaming response from the provider.
type StreamChunk struct {
	Text     string    // Text delta
	ToolCall *ToolCall // Tool call (may arrive in parts)
	Done     bool      // True when the response is complete
	Error    error     // Non-nil if an error occurred
}

// registry holds registered provider constructors.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]func() Provider)
)

// Register adds a provider constructor to the registry.
func Register(name string, constructor func() Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = constructor
}

// Get returns a provider instance by name.
func Get(name string) (Provider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	constructor, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q; available: %v", name, Names())
	}
	return constructor(), nil
}

// Names returns all registered provider names.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
