## omnistrate-ctl deployment-cell amenities

Manage deployment cell amenities synchronization

### Synopsis

Manage deployment cell amenities synchronization with organization templates.

This command helps you:
- Check deployment cells for configuration drift against organization templates
- Sync deployment cells with organization+environment templates
- Apply pending configuration changes to deployment cells

These operations work with deployment cells to align them with organization-level
amenities templates. Use the 'organization amenities' commands to manage the
templates themselves.

Available operations:
  check-drift Check deployment cell for configuration drift
  sync        Sync deployment cell with organization+environment template
  apply       Apply pending changes to deployment cell

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
* [omnistrate-ctl deployment-cell amenities sync](omnistrate-ctl_deployment-cell_amenities_sync.md)	 - Sync deployment cell with organization+environment template

