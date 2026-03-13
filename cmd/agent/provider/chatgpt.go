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
	openaiAPIURL = "https://api.openai.com/v1/chat/completions"
	openaiModel  = "gpt-4o"
	openaiEnvKey = "OPENAI_API_KEY"
)

type chatgptProvider struct{}

func init() {
	Register("chatgpt", func() Provider { return &chatgptProvider{} })
}

func (c *chatgptProvider) Name() string { return "ChatGPT" }

func (c *chatgptProvider) IsConfigured() (bool, string) {
	if os.Getenv(openaiEnvKey) == "" {
		return false, fmt.Sprintf("set %s environment variable", openaiEnvKey)
	}
	return true, ""
}

func (c *chatgptProvider) SendMessage(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	apiKey := os.Getenv(openaiEnvKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", openaiEnvKey)
	}

	body := openaiRequest{
		Model:  openaiModel,
		Stream: true,
	}

	for _, m := range messages {
		om := openaiMessage{Role: m.Role}
		if m.Content != "" {
			om.Content = m.Content
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				om.ToolCalls = append(om.ToolCalls, openaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openaiFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
		}
		if m.ToolResult != nil {
			om.Role = "tool"
			om.Content = m.ToolResult.Content
			om.ToolCallID = m.ToolResult.ToolCallID
		}
		body.Messages = append(body.Messages, om)
	}

	for _, t := range tools {
		body.Tools = append(body.Tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(b))
	}

	ch := make(chan StreamChunk, 64)
	go c.streamResponse(resp, ch)
	return ch, nil
}

func (c *chatgptProvider) streamResponse(resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	// Accumulate tool calls by index
	toolCalls := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			// Emit any accumulated tool calls
			for _, tc := range toolCalls {
				ch <- StreamChunk{ToolCall: tc}
			}
			ch <- StreamChunk{Done: true}
			return
		}

		var event openaiSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		// Text content
		if delta.Content != "" {
			ch <- StreamChunk{Text: delta.Content}
		}

		// Tool calls (streamed incrementally)
		for _, tc := range delta.ToolCalls {
			existing, ok := toolCalls[tc.Index]
			if !ok {
				existing = &ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
				}
				toolCalls[tc.Index] = existing
			}
			if tc.Function.Name != "" {
				existing.Name = tc.Function.Name
			}
			existing.Arguments += tc.Function.Arguments
		}

		if event.Choices[0].FinishReason == "stop" || event.Choices[0].FinishReason == "tool_calls" {
			for _, tc := range toolCalls {
				ch <- StreamChunk{ToolCall: tc}
			}
			ch <- StreamChunk{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
		return
	}
	ch <- StreamChunk{Done: true}
}

// OpenAI API types

type openaiRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Messages []openaiMessage `json:"messages"`
	Tools    []openaiTool    `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type openaiToolCall struct {
	Index    int            `json:"index"`
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type,omitempty"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openaiSSEEvent struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Delta        openaiDelta `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type openaiDelta struct {
	Content   string           `json:"content,omitempty"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}
