## omnistrate-ctl environment create

Create a Service Environment

### Synopsis

This command helps you create a new environment for your service.

```text
omnistrate-ctl environment create [service-name] [environment-name] [flags]
```

### Examples

```text
# Create environment
omnistrate-ctl environment create [service-name] [environment-name] --type=[type] --source=[source]

# Create environment by ID instead of name
omnistrate-ctl environment create [environment-name] --service-id=[service-id] --type=[type] --source=[source]
```

### Options

```text
      --description string   Environment description
  -h, --help                 help for create
      --service-id string    Service ID. Required if service name is not provided
      --source string        Source environment name
      --type string          Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private'
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl environment](../omnistrate-ctl_environment/) - Manage Service Environments for your service
