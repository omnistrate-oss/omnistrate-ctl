## omnistrate-ctl service-plan delete

Delete a Service Plan

### Synopsis

This command helps you delete a Service Plan from your service.

```
omnistrate-ctl service-plan delete [service-name] [plan-name] [flags]
```

### Examples

```
# Delete service plan
omctl service-plan delete [service-name] [plan-name]

# Delete service plan by ID instead of name
omctl service-plan delete --service-id=[service-id] --plan-id=[plan-id]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to delete the service plan in a specific environment
  -h, --help                 help for delete
      --plan-id string       Plan ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
