## omnistrate-ctl deployment-cell apply

Apply pending changes to deployment cell

### Synopsis

Review and confirm the pending configuration changes for deployment cells.

Pending changes are activated and become the live configuration for those cells.
This command allows you to review the pending changes before applying them to 
ensure they are correct.

Examples:
  # Apply pending changes to specific deployment cell
  omnistrate-ctl deployment-cell apply -i cell-123 -s service-id -e env-id

  # Apply with confirmation prompt
  omnistrate-ctl deployment-cell apply -i cell-123 -s service-id -e env-id --confirm

  # Show pending changes without applying
  omnistrate-ctl deployment-cell apply -i cell-123 -s service-id -e env-id --dry-run

```
omnistrate-ctl deployment-cell apply [flags]
```

### Options

```
      --confirm                     Prompt for confirmation before applying changes
  -i, --deployment-cell-id string   Deployment cell ID (required)
      --dry-run                     Show pending changes without applying them
  -e, --environment-id string       Environment ID (required)
  -h, --help                        help for apply
  -s, --service-id string           Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

