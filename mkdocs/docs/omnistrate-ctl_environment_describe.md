## omnistrate-ctl environment describe

Describe a Service Environment

### Synopsis

This command helps you get details of a service environment from your service. You can find details like SaaS portal status, SaaS portal URL, and promote status, etc.

```
omnistrate-ctl environment describe [service-name] [environment-name] [flags]
```

### Examples

```
# Describe environment
omnistrate-ctl environment describe [service-name] [environment-name]

# Describe environment by ID instead of name
omnistrate-ctl environment describe --service-id=[service-id] --environment-id=[environment-id]
```

### Options

```
      --environment-id string   Environment ID. Required if environment name is not provided
  -h, --help                    help for describe
  -o, --output string           Output format. Only json is supported. (default "json")
      --service-id string       Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl environment](omnistrate-ctl_environment.md)	 - Manage Service Environments for your service

