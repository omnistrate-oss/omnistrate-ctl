## omnistrate-ctl deployment-cell amenities

Manage deployment cell amenities configuration

### Synopsis

Manage organization-level amenities configuration and deployment cell synchronization.

This command helps you:
- Initialize organization-level amenities configuration
- Update amenities configuration for target environments
- Check deployment cells for configuration drift
- Sync deployment cells with organization templates
- Apply pending configuration changes

Available operations:
  init        Initialize organization-level amenities configuration
  update      Update organization amenities configuration for target environment
  check-drift Check deployment cell for configuration drift
  sync        Sync deployment cell with organization+environment template
  apply       Apply pending changes to deployment cell
  status      Show amenities status for deployment cell

```
omnistrate-ctl deployment-cell amenities [operation] [flags]
```

### Options

```
  -h, --help   help for amenities
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells
* [omnistrate-ctl deployment-cell amenities apply](omnistrate-ctl_deployment-cell_amenities_apply.md)	 - Apply pending changes to deployment cell
* [omnistrate-ctl deployment-cell amenities check-drift](omnistrate-ctl_deployment-cell_amenities_check-drift.md)	 - Check deployment cell for configuration drift
* [omnistrate-ctl deployment-cell amenities init](omnistrate-ctl_deployment-cell_amenities_init.md)	 - Initialize organization-level amenities configuration
* [omnistrate-ctl deployment-cell amenities status](omnistrate-ctl_deployment-cell_amenities_status.md)	 - Show amenities status for deployment cell
* [omnistrate-ctl deployment-cell amenities sync](omnistrate-ctl_deployment-cell_amenities_sync.md)	 - Sync deployment cell with organization+environment template
* [omnistrate-ctl deployment-cell amenities update](omnistrate-ctl_deployment-cell_amenities_update.md)	 - Update organization amenities configuration for target environment

