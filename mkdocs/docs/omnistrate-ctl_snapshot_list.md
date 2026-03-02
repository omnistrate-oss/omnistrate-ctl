## omnistrate-ctl snapshot list

List all snapshots for a service environment

### Synopsis

This command helps you list all snapshots available across all instances in a service environment.

```
omnistrate-ctl snapshot list --service-id <service-id> --environment-id <environment-id> [flags]
```

### Examples

```
# List all snapshots for a service environment
omnistrate-ctl snapshot list --service-id service-abcd --environment-id env-1234
```

### Options

```
      --environment-id string   The ID of the environment (required)
  -h, --help                    help for list
      --service-id string       The ID of the service (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl snapshot](omnistrate-ctl_snapshot.md)	 - Manage instance snapshots and backups

