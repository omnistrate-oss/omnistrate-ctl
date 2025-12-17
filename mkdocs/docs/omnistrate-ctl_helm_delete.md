## omnistrate-ctl helm delete

Delete a Helm package for your service

### Synopsis

This command helps you delete the templates for your helm packages.

```
omnistrate-ctl helm delete chart --version=[version] [flags]
```

### Examples

```
# Delete a Helm package
omnistrate-ctl helm delete redis --version=20.0.1
```

### Options

```
  -h, --help             help for delete
      --version string   Helm Chart version
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl helm](omnistrate-ctl_helm.md)	 - Manage Helm Charts for your service

