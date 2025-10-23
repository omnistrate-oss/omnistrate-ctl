## omnistrate-ctl instance describe

Describe an instance deployment for your service

### Synopsis

This command helps you describe the instance for your service.

```
omnistrate-ctl instance describe [instance-id] [flags]
```

### Examples

```
# Describe an instance deployment
omctl instance describe instance-abcd1234

# Get compact deployment status information
omctl instance describe instance-abcd1234 --deployment-status

# Get deployment status for specific resource only  
omctl instance describe instance-abcd1234 --deployment-status --resource-key mydb
```

### Options

```
      --deployment-status     Return compact deployment status information instead of full instance details
      --detail                Include detailed information in the response
  -h, --help                  help for describe
  -o, --output string         Output format. Only json is supported (default "json")
      --resource-id string    Filter results by resource ID
      --resource-key string   Filter results by resource key
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

