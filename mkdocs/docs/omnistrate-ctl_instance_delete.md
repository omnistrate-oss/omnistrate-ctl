## omnistrate-ctl instance delete

Delete an instance deployment

### Synopsis

This command helps you delete an instance from your account.

```
omnistrate-ctl instance delete [instance-id] [flags]
```

### Examples

```
# Delete an instance deployment
omctl instance delete instance-abcd1234
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Pre-approve the deletion of the instance without prompting for confirmation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

