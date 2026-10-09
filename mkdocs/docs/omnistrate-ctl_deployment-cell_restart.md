## omnistrate-ctl deployment-cell restart

Restart a deployment cell deployment

### Synopsis

Request a deployment restart for a deployment cell by its host cluster ID (hc-xxxxx). This command returns when the restart request is accepted; it does not wait for the deployment to become healthy.

```
omnistrate-ctl deployment-cell restart [deployment-cell-id] [flags]
```

### Examples

```
# Restart a deployment cell deployment
omnistrate-ctl deployment-cell restart hc-12345

# Request a restart with JSON output
omnistrate-ctl deployment-cell restart hc-12345 --output json
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

