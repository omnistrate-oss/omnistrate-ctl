## omnistrate-ctl mcp vscode list

List configured MCP servers

### Synopsis

List all MCP servers configured in VSCode

```
omnistrate-ctl mcp vscode list [flags]
```

### Options

```
      --config-path string   Path to VSCode config file
      --config-type string   Configuration type: 'workspace' or 'user' (default: user)
  -h, --help                 help for list
      --workspace            List workspace settings (.vscode/mcp.json) instead of user settings
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl mcp vscode](omnistrate-ctl_mcp_vscode.md)	 - Configure VSCode MCP servers

