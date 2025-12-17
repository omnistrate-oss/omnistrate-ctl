## omnistrate-ctl service-plan set-default

Set a Version of a Service Plan as Default(Preferred)

### Synopsis

This command helps you set a Version of a Service Plan as the default (preferred) version for your service.
By setting it as default, new instance deployments from your customers will be created with this version by default.

```
omnistrate-ctl service-plan set-default [service-name] [plan-name] --version=[version] [flags]
```

### Examples

```
# Set service plan as default
omnistrate-ctl service-plan set-default [service-name] [plan-name] --version=[version]

# Set  service plan as default by ID instead of name
omnistrate-ctl service-plan set-default --service-id=[service-id] --plan-id=[plan-id] --version=[version]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to set the default version in a specific environment
  -h, --help                 help for set-default
      --plan-id string       Plan ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
      --version string       Specify the version number to set the default to. Use 'latest' to set the latest version as default.
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md)	 - Manage Service Plans for your service

