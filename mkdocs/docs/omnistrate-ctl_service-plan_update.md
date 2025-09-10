## omnistrate-ctl service-plan update

Update Service Plan properties

### Synopsis

This command helps you update various properties of a Service Plan.
Currently supports updating the name of a specific version of a Service Plan.
The version name is used as the release description for the version.

```
omnistrate-ctl service-plan update [service-name] [plan-name] --version=[version] --name=[new-name] [flags]
```

### Examples

```
# Update service plan version name
omctl service-plan update [service-name] [plan-name] --version=[version] --name=[new-name]

# Update service plan version name by ID instead of name
omctl service-plan update --service-id=[service-id] --plan-id=[plan-id] --version=[version] --name=[new-name]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to update the version name in a specific environment
  -h, --help                 help for update
      --name string          Specify the new name for the version.
      --plan-id string       Plan ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
      --version string       Specify the version number to update the name for.
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
