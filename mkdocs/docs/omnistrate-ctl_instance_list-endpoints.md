## omnistrate-ctl instance list-endpoints

List endpoints for a specific instance

### Synopsis

This command lists all additional endpoints and cluster endpoint for a specific instance by instance ID.

```
omnistrate-ctl instance list-endpoints [instance-id] [flags]
```

### Examples

```
# List endpoints for a specific instance
omctl instance list-endpoints instance-abcd1234
```

### Options

```
  -h, --help   help for list-endpoints
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](omnistrate-ctl_instance.md) - Manage Instance Deployments for your service
