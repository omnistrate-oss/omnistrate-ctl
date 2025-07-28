## omnistrate-ctl deployment-cell apply-pending-changes

Apply pending changes to deployment cell

### Synopsis

Review and confirm the pending configuration changes for deployment cells.

Pending changes are activated and become the live configuration for those cells.
This command allows you to review the pending changes before applying them to 
ensure they are correct.

Examples:
  # Apply pending changes to specific deployment cell
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 -s service-id

  # Apply without confirmation prompt
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 -s service-id --force

```
omnistrate-ctl deployment-cell apply-pending-changes [flags]
```

### Options

```
  -i, --deployment-cell-id string   Deployment cell ID (format: hc-xxxxx)
      --force                       Skip confirmation prompt and apply changes immediately
  -h, --help                        help for apply-pending-changes
  -s, --service-id string           Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

