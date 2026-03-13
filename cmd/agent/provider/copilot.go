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
	"os/exec"
	"strings"
)

const (
	// GitHub Models API uses the same OpenAI-compatible format
	copilotAPIURL      = "https://models.inference.ai.azure.com/chat/completions"
	copilotModel       = "gpt-4o-mini"
	copilotEnvKey      = "GITHUB_TOKEN"
	copilotMaxMsgChars = 4000  // char budget for messages (system + conversation)
	copilotMaxTools    = 5     // max tools to send (tool schemas are token-heavy)
	copilotMaxPayload  = 20000 // max total JSON payload bytes (~5K tokens)
)

type copilotProvider struct{}

func init() {
	Register("copilot", func() Provider { return &copilotProvider{} })
}

func (c *copilotProvider) Name() string { return "Copilot" }

func (c *copilotProvider) IsConfigured() (bool, string) {
	if os.Getenv(copilotEnvKey) != "" {
		return true, ""
	}
	// Fallback: try `gh auth token`
	if token := ghAuthToken(); token != "" {
		return true, ""
	}
	return false, fmt.Sprintf("set %s environment variable or authenticate with `gh auth login`", copilotEnvKey)
}

func (c *copilotProvider) SendMessage(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	token := os.Getenv(copilotEnvKey)
	if token == "" {
		token = ghAuthToken()
	}
	if token == "" {
		return nil, fmt.Errorf("%s not set and `gh auth token` failed", copilotEnvKey)
	}

	// Copilot/GitHub Models uses OpenAI-compatible format
	body := openaiRequest{
		Model:  copilotModel,
		Stream: true,
	}

	// Truncate conversation to fit within GitHub Models token limits.
	// Keep system message + most recent messages that fit the char budget.
	truncated := truncateMessages(messages, copilotMaxMsgChars)

	for _, m := range truncated {
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

	// Limit tools to stay under token budget — tool schemas are very token-heavy
	limitedTools := limitTools(tools, copilotMaxTools)
	for _, t := range limitedTools {
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

	// If payload is still too large, retry without tools
	if len(jsonBody) > copilotMaxPayload {
		body.Tools = nil
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, copilotAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github models API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github models API error (status %d): %s", resp.StatusCode, string(b))
	}

	ch := make(chan StreamChunk, 64)
	go c.streamResponse(resp, ch)
	return ch, nil
}

// streamResponse parses OpenAI-compatible SSE stream (same as ChatGPT)
func (c *copilotProvider) streamResponse(resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	toolCalls := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
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

		if delta.Content != "" {
			ch <- StreamChunk{Text: delta.Content}
		}

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

// truncateMessages keeps the system message and the most recent messages
// that fit within maxChars, ensuring the conversation stays under token limits.
func truncateMessages(messages []Message, maxChars int) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Separate system message (always kept but may be trimmed)
	var system *Message
	var rest []Message
	for i := range messages {
		if messages[i].Role == RoleSystem {
			sys := messages[i]
			system = &sys
		} else {
			rest = append(rest, messages[i])
		}
	}

	// Trim system prompt if it alone exceeds a third of the budget
	if system != nil && len(system.Content) > maxChars/3 {
		system.Content = system.Content[:maxChars/3] + "\n...(truncated)"
	}

	budget := maxChars
	if system != nil {
		budget -= len(system.Content)
	}

	// Walk from most recent backward, accumulating messages that fit
	var kept []Message
	for i := len(rest) - 1; i >= 0; i-- {
		msgSize := len(rest[i].Content)
		if rest[i].ToolResult != nil {
			msgSize += len(rest[i].ToolResult.Content)
		}
		for _, tc := range rest[i].ToolCalls {
			msgSize += len(tc.Arguments)
		}
		// Truncate individual tool results that are too large
		if msgSize > budget/2 && rest[i].ToolResult != nil && len(rest[i].ToolResult.Content) > 1000 {
			trimmed := rest[i]
			tr := *trimmed.ToolResult
			tr.Content = tr.Content[:1000] + "\n...(truncated)"
			trimmed.ToolResult = &tr
			msgSize = len(trimmed.Content) + len(tr.Content)
			rest[i] = trimmed
		}

		budget -= msgSize
		if budget < 0 {
			break
		}
		kept = append([]Message{rest[i]}, kept...)
	}

	var result []Message
	if system != nil {
		result = append(result, *system)
	}
	result = append(result, kept...)
	return result
}

// ghAuthToken tries to get a GitHub token from the gh CLI.
func ghAuthToken() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// limitTools returns at most maxTools tools, prioritizing the most essential ones.
// It picks tools whose names contain common operations first.
func limitTools(tools []Tool, maxTools int) []Tool {
	if len(tools) <= maxTools {
		return tools
	}

	// Priority keywords — tools matching these are selected first
	priorities := []string{"list", "describe", "build", "deploy", "instance"}

	var selected []Tool
	used := map[int]bool{}

	for _, kw := range priorities {
		if len(selected) >= maxTools {
			break
		}
		for i, t := range tools {
			if used[i] {
				continue
			}
			if strings.Contains(strings.ToLower(t.Name), kw) {
				selected = append(selected, t)
				used[i] = true
				if len(selected) >= maxTools {
					break
				}
			}
		}
	}

	// Fill remaining slots with any unused tools
	for i, t := range tools {
		if len(selected) >= maxTools {
			break
		}
		if !used[i] {
			selected = append(selected, t)
		}
	}

	return selected
}
