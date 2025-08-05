## omnistrate-ctl deployment-cell apply-pending-changes

Apply pending configuration changes to a deployment cell

### Synopsis

Apply pending configuration changes to a deployment cell.

When you update a deployment cell's configuration template, the changes are initially 
stored as "pending changes" and do not take effect immediately. This command reviews 
and applies those pending changes to make them active in the deployment cell.

This is useful for:
- Activating configuration changes made through update-config-template
- Reviewing pending changes before they take effect
- Confirming configuration updates in a controlled manner

Examples:
  # Apply pending changes to specific deployment cell
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345

  # Apply without confirmation prompt
  omnistrate-ctl deployment-cell apply-pending-changes -i hc-12345 --force

```
omnistrate-ctl deployment-cell apply-pending-changes [flags]
```

### Options

```
      --force       Skip confirmation prompt and apply changes immediately
  -h, --help        help for apply-pending-changes
  -i, --id string   Deployment cell ID (format: hc-xxxxx)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

