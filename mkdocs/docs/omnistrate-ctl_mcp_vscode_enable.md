## omnistrate-ctl mcp vscode enable

Add server to VSCode config

### Synopsis

Add this application as an MCP server in VSCode

```
omnistrate-ctl mcp vscode enable [flags]
```

### Options

```
      --config-path string   Path to VSCode config file
      --config-type string   Configuration type: 'workspace' or 'user' (default: user)
  -h, --help                 help for enable
      --log-level string     Log level (debug, info, warn, error)
      --server-name string   Name for the MCP server (default: derived from executable name)
      --workspace            Add to workspace settings (.vscode/mcp.json) instead of user settings
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl mcp vscode](omnistrate-ctl_mcp_vscode.md) - Configure VSCode MCP servers
