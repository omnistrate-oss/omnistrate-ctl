## omnistrate-ctl deployment-cell debug

Debug deployment cell amenities

### Synopsis

Debug deployment cell amenity template resolution, per-amenity status, rendered artifacts, and workflow logs. Use --output=json for automation.

```
omnistrate-ctl deployment-cell debug [flags]
```

### Examples

```
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id>
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output json
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output-dir ./debug-output
```

### Options

```
  -h, --help                help for debug
  -i, --id string           Deployment cell ID (required)
  -d, --output-dir string   Optional directory to export Helm logs
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

