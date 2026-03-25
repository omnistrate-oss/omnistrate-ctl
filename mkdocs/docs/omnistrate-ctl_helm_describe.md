## omnistrate-ctl helm describe

Describe a Helm Chart for your service

### Synopsis

This command helps you describe the templates for your helm charts.

```
omnistrate-ctl helm describe chart --version=[version] [flags]
```

### Examples

```
# Describe the Redis Operator Helm Chart
omnistrate-ctl helm describe redis --version=20.0.1
```

### Options

```
  -h, --help             help for describe
      --version string   Helm Chart version
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl helm](omnistrate-ctl_helm.md)	 - Manage Helm Charts for your service

