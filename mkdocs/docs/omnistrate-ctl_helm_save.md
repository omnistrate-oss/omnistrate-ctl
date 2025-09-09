## omnistrate-ctl helm save

Save a Helm Chart for your service

### Synopsis

This command helps you save the templates for your helm charts.

```
omnistrate-ctl helm save chart --repo-name=[repo-name] --repo-url=[repo-url] --version=[version] --namespace=[namespace] --values-file=[values-file] [flags]
```

### Examples

```
# Install the Redis Operator Helm Chart
omctl helm save redis --repo-url=https://charts.bitnami.com/bitnami --version=20.0.1 --namespace=redis-operator
```

### Options

```
  -h, --help                 help for save
      --namespace string     Helm Chart namespace
      --repo-name string     Helm Chart repository name
      --repo-url string      Helm Chart repository URL
      --values-file string   Helm Chart values file containing custom values defined as a JSON
      --version string       Helm Chart version
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

- [omnistrate-ctl helm](omnistrate-ctl_helm.md) - Manage Helm Charts for your service
