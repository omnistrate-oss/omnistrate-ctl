## omnistrate-ctl service-plan disable-feature

Disable feature for a service plan

### Synopsis

This command helps you disable active service plan feature.

```
omnistrate-ctl service-plan disable-feature [service-name] [plan-name] [flags]
```

### Examples

```
# Disable service plan feature 
omctl service-plan disable-feature [service-name] [plan-name] --feature [feature-name]

#  Disable service plan feature by ID instead of name
omctl service-plan enable-feature --service-id [service-id] --plan-id [plan-id] --feature [feature-name]
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment
      --feature string       Name / identifier of the feature to disable
  -h, --help                 help for disable-feature
      --plan-id string       Environment ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md)	 - Manage Service Plans for your service

