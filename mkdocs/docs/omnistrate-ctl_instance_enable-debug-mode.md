## omnistrate-ctl instance enable-debug-mode

Enable debug mode for an instance deployment

### Synopsis

This command helps you enable debug mode for an instance deployment

```
omnistrate-ctl instance enable-debug-mode [instance-id] --resource-name [resource-name] --force [flags]
```

### Examples

```
# Enable debug mode for an instance deployment
omctl instance enable-debug-mode i-1234 --resource-name terraform --force
```

### Options

```
  -f, --force                  Force enable debug mode
  -h, --help                   help for enable-debug-mode
  -o, --output string          Output format. Only json is supported (default "json")
  -r, --resource-name string   Resource name
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](omnistrate-ctl_instance.md) - Manage Instance Deployments for your service
