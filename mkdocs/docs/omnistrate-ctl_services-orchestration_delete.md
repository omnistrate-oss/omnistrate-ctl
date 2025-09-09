## omnistrate-ctl services-orchestration delete

Delete a services orchestration deployment

### Synopsis

This command helps you delete a services orchestration deployment from your account.

```
omnistrate-ctl services-orchestration delete [services-orchestration-id] [flags]
```

### Examples

```
# Delete an services orchestration deployment
omctl services-orchestration delete so-abcd1234
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Pre-approve the deletion of the services orchestration deployment without prompting for confirmation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl services-orchestration](omnistrate-ctl_services-orchestration.md) - Manage Services Orchestration Deployments across services
