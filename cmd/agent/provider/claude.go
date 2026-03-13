package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	claudeAPIURL     = "https://api.anthropic.com/v1/messages"
	claudeModel      = "claude-sonnet-4-20250514"
	claudeAPIVersion = "2023-06-01"
	claudeEnvKey     = "ANTHROPIC_API_KEY"
)

type claudeProvider struct{}

func init() {
	Register("claude", func() Provider { return &claudeProvider{} })
}

func (c *claudeProvider) Name() string { return "Claude" }

func (c *claudeProvider) IsConfigured() (bool, string) {
	if os.Getenv(claudeEnvKey) == "" {
		return false, fmt.Sprintf("set %s environment variable", claudeEnvKey)
	}
	return true, ""
}

func (c *claudeProvider) SendMessage(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	apiKey := os.Getenv(claudeEnvKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", claudeEnvKey)
	}

	// Build request body
	body := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 8192,
		Stream:    true,
	}

	for _, m := range messages {
		switch m.Role {
		case RoleSystem:
			body.System = m.Content
		case RoleUser:
			body.Messages = append(body.Messages, claudeMessage{Role: "user", Content: m.Content})
		case RoleAssistant:
			cm := claudeMessage{Role: "assistant"}
			if m.Content != "" {
				cm.Content = m.Content
			}
			body.Messages = append(body.Messages, cm)
		case RoleTool:
			if m.ToolResult != nil {
				body.Messages = append(body.Messages, claudeMessage{
					Role: "user",
					ToolResult: &claudeToolResult{
						Type:      "tool_result",
						ToolUseID: m.ToolResult.ToolCallID,
						Content:   m.ToolResult.Content,
						IsError:   m.ToolResult.IsError,
					},
				})
			}
		}
	}

	// Convert tools
	for _, t := range tools {
		body.Tools = append(body.Tools, claudeTool(t))
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(b))
	}

	ch := make(chan StreamChunk, 64)
	go c.streamResponse(resp, ch)
	return ch, nil
}

func (c *claudeProvider) streamResponse(resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var currentToolCall *ToolCall

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event claudeSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				currentToolCall = &ToolCall{
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}
			}
		case "content_block_delta":
			if event.Delta != nil {
				switch event.Delta.Type {
				case "text_delta":
					ch <- StreamChunk{Text: event.Delta.Text}
				case "input_json_delta":
					if currentToolCall != nil {
						currentToolCall.Arguments += event.Delta.PartialJSON
					}
				}
			}
		case "content_block_stop":
			if currentToolCall != nil {
				ch <- StreamChunk{ToolCall: currentToolCall}
				currentToolCall = nil
			}
		case "message_stop":
			ch <- StreamChunk{Done: true}
			return
		case "error":
			ch <- StreamChunk{Error: fmt.Errorf("claude stream error: %s", data)}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
		return
	}
	ch <- StreamChunk{Done: true}
}

// Claude API types

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
	Tools     []claudeTool    `json:"tools,omitempty"`
}

type claudeMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolResult *claudeToolResult `json:"tool_result,omitempty"`
}

type claudeToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type claudeTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type claudeSSEEvent struct {
	Type         string              `json:"type"`
	Delta        *claudeDelta        `json:"delta,omitempty"`
	ContentBlock *claudeContentBlock `json:"content_block,omitempty"`
}

type claudeDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
