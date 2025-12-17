## omnistrate-ctl service-plan describe

Describe a Service Plan

### Synopsis

This command helps you get details of a Service Plan for your service.

```
omnistrate-ctl service-plan describe [service-name] [plan-name] [flags]
```

### Examples

```
# Describe service plan
omnistrate-ctl service-plan describe [service-name] [plan-name]

# Describe service plan by ID instead of name
omnistrate-ctl service-plan describe --service-id [service-id] --plan-id [plan-id]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment
  -h, --help                 help for describe
  -o, --output string        Output format. Only json is supported (default "json")
      --plan-id string       Environment ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md)	 - Manage Service Plans for your service

