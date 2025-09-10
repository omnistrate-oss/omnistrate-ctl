## omnistrate-ctl service-plan enable-feature

Enable feature for a service plan

### Synopsis

This command helps you enable & configure service plan features such as CUSTOM_TERRAFORM_POLICY.

```
omnistrate-ctl service-plan enable-feature [service-name] [plan-name] [flags]
```

### Examples

```
# Enable service plan feature 
omctl service-plan enable-feature [service-name] [plan-name] --feature [feature-name] --feature-configuration [feature-configuration]

# Enable service plan feature by ID instead of name and configure using file
omctl service-plan enable-feature --service-id [service-id] --plan-id [plan-id] --feature [feature-name] --feature-configuration-file /path/to/feature-config-file.json
```

### Options

```
      --environment string                  Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment
      --feature string                      Name / identifier of the feature to enable
      --feature-configuration string        Configuration of the feature
      --feature-configuration-file string   Json file containing feature configuration
  -h, --help                                help for enable-feature
      --plan-id string                      Environment ID. Required if plan name is not provided
      --service-id string                   Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md)	 - Manage Service Plans for your service

