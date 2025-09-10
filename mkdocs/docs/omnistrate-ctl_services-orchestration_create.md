## omnistrate-ctl services-orchestration create

Create a services orchestration deployment

### Synopsis

This command helps you create a services orchestration deployment, coordinating the creation of multiple services.

```
omnistrate-ctl services-orchestration create --dsl-file=[file-path] [flags]
```

### Examples

```
# Create a services orchestration deployment from a DSL file
omctl services-orchestration create --dsl-file /path/to/dsl.yaml
```

### Options

```
      --dsl-file string   Yaml file containing DSL for services orchestration deployment
  -h, --help              help for create
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl services-orchestration](omnistrate-ctl_services-orchestration.md)	 - Manage Services Orchestration Deployments across services

