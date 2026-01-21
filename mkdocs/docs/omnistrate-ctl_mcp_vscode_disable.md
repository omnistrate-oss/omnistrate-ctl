## omnistrate-ctl mcp vscode disable

Remove server from VSCode config

### Synopsis

Remove this application as an MCP server from VSCode

```
omnistrate-ctl mcp vscode disable [flags]
```

### Options

```
      --config-path string   Path to VSCode config file
      --config-type string   Configuration type: 'workspace' or 'user' (default: user)
  -h, --help                 help for disable
      --server-name string   Name of the MCP server to disable (default: derived from executable name)
      --workspace            Remove from workspace settings (.vscode/mcp.json) instead of user settings
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl mcp vscode](omnistrate-ctl_mcp_vscode.md)	 - Configure VSCode MCP servers

