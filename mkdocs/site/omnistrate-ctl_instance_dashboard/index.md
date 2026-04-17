## omnistrate-ctl instance dashboard

Get Grafana dashboard access details for an instance

### Synopsis

This command opens an interactive dashboard TUI with customer and internal metrics views when metrics are enabled. Use -o json for raw metadata.

```text
omnistrate-ctl instance dashboard [instance-id] [flags]
```

### Examples

```text
# Open the interactive dashboard TUI for an instance
omnistrate-ctl instance dashboard instance-abcd1234

# Get raw dashboard metadata as JSON
omnistrate-ctl instance dashboard instance-abcd1234 -o json
```

### Options

```text
  -h, --help            help for dashboard
  -o, --output string   Output format (text|table|json). (default "text")
```

### Options inherited from parent commands

```text
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](../omnistrate-ctl_instance/) - Manage Instance Deployments for your service
