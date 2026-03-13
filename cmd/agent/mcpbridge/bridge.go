package mcpbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent/provider"
)

// Bridge manages the connection to this utility's own MCP server.
type Bridge struct {
	mu      sync.Mutex
	client  *mcp.Client
	session *mcp.ClientSession
	tools   []provider.Tool
}

// New creates a new MCP bridge (not yet connected).
func New() *Bridge {
	return &Bridge{}
}

// Connect spawns `omnistrate-ctl mcp start` and establishes the MCP session.
func (b *Bridge) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	transport := &mcp.CommandTransport{
		Command: exec.CommandContext(ctx, execPath, "mcp", "start"),
	}

	b.client = mcp.NewClient(
		&mcp.Implementation{Name: "omnistrate-ctl-agent", Version: "1.0.0"},
		nil,
	)

	session, err := b.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	b.session = session

	// Discover tools
	if err := b.refreshTools(ctx); err != nil {
		session.Close()
		return fmt.Errorf("failed to list tools: %w", err)
	}

	return nil
}

// refreshTools fetches the tool list from the MCP server.
func (b *Bridge) refreshTools(ctx context.Context) error {
	result, err := b.session.ListTools(ctx, nil)
	if err != nil {
		return err
	}

	b.tools = make([]provider.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		b.tools = append(b.tools, provider.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return nil
}

// Tools returns the available MCP tools in provider.Tool format.
func (b *Bridge) Tools() []provider.Tool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tools
}

// CallTool invokes a tool on the MCP server and returns the text result.
func (b *Bridge) CallTool(ctx context.Context, name string, arguments string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.session == nil {
		return "", fmt.Errorf("MCP bridge not connected")
	}

	var args map[string]any
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	result, err := b.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	// Extract text from result content
	var text string
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}

	if result.IsError {
		return text, fmt.Errorf("tool returned error: %s", text)
	}

	return text, nil
}

// Close shuts down the MCP session and subprocess.
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.session != nil {
		return b.session.Close()
	}
	return nil
}

// IsConnected returns whether the bridge has an active session.
func (b *Bridge) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.session != nil
}
