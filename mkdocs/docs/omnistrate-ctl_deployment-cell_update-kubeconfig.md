## omnistrate-ctl deployment-cell update-kubeconfig

Update kubeconfig for a deployment cell

### Synopsis

Create or reuse a deployment-cell access request, wait for approval when required, then update the local kubeconfig and set it as the default context.

```
omnistrate-ctl deployment-cell update-kubeconfig [deployment-cell-id] [flags]
```

### Examples

```
# Update kubeconfig for a deployment cell
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123

# Update kubeconfig with custom kubeconfig path
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --kubeconfig ~/.kube/my-config

# Request access when manual approval is enabled
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --reason "Investigate deployment health"
```

### Options

```
      --customer-email string   Customer email to filter by (optional)
  -h, --help                    help for update-kubeconfig
      --kubeconfig string       Path to kubeconfig file (default: /tmp/kubeconfig)
      --reason string           Reason for a new manual access request (required when manual approval is enabled)
      --role string             Access role for the kube context (optional, default: 'cluster-reader')
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

