## omnistrate-ctl deployment-cell sync

Sync deployment cell with organization+environment template

### Synopsis

Synchronize deployment cells to adopt the current organization+environment configuration.

Changes are placed in a pending state and not active until you approve them using 
the 'apply' command. This allows you to review the changes before they are applied 
to the deployment cell.

Examples:
  # Sync specific deployment cell with organization template
  omnistrate-ctl deployment-cell sync -i cell-123 -e production

  # Sync with confirmation prompt
  omnistrate-ctl deployment-cell sync -i cell-123 -e production --confirm

  # Sync all deployment cells that have drift
  omnistrate-ctl deployment-cell sync -e production --all --drift-only

```
omnistrate-ctl deployment-cell sync [flags]
```

### Options

```
      --all                         Sync all deployment cells in the organization
      --confirm                     Prompt for confirmation before syncing
  -i, --deployment-cell-id string   Deployment cell ID (required unless --all is used)
      --drift-only                  Only sync cells that have configuration drift (use with --all)
      --dry-run                     Show what would be synced without making changes
  -e, --environment string          Target environment (required)
  -h, --help                        help for sync
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

