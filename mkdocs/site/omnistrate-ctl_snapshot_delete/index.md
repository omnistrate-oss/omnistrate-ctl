## omnistrate-ctl snapshot delete

Delete an instance snapshot

### Synopsis

This command helps you delete a specific snapshot.

```text
omnistrate-ctl snapshot delete [snapshot-id] --service-id <service-id> --environment-id <environment-id> [flags]
```

### Examples

```text
# Delete a specific snapshot
omnistrate-ctl snapshot delete snapshot-xyz789 --service-id service-abcd --environment-id env-1234
```

### Options

```text
      --environment-id string   The ID of the environment (required)
  -h, --help                    help for delete
      --service-id string       The ID of the service (required)
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl snapshot](../omnistrate-ctl_snapshot/) - Manage instance snapshots and backups
