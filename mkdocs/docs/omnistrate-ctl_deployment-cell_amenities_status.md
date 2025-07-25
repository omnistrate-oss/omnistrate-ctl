## omnistrate-ctl deployment-cell amenities status

Show amenities status for deployment cell

### Synopsis

Show the current amenities configuration status for a deployment cell.

This command displays:
- Current amenities configuration status
- Configuration drift information
- Pending changes information
- Last synchronization check time

Examples:
  # Show status for specific deployment cell
  omnistrate-ctl deployment-cell amenities status -i cell-123

  # Show detailed status including configuration details
  omnistrate-ctl deployment-cell amenities status -i cell-123 --detailed

  # Show status for multiple deployment cells
  omnistrate-ctl deployment-cell amenities status -i cell-123,cell-456,cell-789

```
omnistrate-ctl deployment-cell amenities status [flags]
```

### Options

```
  -i, --deployment-cell-id string   Deployment cell ID(s) - comma-separated for multiple cells (required)
      --detailed                    Show detailed configuration information
  -h, --help                        help for status
      --show-config                 Include current configuration in output
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell amenities](omnistrate-ctl_deployment-cell_amenities.md)	 - Manage deployment cell amenities configuration

