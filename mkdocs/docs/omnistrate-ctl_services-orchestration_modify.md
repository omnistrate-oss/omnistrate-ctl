## omnistrate-ctl services-orchestration modify

DEPRECATED - Modify a services orchestration deployment

### Synopsis

DEPRECATED - This command helps you modify a services orchestration deployment, coordinating the modification of multiple services.

```
omnistrate-ctl services-orchestration modify [so-id] -dsl-file=[file-path] [flags]
```

### Examples

```
# Modify a services orchestration deployment from a DSL file
omctl services-orchestration modify so-abcd1234 --dsl-file /path/to/dsl.yaml
```

### Options

```
      --dsl-file string   Yaml file containing DSL for services orchestration deployment
  -h, --help              help for modify
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl services-orchestration](omnistrate-ctl_services-orchestration.md)	 - DEPRECATED - Manage Services Orchestration Deployments across services

