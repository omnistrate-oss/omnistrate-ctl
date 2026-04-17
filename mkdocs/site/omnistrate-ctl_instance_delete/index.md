## omnistrate-ctl instance delete

Delete an instance deployment

### Synopsis

This command helps you delete an instance from your account.

```text
omnistrate-ctl instance delete [instance-id] [flags]
```

### Examples

```text
# Delete an instance deployment
omnistrate-ctl instance delete instance-abcd1234
```

### Options

```text
  -h, --help   help for delete
  -y, --yes    Pre-approve the deletion of the instance without prompting for confirmation
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](../omnistrate-ctl_instance/) - Manage Instance Deployments for your service
