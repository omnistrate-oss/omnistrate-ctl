## omnistrate-ctl mcp stream

Stream the MCP server over HTTP

### Synopsis

Start HTTP server to expose CLI commands to AI assistants

```text
omnistrate-ctl mcp stream [flags]
```

### Options

```text
  -h, --help               help for stream
      --host string        host to listen on
      --log-level string   Log level (debug, info, warn, error)
      --port int           port number to listen on (default 8080)
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl mcp](../omnistrate-ctl_mcp/) - MCP server management
