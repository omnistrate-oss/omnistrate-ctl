## omnistrate-ctl instance describe-snapshot

Describe a specific instance snapshot

### Synopsis

This command helps you get detailed information about a specific snapshot for your instance.

```
omnistrate-ctl instance describe-snapshot [instance-id] [snapshot-id] [flags]
```

### Examples

```
# Describe a specific snapshot
omnistrate-ctl instance describe-snapshot instance-abcd1234 snapshot-xyz789
```

### Options

```
  -h, --help   help for describe-snapshot
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

