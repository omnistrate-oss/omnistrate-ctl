## omnistrate-ctl deployment-cell check-drift

Check deployment cell for configuration drift

### Synopsis

Review deployment cells to determine if their amenities configuration matches 
the latest organization template for the relevant environment.

Identify differences that may require alignment between the current deployment 
cell configuration and the organization's target configuration template.

Examples:
  # Check drift for a specific deployment cell
  omnistrate-ctl deployment-cell check-drift -i cell-123 -e production

  # Check drift for all deployment cells (if supported)
  omnistrate-ctl deployment-cell check-drift -e production --all

```
omnistrate-ctl deployment-cell check-drift [flags]
```

### Options

```
      --all                         Check drift for all deployment cells in the organization
  -i, --deployment-cell-id string   Deployment cell ID (required unless --all is used)
  -e, --environment string          Target environment (required)
  -h, --help                        help for check-drift
      --summary                     Show only summary of drift status
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

