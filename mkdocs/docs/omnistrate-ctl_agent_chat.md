## omnistrate-ctl agent chat

Start an interactive AI agent chat with Omnistrate tools

### Synopsis

Start an interactive TUI chat session with an AI provider that has access
to Omnistrate tools via the built-in MCP server.

Supported providers: claude, chatgpt, copilot

The AI agent can help you:
- Create and edit omnistrate-compose.yaml spec files
- Build and deploy services
- Manage instances and subscriptions
- Debug deployment issues

```
omnistrate-ctl agent chat [provider] [flags]
```

### Examples

```
  omnistrate-ctl agent chat claude
  omnistrate-ctl agent chat chatgpt
  omnistrate-ctl agent chat copilot
```

### Options

```
  -h, --help   help for chat
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl agent](omnistrate-ctl_agent.md)	 - Manage AI agent configurations and skills

