## omnistrate-ctl deployment-cell restart

Restart a deployment cell

### Synopsis

Restart the deployment of a deployment cell by ID. This command returns once the restart has been requested.

```
omnistrate-ctl deployment-cell restart [deployment-cell-id] [flags]
```

### Examples

```
# Restart a deployment cell
omnistrate-ctl deployment-cell restart hc-12345678

# Request a restart with JSON output
omnistrate-ctl deployment-cell restart hc-12345678 --output json
```

### Options

```
  -h, --help   help for restart
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

