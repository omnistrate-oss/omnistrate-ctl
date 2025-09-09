## omnistrate-ctl service-plan release

Release a Service Plan

### Synopsis

This command helps you release a Service Plan for your service. You can specify a custom release description and set the service plan as preferred if needed.

```
omnistrate-ctl service-plan release [service-name] [plan-name] [flags]
```

### Examples

```
# Release service plan by name
omctl service-plan release [service-name] [plan-name]

# Release service plan by ID
omctl service-plan release --service-id=[service-id] --plan-id=[plan-id]
```

### Options

```
      --dryrun                       Perform a dry run without making any changes
      --environment string           Environment name. Use this flag with service name and plan name to release the service plan in a specific environment
  -h, --help                         help for release
      --plan-id string               Plan ID. Required if plan name is not provided
      --release-as-preferred         Release the service plan as preferred
      --release-description string   Set custom release description for this release version
      --service-id string            Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
