## omnistrate-ctl deployment-cell adopt

Adopt a deployment cell

### Synopsis

Adopt a deployment cell with the specified parameters.

```
omnistrate-ctl deployment-cell adopt [flags]
```

### Options

```
  -c, --cloud-provider string   Cloud provider name (required)
  -u, --customer-email string   Customer email to adopt the deployment cell for (optional)
  -d, --description string      Description for the deployment cell (default "Deployment cell adopted via CLI")
  -h, --help                    help for adopt
  -i, --id string               Deployment cell ID (required)
  -r, --region string           Region name (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md) - Manage Deployment Cells
