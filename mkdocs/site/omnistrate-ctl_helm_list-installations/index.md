## omnistrate-ctl helm list-installations

List all Helm Packages and the Kubernetes clusters that they are installed on

### Synopsis

This command helps you list all the Helm Packages and the Kubernetes clusters that they are installed on.

```text
omnistrate-ctl helm list-installations --host-cluster-id=[host-cluster-id] [flags]
```

### Examples

```text
# List all Helm Packages and the Kubernetes clusters that they are installed on
omnistrate-ctl helm list-installations --host-cluster-id=[host-cluster-id]
```

### Options

```text
  -h, --help                     help for list-installations
      --host-cluster-id string   Host cluster ID
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl helm](../omnistrate-ctl_helm/) - Manage Helm Charts for your service
