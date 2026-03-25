## omnistrate-ctl deployment-cell update-kubeconfig

Update kubeconfig for a deployment cell

### Synopsis

Update your local kubeconfig with the configuration for the specified deployment cell and set it as the default context.

```
omnistrate-ctl deployment-cell update-kubeconfig [deployment-cell-id] [flags]
```

### Examples

```
# Update kubeconfig for a deployment cell
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123

# Update kubeconfig with custom kubeconfig path
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --kubeconfig ~/.kube/my-config
```

### Options

```
      --customer-email string   Customer email to filter by (optional)
  -h, --help                    help for update-kubeconfig
      --kubeconfig string       Path to kubeconfig file (default: /tmp/kubeconfig)
      --role string             Access role for the kube context (optional, default: 'cluster-reader')
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

