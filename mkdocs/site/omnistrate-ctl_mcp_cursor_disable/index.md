## omnistrate-ctl mcp cursor disable

Remove server from Cursor config

### Synopsis

Remove this application from Cursor MCP servers

```text
omnistrate-ctl mcp cursor disable [flags]
```

### Options

```text
      --config-path string   Path to Cursor config file
  -h, --help                 help for disable
      --server-name string   Name of the MCP server to remove (default: derived from executable name)
      --workspace            Remove from workspace settings (.cursor/mcp.json) instead of user settings
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl mcp cursor](../omnistrate-ctl_mcp_cursor/) - Manage Cursor MCP servers
