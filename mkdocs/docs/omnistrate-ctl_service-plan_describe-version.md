## omnistrate-ctl service-plan describe-version

Describe a specific version of a Service Plan

### Synopsis

This command helps you get details of a specific version of a Service Plan for your service. You can get environment, enabled features, and resource configuration details for the version.

```
omnistrate-ctl service-plan describe-version [service-name] [plan-name] [flags]
```

### Examples

```
# Describe a service plan version
omctl service-plan describe-version [service-name] [plan-name] --version [version]

# Describe a service plan version by ID instead of name
omctl service-plan describe-version --service-id [service-id] --plan-id [plan-id] --version [version]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to describe the version in a specific environment
  -h, --help                 help for describe-version
  -o, --output string        Output format. Only json is supported (default "json")
      --plan-id string       Environment ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
  -v, --version string       Service plan version (latest|preferred|1.0 etc.)
```

### SEE ALSO

- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
