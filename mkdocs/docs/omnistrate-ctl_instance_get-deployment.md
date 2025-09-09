## omnistrate-ctl instance get-deployment

Get the deployment entity metadata of the instance

### Synopsis

This command helps you get the deployment entity metadata of the instance.

```
omnistrate-ctl instance get-deployment [instance-id] --resource-name <resource-name> --output-path <output-path> [flags]
```

### Examples

```
  # Get the deployment entity metadata of the instance
	  omctl instance get-deployment instance-abcd1234 --resource-name my-terraform-deployment --output-path /tmp
```

### Options

```
  -h, --help                   help for get-deployment
  -o, --output string          Output format. Only json is supported (default "json")
  -p, --output-path string     Output path
  -r, --resource-name string   Resource name
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](omnistrate-ctl_instance.md) - Manage Instance Deployments for your service
