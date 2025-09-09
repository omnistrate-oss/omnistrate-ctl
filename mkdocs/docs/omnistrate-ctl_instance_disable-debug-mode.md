## omnistrate-ctl instance disable-debug-mode

Disable debug mode for an instance deployment

### Synopsis

This command helps you disable debug mode for an instance deployment

```
omnistrate-ctl instance disable-debug-mode [instance-id] --resource-name [resource-name] --force [flags]
```

### Examples

```
# Disable debug mode for an instance deployment
omctl instance disable-debug-mode i-1234 --resource-name terraform --force
```

### Options

```
  -f, --force                  Force enable debug mode
  -h, --help                   help for disable-debug-mode
  -o, --output string          Output format. Only json is supported (default "json")
  -r, --resource-name string   Resource name
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

