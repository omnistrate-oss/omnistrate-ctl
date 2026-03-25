## omnistrate-ctl environment promote

Promote a environment

### Synopsis

This command helps you promote a environment in your service.

```
omnistrate-ctl environment promote [service-name] [environment-name] [flags]
```

### Examples

```
# Promote environment
omnistrate-ctl environment promote [service-name] [environment-name]

# Promote environment by ID instead of name
omnistrate-ctl environment promote --service-id=[service-id] --environment-id=[environment-id]
```

### Options

```
      --environment-id string   Environment ID. Required if environment name is not provided
  -h, --help                    help for promote
      --service-id string       Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl environment](omnistrate-ctl_environment.md)	 - Manage Service Environments for your service

