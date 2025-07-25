## omnistrate-ctl deployment-cell update

Update deployment cell configuration

### Synopsis

Update deployment cell configuration including amenities settings.

This command allows you to update the configuration of a deployment cell,
including its amenities configuration such as logging, monitoring, and
security settings.

Examples:
  # Update deployment cell amenities configuration from YAML file
  omnistrate-ctl deployment-cell update -i cell-123 -s service-id -e env-id -f amenities.yaml

  # Update deployment cell amenities configuration interactively  
  omnistrate-ctl deployment-cell update -i cell-123 -s service-id -e env-id --interactive

```
omnistrate-ctl deployment-cell update [flags]
```

### Options

```
  -f, --config-file string          YAML file containing configuration to update
  -i, --deployment-cell-id string   Deployment cell ID (required)
  -e, --environment-id string       Environment ID (required)
  -h, --help                        help for update
      --interactive                 Use interactive mode to update configuration
      --merge                       Merge with existing configuration instead of replacing
  -s, --service-id string           Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

