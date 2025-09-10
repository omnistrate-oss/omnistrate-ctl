## omnistrate-ctl environment delete

Delete a Service Environment

### Synopsis

This command helps you delete an environment from your service.

```
omnistrate-ctl environment delete [service-name] [environment-name] [flags]
```

### Examples

```
# Delete environment
omctl environment delete [service-name] [environment-name]

# Delete environment by ID instead of name
omctl environment delete --service-id=[service-id] --environment-id=[environment-id]
```

### Options

```
      --environment-id string   Environment ID. Required if environment name is not provided
  -h, --help                    help for delete
      --service-id string       Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl environment](omnistrate-ctl_environment.md) - Manage Service Environments for your service
