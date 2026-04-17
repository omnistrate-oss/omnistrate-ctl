## omnistrate-ctl instance describe

Describe an instance deployment for your service

### Synopsis

This command helps you describe the instance for your service.

```text
omnistrate-ctl instance describe [instance-id] [flags]
```

### Examples

```text
# Describe an instance deployment
omnistrate-ctl instance describe instance-abcd1234

# Get compact deployment status information
omnistrate-ctl instance describe instance-abcd1234 --deployment-status

# Get deployment status for specific resource only  
omnistrate-ctl instance describe instance-abcd1234 --deployment-status --resource-key mydb
```

### Options

```text
      --deployment-status     Return compact deployment status information instead of full instance details
  -h, --help                  help for describe
  -o, --output string         Output format. Only json is supported (default "json")
      --resource-id string    Filter results by resource ID
      --resource-key string   Filter results by resource key
```

### Options inherited from parent commands

```text
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](../omnistrate-ctl_instance/) - Manage Instance Deployments for your service
