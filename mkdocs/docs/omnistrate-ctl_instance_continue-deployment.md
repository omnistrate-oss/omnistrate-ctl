## omnistrate-ctl instance continue-deployment

Continue instance deployment

### Synopsis

This command helps you continue instance deployment.

```
omnistrate-ctl instance continue-deployment [instance-id] --resource-name <resource-name> --deployment-action <deployment-action> [flags]
```

### Examples

```
# Continue instance deployment
omnistrate-ctl instance continue-deployment instance-abcd1234 --resource-name my-terraform-deployment --deployment-action apply
```

### Options

```
  -e, --deployment-action string   Deployment action
  -h, --help                       help for continue-deployment
  -o, --output string              Output format. Only json is supported (default "json")
  -r, --resource-name string       Resource name
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

