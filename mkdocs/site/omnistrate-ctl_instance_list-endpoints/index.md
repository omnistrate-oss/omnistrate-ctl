## omnistrate-ctl instance list-endpoints

List endpoints for a specific instance

### Synopsis

This command lists all additional endpoints and cluster endpoint for a specific instance by instance ID.

```text
omnistrate-ctl instance list-endpoints [instance-id] [flags]
```

### Examples

```text
# List endpoints for a specific instance
omnistrate-ctl instance list-endpoints instance-abcd1234
```

### Options

```text
  -h, --help   help for list-endpoints
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](../omnistrate-ctl_instance/) - Manage Instance Deployments for your service
