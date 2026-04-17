## omnistrate-ctl instance trigger-backup

Trigger an automatic backup for your instance

### Synopsis

This command helps you trigger an automatic backup for your instance.

```text
omnistrate-ctl instance trigger-backup [instance-id] [flags]
```

### Examples

```text
# Trigger an automatic backup for an instance
omnistrate-ctl instance trigger-backup instance-abcd1234
```

### Options

```text
  -h, --help   help for trigger-backup
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](../omnistrate-ctl_instance/) - Manage Instance Deployments for your service
