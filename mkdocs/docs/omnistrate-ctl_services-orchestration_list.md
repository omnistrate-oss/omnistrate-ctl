## omnistrate-ctl services-orchestration list

List services orchestration deployments

### Synopsis

This command helps you list services orchestration deployments.

```
omnistrate-ctl services-orchestration list [flags]
```

### Examples

```
# List services orchestration deployments of the service postgres in the prod and dev environments
omctl services-orchestration list --environment-type=prod
```

### Options

```
      --environment-type string   Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private' (default "dev")
  -h, --help                      help for list
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl services-orchestration](omnistrate-ctl_services-orchestration.md) - Manage Services Orchestration Deployments across services
